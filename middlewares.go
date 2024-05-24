package main

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type BasicAuthMiddleWare struct {
	username string
	password string
}

func NewBasicAuthMiddleWare(username, password string) BasicAuthMiddleWare {
	return BasicAuthMiddleWare{
		username: username,
		password: password,
	}
}

func (mw BasicAuthMiddleWare) MiddleWare(downstream http.HandlerFunc) http.HandlerFunc {
	// no basic auth
	if len(mw.username) == 0 {
		return downstream
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok && (username == mw.username) && (password == mw.password) {
			downstream(w, r)
			return
		}

		// need to authenticate
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func LogMiddleware(downstream func(*log.Entry, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := log.WithFields(log.Fields{
			LogHTTPPath:     r.URL.Path,
			LogHTTPMethod:   r.Method,
			LogHTTPClientIP: r.RemoteAddr,
		})
		if len(r.URL.RawQuery) > 0 {
			l = l.WithField(LogHTTPQuery, r.URL.RawQuery)
		}
		downstream(l, w, r)
	}
}

func HeadersMiddleWare(downstream http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		downstream(w, r)
	}
}

func LogRequestMiddleWare(downstream http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startDate := time.Now()
		rwi := NewResponseWriterInspector(w)
		downstream(rwi, r)
		processingDuration := time.Since(startDate)
		path := r.URL.Path
		if len(r.URL.Query().Encode()) > 0 {
			path += "?" + r.URL.Query().Encode()
		}
		log.Infof("request processed client:%s request:\"%s %s %s\" status_code:%d length:%d timing_ns:%d", r.RemoteAddr, r.Method, path, r.Proto, rwi.GetStatus(), rwi.GetAnswerLength(), processingDuration.Nanoseconds())
	}
}

// ResponseWriterInspector defines an interface that give the ability to inpect
// HTTP answers, the HTTP status code and the answer length.
type ResponseWriterInspector interface {
	http.ResponseWriter
	// GetStatus returns the HTTP status code set in the request.
	// If the status hasn't been set, the default value (0) should
	// be returned.
	GetStatus() int
	// GetAnswerLength return the length of the request (in bytes).
	// The length is only for the answer payload, and excludes every
	// headers. An empty payload will return a length of 0.
	GetAnswerLength() int64
}

type responseWriterInspector struct {
	w            http.ResponseWriter
	status       int
	answerLength int64
}

// NewResponseWriterInspector creates basic inspector. The inspector doesn't do much more than
// the basic interface contract.
func NewResponseWriterInspector(w http.ResponseWriter) ResponseWriterInspector {
	return &responseWriterInspector{
		w: w,
	}
}

func (w *responseWriterInspector) Header() http.Header {
	return w.w.Header()
}
func (w *responseWriterInspector) Write(b []byte) (int, error) {
	length, err := w.w.Write(b)
	w.answerLength += int64(length)
	return length, err
}
func (w *responseWriterInspector) WriteHeader(statusCode int) {
	w.w.WriteHeader(statusCode)
	w.status = statusCode
}

func (w *responseWriterInspector) GetStatus() int {
	return w.status
}

func (w *responseWriterInspector) GetAnswerLength() int64 {
	return w.answerLength
}
