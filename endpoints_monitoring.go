package main

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	monitoringQueryParamFail       string = "fail"
	monitoringQueryParamFailNumber string = "nb_failures"
	monitoringQueryParamDelay      string = "delay"
)

func NewMonitoringEndpoints(config MonitoringConfig) *MonitoringEndpoints {
	return &MonitoringEndpoints{
		startup:   newMonitoringEndpoints(config.Startup),
		liveness:  newMonitoringEndpoints(config.Liveness),
		readiness: newMonitoringEndpoints(config.Readiness),
	}
}

type MonitoringEndpoints struct {
	startup   *monitoringEndpoints
	liveness  *monitoringEndpoints
	readiness *monitoringEndpoints
}

func (e *MonitoringEndpoints) Startup(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	e.endpoint(e.startup, l, w, r)
}

func (e *MonitoringEndpoints) Liveness(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	e.endpoint(e.liveness, l, w, r)
}

func (e *MonitoringEndpoints) Readiness(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	e.endpoint(e.readiness, l, w, r)
}

func (*MonitoringEndpoints) endpoint(e *monitoringEndpoints, l *log.Entry, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		e.Endpoint(l, w, r)

	case http.MethodPost:
		e.Configure(l, w, r)

	default:
		l.Errorf("invalid method %s for endpoint", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func newMonitoringEndpoints(config MonitoringEndpointConfig) *monitoringEndpoints {
	return &monitoringEndpoints{
		lock:       &sync.Mutex{},
		okStatus:   config.StatusOk,
		failStatus: config.StatusError,
		fail:       config.Fail,
		failNb:     config.FailNb,
		delay:      config.Delay,
	}
}

type monitoringEndpoints struct {
	lock       *sync.Mutex
	okStatus   int
	failStatus int
	fail       bool
	failNb     int
	delay      time.Duration
}

func (e *monitoringEndpoints) Endpoint(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	// delay
	if e.delay > 0 { // not sure this improves a lot
		l.Infof("endpoint will sleep for: %s", e.delay.String())
		time.Sleep(e.delay)
	}

	// lock
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.fail {
		errorString := "endpoint set to fail"
		w.WriteHeader(e.failStatus)
		w.Write([]byte(errorString))
		l.Info(errorString)
		return
	} else if e.failNb > 0 {
		e.failNb--
		errorString := fmt.Sprintf("%d failure(s) remaining", e.failNb)
		w.WriteHeader(e.failStatus)
		w.Write([]byte(errorString))
		l.Info(errorString)
		return
	}
	w.WriteHeader(e.okStatus)
}

func (e *monitoringEndpoints) Configure(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	// query params
	failString := r.URL.Query().Get(monitoringQueryParamFail)
	failNbString := r.URL.Query().Get(monitoringQueryParamFailNumber)
	delayString := r.URL.Query().Get(monitoringQueryParamDelay)

	// nothing set
	if len(failString) == 0 && len(failNbString) == 0 && len(delayString) == 0 {
		l.Warnf("no query param set")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("please set a configuration to change with any of following query params: %s, %s, %s", monitoringQueryParamFail, monitoringQueryParamFailNumber, monitoringQueryParamDelay)))
		return
	}

	// lock
	e.lock.Lock()
	defer e.lock.Unlock()

	// failed set
	if len(failString) > 0 {
		if fail, err := strconv.ParseBool(failString); err != nil {
			errorString := fmt.Sprintf("the %s query param is not a boolean", monitoringQueryParamFail)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.WithError(err).Warn(errorString)
			return
		} else {
			e.fail = fail
			l.Infof("endpoint set to fail: %t", fail)
		}
	}

	// number of failed set
	if len(failNbString) > 0 {
		if failNb, err := strconv.Atoi(failNbString); err != nil {
			errorString := fmt.Sprintf("failed to parse %s query param to integer", monitoringQueryParamFailNumber)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.WithError(err).Warn(errorString)
			return
		} else if failNb < 0 {
			errorString := fmt.Sprintf("the %s query param is inferior to 0 (%d)", monitoringQueryParamFailNumber, failNb)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		} else {
			e.failNb = failNb
			l.Infof("endpoint set to fail %d time(s)", failNb)
		}
	}

	// delay set
	if len(delayString) > 0 {
		if delay, err := time.ParseDuration(delayString); err != nil {
			errorString := fmt.Sprintf("failed to parse %s query param to duration", monitoringQueryParamDelay)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.WithError(err).Warn(errorString)
			return
		} else if delay < 0 {
			errorString := fmt.Sprintf("the %s query param is inferior to 0 (%s)", monitoringQueryParamFailNumber, delay.String())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		} else {
			e.delay = delay
			l.Infof("endpoint set to delay answers for %s", delay.String())
		}
	}

	// ok
	w.WriteHeader(http.StatusOK)
}
