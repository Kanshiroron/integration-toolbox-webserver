package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"syscall"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	log "github.com/sirupsen/logrus"
)

const (
	// query params
	queryParamCode     string = "code"
	queryParamCount    string = "count"
	queryParamDuration string = "duration"
	queryParamHeaders  string = "headers"
	queryParamHost     string = "host"
	queryParamSize     string = "size"
	queryParamTimeout  string = "timeout"

	size1MiB           int           = 1024 * 1024 // 1MiB
	defaultPingTimeout time.Duration = 20 * time.Second
)

var (
	positiveIntegerRegex *regexp.Regexp = regexp.MustCompile("^[0-9]+$")
	statusCodeRegex      *regexp.Regexp = regexp.MustCompile("^[1-5][0-9]{2}$")
)

/* CRASH */
func crash(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	// default exit code
	exitCode := 1

	// parsing exit code
	var err error
	exitCodeString := r.URL.Query().Get(queryParamCode)
	if len(exitCodeString) > 0 {
		if exitCode, err = strconv.Atoi(exitCodeString); err != nil {
			errorString := fmt.Sprintf("query param %s is not an integer", queryParamCode)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString + ": " + err.Error()))
			l.WithError(err).Warn(errorString)
			return
		} else if exitCode < 0 {
			errorString := fmt.Sprintf("query param %s is inferior to 0 (value: %d)", queryParamCode, exitCode)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		}
	}
	// parsing timeout
	timeout := time.Second
	timeoutString := r.URL.Query().Get(queryParamTimeout)
	if len(timeoutString) > 0 {
		timeout, err = time.ParseDuration(timeoutString)
		if err != nil {
			errorString := "timeout is incorrect: " + err.Error()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		} else if timeout < 0 {
			errorString := "timeout is inferior to zero: " + timeout.String()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	l.Infof("server will crash in %s with exit code: %d", timeout.String(), exitCode)

	// crash
	go func(exitCode int) {
		time.Sleep(timeout)
		syscall.Exit(exitCode)
	}(exitCode)
}

/* DOWNLOAD */
func download(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	// parsing size
	size := size1MiB
	sizeString := r.URL.Query().Get(queryParamSize)
	if len(sizeString) > 0 {
		if !positiveIntegerRegex.MatchString(sizeString) {
			errorString := "size doesn't match regex: " + positiveIntegerRegex.String()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		}
		size, _ = strconv.Atoi(sizeString) // can't fail thanks to the regexp
	}

	l.Infof("starting download of %s (%d Bytes)", SizeToHumanReadable(float64(size)), size)
	defer l.Info("download finished")
	w.Header().Add("Content-Length", sizeString)
	w.WriteHeader(http.StatusOK)

	// populating download data
	downloadData := make([]byte, 10*1024*1024) // 10KB
	for i := range downloadData {
		downloadData[i] = 0x00
	}

	// sending data
	dataSize := len(downloadData)
	for size > dataSize {
		w.Write(downloadData)
		size = size - dataSize
	}
	w.Write(downloadData[:size])
}

/* ECHO */
func echo(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	l.Debug("parsing query variables")
	// headers query
	var err error
	var echoHeaders bool
	headersQueryVar := r.URL.Query().Get(queryParamHeaders)
	if len(headersQueryVar) > 0 {
		if echoHeaders, err = strconv.ParseBool(headersQueryVar); err != nil {
			errorString := "failed to parse headers query var"
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString + ": " + err.Error()))
			l.WithError(err).Warn(errorString)
			return
		}
	}

	w.WriteHeader(http.StatusOK)

	// write headers
	if echoHeaders {
		writeRequestHeaders(w, r)
	}

	// body
	w.Write([]byte("--- BODY\n"))
	if writeBody(l, w, r.Body) == 0 {
		w.Write([]byte(">>>>> EMPTY REQUEST BODY <<<<<"))
	}
}

func echoForm(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	// do we have a POST method? (mandatory according to RFC)
	if r.Method != http.MethodPost {
		l.Warnf("only the POST method is allowed for posting forms, according to RFC 1867 (%s used)", r.Method)
	}

	l.Debug("parsing query variables")
	// headers query
	var err error
	var displayHeaders bool
	headersQueryVar := r.URL.Query().Get(queryParamHeaders)
	if len(headersQueryVar) > 0 {
		if displayHeaders, err = strconv.ParseBool(headersQueryVar); err != nil {
			errorString := "failed to parse headers query var"
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString + ": " + err.Error()))
			l.WithError(err).Warn(errorString)
			return
		}
	}

	// parse form
	if err = r.ParseMultipartForm(MaxFormSize); err != nil {
		errorString := "failed to parse form"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warnf(errorString)
		return
	}

	// write headers
	w.WriteHeader(http.StatusOK)
	if displayHeaders {
		writeRequestHeaders(w, r)
	}

	// form content
	w.Write([]byte("--- FORM\n"))
	// type Form = url.Values = map[string][]string
	if len(r.Form) == 0 {
		w.Write([]byte(">>>>> EMPTY FORM <<<<<"))
	} else {
		for name, values := range r.Form {
			for _, value := range values {
				w.Write([]byte(name + ": " + value + "\n"))
			}
		}
	}
}

