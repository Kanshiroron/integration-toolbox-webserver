package main

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	// form data keys
	tcpFormDataHost              string = "host"
	tcpFormDataConnectionTimeout string = "connection_timeout"
	tcpFormDataEchoBody          string = "echo_body"
	tcpFormDataEchoBodySize      string = "echo_body_size"

	// defauts
	tcpDefaultConnectTimeout time.Duration = 20 * time.Second
)

func tcp(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	// request config
	config, err := parseTCPConfigFromFormData(l, r)
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

	// opening connection
	var conn net.Conn
	dialer := net.Dialer{
		Timeout: config.connectionTimeout,
	}
	if !config.tlsConfig.Enabled { // without TLS
		l.Infof("opening TCP connection to %s", config.host)
		conn, err = dialer.Dial("tcp", config.host)
	} else { // with TLS
		l.Infof("opening TCP connection to %s over TLS", config.host)
		var tlsConfig *tls.Config
		tlsConfig, err = config.tlsConfig.GetTLSConfig(l)
		if err != nil {
			errorString := "invalid TLS configuration"
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString + ": " + err.Error()))
			l.WithError(err).Warn(errorString)
			return
		}
		conn, err = tls.DialWithDialer(&dialer, "tcp", config.host, tlsConfig)
	}
	if err != nil {
		errorString := "failed to estabish connection with remote server"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}
	defer conn.Close()

	w.WriteHeader(http.StatusOK)

	// echo body
	if config.echoBody {
		l.Debug("reading answer body to echo")
		buff := make([]byte, config.echoBodySize)
		conn.SetDeadline(time.Now().Add(config.connectionTimeout))
		n, err := conn.Read(buff)
		if err != nil && (err != io.EOF) && !errors.Is(err, os.ErrDeadlineExceeded) {
			errorString := "failed to read response from server"
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString + ": " + err.Error()))
			l.WithError(err).Warn(errorString)
			return
		}
		if n > 0 {
			w.Write(buff[:n])
		} else {
			w.Write([]byte(">>>>> EMPTY ANSWER FROM SERVER <<<<<"))
		}
	}
}

type tcpConfig struct {
	host              string
	tlsConfig         TLSConfig
	connectionTimeout time.Duration
	echoBody          bool
	echoBodySize      int
}

func parseTCPConfigFromFormData(l *log.Entry, r *http.Request) (c tcpConfig, err error) {
	l.Debug("parsing tcp configuration")
	// parse form
	if err = r.ParseMultipartForm(MaxFormSize); err != nil {
		return
	}

	// host
	c.host = strings.TrimSpace(r.FormValue(tcpFormDataHost))

	// connection timeout
	connectionTimeoutString := strings.TrimSpace(r.FormValue(requestFormDataConnectionTimeout))
	if len(connectionTimeoutString) > 0 {
		if c.connectionTimeout, err = time.ParseDuration(connectionTimeoutString); err != nil {
			return c, errors.WithMessage(err, "failed to parse connection timeout to Golang duration")
		}
	} else {
		c.connectionTimeout = tcpDefaultConnectTimeout
	}
	// echo body
	echoBodyString := strings.TrimSpace(r.FormValue(requestFormDataEchoBody))
	if len(echoBodyString) > 0 {
		if c.echoBody, err = strconv.ParseBool(echoBodyString); err != nil {
			return c, errors.WithMessage(err, "failed to parse echo body to boolean")
		}
	}
	if c.echoBody {
		// echo body size
		echoBodySize := strings.TrimSpace(r.FormValue(tcpFormDataEchoBodySize))
		if len(echoBodySize) > 0 {
			if c.echoBodySize, err = strconv.Atoi(echoBodySize); err != nil {
				return c, errors.WithMessage(err, "failed to parse echo body size to int")
			}
		} else {
			c.echoBodySize = size1MiB
		}
	}
	// tls
	c.tlsConfig, err = ParseTLSConfigFromFormData(l, r)
	return
}

func (c tcpConfig) Validate() error {
	// host
	if len(c.host) == 0 {
		return errors.New("tcp host not set")
	}
	if strings.Contains(c.host, "://") {
		return errors.New("the host must not containt the scheme (i.e.: tcp:// or equivalent)")
	}
	// connection timeout
	if c.connectionTimeout < 0 {
		return errors.New("connection timeout can't be negative")
	}
	// echo body size
	if c.echoBody && c.echoBodySize < 0 {
		return errors.New("echo body size can't be negative")
	}
	// tls
	return c.tlsConfig.Validate()
}
