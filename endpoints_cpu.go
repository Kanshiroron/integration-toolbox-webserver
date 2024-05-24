package main

import (
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// params
	queryParamNbTheads string = "nb_threads"
)

type CPUEndpoints struct {
	lock      *sync.Mutex
	stopFuncs []func()
	workers   *sync.WaitGroup
}

func NewCPUEndpoints() *CPUEndpoints {
	return &CPUEndpoints{
		lock:    &sync.Mutex{},
		workers: &sync.WaitGroup{},
	}
}

func (e *CPUEndpoints) Load(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	l.Debug("parsing query variables")
	// parsing size
	nbThreads := 1
	nbThreadsString := r.URL.Query().Get(queryParamNbTheads)
	if len(nbThreadsString) > 0 {
		if !positiveIntegerRegex.MatchString(nbThreadsString) {
			errorString := "number of threads doesn't match regex: " + positiveIntegerRegex.String()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		}
		nbThreads, _ = strconv.Atoi(nbThreadsString) // can't fail thanks to the regexp
	}
	if nbThreads == 0 {
		// setting it to the number of cores in the system
		nbThreads = runtime.NumCPU()
	}

	// parsing timeout
	var timeout time.Duration
	var err error
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

	// starting workers
	e.lock.Lock()
	defer e.lock.Unlock()
	e.workers.Add(nbThreads)
	l.Infof("starting %d load workers, with timeout of %s", nbThreads, timeout.String())
	for range nbThreads {
		stopCh := make(chan int)
		e.stopFuncs = append(e.stopFuncs, func() { stopCh <- 0 })
		go func(timeout time.Duration, stopChan <-chan int) {
			defer e.workers.Done()

			// worker loop
			for {
				select {
				case <-stopCh:
					return

				default:
					if timeout > 0 {
						time.Sleep(timeout)
					}
				}
			}
		}(timeout, stopCh)
	}
	w.WriteHeader(http.StatusOK)
}

func (e *CPUEndpoints) Reset(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	// stopping load workers
	e.lock.Lock()
	for _, stopFunc := range e.stopFuncs {
		go stopFunc()
	}
	e.stopFuncs = []func(){}
	e.lock.Unlock()
	e.workers.Wait()
	l.Info("load workers stopped")

	w.WriteHeader(http.StatusOK)
}