func echoRaw(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	l.Debug("parsing query variables")
	// headers query
	var err error
	var echoHeaders bool
	headersQueryVar := r.URL.Query().Get(queryParamHeaders)
	if len(headersQueryVar) > 0 {
		if echoHeaders, err = strconv.ParseBool(headersQueryVar); err != nil {
			errorString := "failed to parse headers query var"
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString + ": " + err.Error()))
			l.WithError(err).Warn(errorString)
			return
		}
	}

	// write headers
	if echoHeaders {
		// type http.Header = map[string][]string
		for headerName, headerValues := range r.Header {
			for _, headerValue := range headerValues {
				w.Header().Add(headerName, headerValue)
			}
		}
	}

	w.WriteHeader(http.StatusOK)

	// body
	writeBody(l, w, r.Body)
}

func writeRequestHeaders(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("--- REQUEST HEADERS\n"))
	w.Write([]byte(r.Method + " " + r.URL.Path + " " + r.Proto + "\n"))
	w.Write([]byte("Host: " + r.Host + "\n"))
	writeHeaders(w, r.Header)
}

func writeHeaders(w http.ResponseWriter, headers http.Header) {
	// type Header map[string][]string
	for name, values := range headers {
		for _, value := range values {
			w.Write([]byte(name + ": " + value + "\n"))
		}
	}
	w.Write([]byte("\n"))
}

func writeBody(l *log.Entry, w http.ResponseWriter, body io.ReadCloser) (size int) {
	// buffers
	tmp := make([]byte, 1024) // 1KB read buffer
	// sequential read
	for {
		// partial read
		n, err := body.Read(tmp)
		if n > 0 {
			w.Write(tmp[:n])
			size += n
		}

		// EOF or error
		if err != nil {
			if err != io.EOF {
				l.WithError(err).Error("failed to copy request body to answer")
				return
			}
			break
		}
	}
	return size
}

/* PING */
func ping(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	l.Debug("parsing query variables")
	// parsing ip
	host := r.URL.Query().Get(queryParamHost)
	if len(host) == 0 {
		errorString := "empty hostname or IP"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString))
		l.Warn(errorString)
		return
	}
	// parsing count
	pingCount := 3
	var err error
	pingCountString := r.URL.Query().Get(queryParamCount)
	if len(pingCountString) > 0 {
		if !positiveIntegerRegex.MatchString(pingCountString) {
			errorString := "count doesn't match regex: " + positiveIntegerRegex.String()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		}
		pingCount, _ = strconv.Atoi(pingCountString) // can't fail thanks to the regexp
		if pingCount < 1 {
			errorString := "count can't be inferior to 1"
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		}
	}

	// creating pinger
	l.Debug("creating new pinger")
	pinger, err := probing.NewPinger(host)
	if err != nil {
		errorString := "failed to created pinger"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}
	l.Debug("setting privileges on the OS")
	pinger.SetPrivileged(true) // mandatory on windows

	// ping
	pinger.Count = pingCount
	pinger.Timeout = defaultPingTimeout
	l.Infof("sending %d pings to %s", pingCount, host)
	if err = pinger.Run(); err != nil {
		errorString := fmt.Sprintf("failed to ping host %s", host)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}

	// sending result
	pingStats := pinger.Statistics()
	pingResults := fmt.Sprintf("ping results: sent: %d, received: %d (%.2f%%), min timing: %s, max timing: %s, average timing: %s",
		pingStats.PacketsSent,
		pingStats.PacketsRecv,
		(1.0-(float32(pingStats.PacketsSent)-float32(pingStats.PacketsRecv))/float32(pingStats.PacketsSent))*100,
		pingStats.MinRtt.String(),
		pingStats.MaxRtt.String(),
		pingStats.AvgRtt.String(),
	)
	l.Info(pingResults)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(pingResults))
}

/* SLEEP */
func sleep(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	l.Debug("parsing query variables")
	// parsing duration
	duration := time.Second
	var err error
	sleepDurationString := r.URL.Query().Get(queryParamDuration)
	if len(sleepDurationString) > 0 {
		duration, err = time.ParseDuration(sleepDurationString)
		if err != nil {
			errorString := "sleep duration is incorrect"
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString + ": " + err.Error()))
			l.WithError(err).Warn(errorString)
			return
		}
	}
	// parsing status code
	status := http.StatusOK
	statusCodeString := r.URL.Query().Get(queryParamCode)
	if len(statusCodeString) > 0 {
		if !statusCodeRegex.MatchString(statusCodeString) {
			errorString := "status code doesn't match regex: " + positiveIntegerRegex.String()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		}
		status, _ = strconv.Atoi(statusCodeString) // can't fail thanks to the regexp
	}

	// sleep
	l.Infof("endpoint is going to sleep for: %s", duration.String())
	time.Sleep(duration)

	// sending back status code
	w.WriteHeader(status)
}

/* STATUS */
func statusCode(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	l.Debug("parsing query variables")
	// parsing status code
	statusCodeString := r.URL.Query().Get(queryParamCode)
	if !statusCodeRegex.MatchString(statusCodeString) {
		errorString := "status code doesn't match regex: " + positiveIntegerRegex.String()
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString))
		l.Warn(errorString)
		return
	}

	// sending back status code
	status, _ := strconv.Atoi(statusCodeString) // can't fail thanks to the regexp
	w.WriteHeader(status)
}

/* UPLOAD */
func upload(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	l.Info("start uploading")

	// buffers
	tmp := make([]byte, 1024) // 1KB temp buffer
	size := 0                 // size of the body request
	for {
		// partial read
		n, err := r.Body.Read(tmp)
		size += n

		// EOF or error
		if err != nil {
			if err != io.EOF {
				l.WithError(err).Error("failed to read request body")
				return
			}
			break
		}
	}

	answerText := fmt.Sprintf("upload done, %s sent (%d Bytes)", SizeToHumanReadable(float64(size)), size)
	l.Info(answerText)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(answerText))
}
