package main

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	// form data keys
	requestFormDataURL               string = "url"
	requestFormDataMethod            string = "method"
	requestFormDataProxyURL          string = "proxy_url"
	requestFormDataProxyUsername     string = "proxy_username"
	requestFormDataProxyPassword     string = "proxy_password"
	requestFormDataConnectionTimeout string = "connection_timeout"
	requestFormDataEchoHeaders       string = "echo_headers"
	requestFormDataEchoBody          string = "echo_body"

	// defauts
	requestDefaultConnectTimeout time.Duration = 20 * time.Second
)

var (
	tlsSchemes   []string = []string{"https://", "wss://"}
	validSchemes []string = append(tlsSchemes, "http://", "ws://")
)

func request(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	// request config
	config, err := parseRequestConfigFromFormData(l, r)
	if err != nil {
		errorString := "failed to parse request config"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}
	if err = config.Validate(); err != nil {
		errorString := "invalid request config"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}

	// url
	l.Debug("parsing URL")
	u, err := url.Parse(config.url)
	if err != nil {
		errorString := "invalid URL"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}

	// tls config
	transport := &http.Transport{}
	l.Debug("generating TLS configuration (if any)")
	transport.TLSClientConfig, err = config.tlsConfig.GetTLSConfig(l)
	if err != nil {
		errorString := "invalid TLS configugration"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}

	// proxy config
	l.Debug("generating proxy configuration (if any)")
	proxyURL, err := config.GetProxyURL()
	if err != nil {
		errorString := "invalid proxy configugration"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	} else if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
		l.Debug("proxy configuration attached")
	}

	// making request
	if strings.HasPrefix(config.url, "http") { // regular HTTP request
		request := &http.Request{
			Method: config.method,
			URL:    u,
		}
		cli := http.Client{
			Transport: transport,
			Timeout:   config.connectionTimeout,
		}
		l.Infof("starting HTTP request to %s", config.url)
		answer, err := cli.Do(request)
		if err != nil {
			errorString := "failed to perform the request"
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString + ": " + err.Error()))
			l.WithError(err).Warn(errorString)
			return
		}

		w.WriteHeader(http.StatusOK)
		// echo headers
		if config.echoHeaders {
			w.Write([]byte("--- ANSWER HEADERS\n"))
			writeHeaders(w, answer.Header)
		}
		// echo body
		if config.echoBody {
			w.Write([]byte("--- ANSWER BODY\n"))
			writeBody(l, w, answer.Body)
		}
	} else { // websocket connection
		ctx, ctxCancelFunc := context.WithTimeout(context.Background(), config.connectionTimeout)
		defer ctxCancelFunc()

		// opening websocket connection
		l.Infof("starting websocket request to %s", config.url)
		ws, answer, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
		if err != nil {
			errorString := "failed to open websocket connection"
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString + ": " + err.Error()))
			l.WithError(err).Warn(errorString)
			return
		}
		ws.Close()

		w.WriteHeader(http.StatusOK)
		// echo headers
		if config.echoHeaders {
			w.Write([]byte("--- ANSWER HEADERS\n"))
			writeHeaders(w, answer.Header)
		}
		// echo body
		if config.echoBody {
			w.Write([]byte("--- ANSWER BODY\n"))
			writeBody(l, w, answer.Body)
		}
	}
	l.Infof("request to %s correctly run", config.url)
}

type requestConfig struct {
	method            string
	url               string
	tlsConfig         TLSConfig
	proxyURL          string
	proxyUsername     string
	proxyPassword     string
	connectionTimeout time.Duration
	echoHeaders       bool
	echoBody          bool
}

func parseRequestConfigFromFormData(l *log.Entry, r *http.Request) (c requestConfig, err error) {
	l.Debug("parsing request configuration")
	// parse form
	if err = r.ParseMultipartForm(MaxFormSize); err != nil {
		return
	}

	c.method = strings.ToUpper(strings.TrimSpace(r.FormValue(requestFormDataMethod)))
	if len(c.method) == 0 {
		c.method = http.MethodGet
	}
	c.url = strings.TrimSpace(r.FormValue(requestFormDataURL))
	c.proxyURL = strings.TrimSpace(r.FormValue(requestFormDataProxyURL))
	c.proxyUsername = strings.TrimSpace(r.FormValue(requestFormDataProxyUsername))
	c.proxyPassword = strings.TrimSpace(r.FormValue(requestFormDataProxyPassword))

	// connection timeout
	connectionTimeoutString := strings.TrimSpace(r.FormValue(requestFormDataConnectionTimeout))
	if len(connectionTimeoutString) > 0 {
		if c.connectionTimeout, err = time.ParseDuration(connectionTimeoutString); err != nil {
			return c, errors.WithMessage(err, "failed to parse connection timeout to Golang duration")
		}
	} else {
		c.connectionTimeout = requestDefaultConnectTimeout
	}
	// echo headers
	echoHeadersString := strings.TrimSpace(r.FormValue(requestFormDataEchoHeaders))
	if len(echoHeadersString) > 0 {
		if c.echoHeaders, err = strconv.ParseBool(echoHeadersString); err != nil {
			return c, errors.WithMessage(err, "failed to parse echo headers to boolean")
		}
	}
	// echo body
	echoBodyString := strings.TrimSpace(r.FormValue(requestFormDataEchoBody))
	if len(echoBodyString) > 0 {
		if c.echoBody, err = strconv.ParseBool(echoBodyString); err != nil {
			return c, errors.WithMessage(err, "failed to parse echo body to boolean")
		}
	}
	// tls
	r.Form.Add(tlsFormDataTLSEnabled, strconv.FormatBool(c.tlsEnabled())) // needed to be manually added since the option does not exist in this endpoint (determined from scheme)
	c.tlsConfig, err = ParseTLSConfigFromFormData(l, r)
	return
}

func (c requestConfig) Validate() error {
	// url
	if len(c.url) == 0 {
		return errors.New("request URL not set")
	}
	if !strings.Contains(c.url, "://") {
		return errors.New("the URL must contain the scheme (i.e.: http(s)://)")
	}
	// validate scheme
	var found bool
	for _, scheme := range validSchemes {
		if strings.HasPrefix(c.url, scheme) {
			found = true
		}
	}
	if !found {
		return errors.Errorf("the URL must start with one of following schemes: %s", strings.Join(validSchemes, ", "))
	}
	// method
	switch c.method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
	default:
		return errors.Errorf("unknown method: %q, must be one of: %s, %s, %s, %s, %s, %s, %s, %s, %s", c.method, http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace)
	}
	// tls
	return c.tlsConfig.Validate()
}

func (c requestConfig) GetProxyURL() (*url.URL, error) {
	if len(c.proxyURL) == 0 {
		return nil, nil
	}

	// parsing proxy URL
	proxyURL, err := url.Parse(c.proxyURL)
	if err != nil {
		return nil, err
	}

	// authentication
	if len(c.proxyUsername) > 0 {
		if len(c.proxyPassword) > 0 {
			proxyURL.User = url.UserPassword(c.proxyUsername, c.proxyPassword)
		} else {
			proxyURL.User = url.User(c.proxyUsername)
		}
	}

	return proxyURL, nil
}

func (c requestConfig) tlsEnabled() bool {
	for _, scheme := range tlsSchemes {
		if strings.HasPrefix(c.url, scheme) {
			return true
		}
	}
	return false
}
