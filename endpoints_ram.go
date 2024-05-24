package main

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// params
	ramQueryParamFrequency string = "frequency"
)

type RAMEndpoints struct {
	lock      *sync.Mutex
	leaks     [][]byte
	stopFuncs []func()
	workerID  int
	workers   *sync.WaitGroup
}

func NewRAMEndpoints() *RAMEndpoints {
	return &RAMEndpoints{
		lock:    &sync.Mutex{},
		workers: &sync.WaitGroup{},
	}
}

func (e *RAMEndpoints) Increase(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	l.Debug("parsing query variables")
	// parsing size
	size := size1MiB
	sizeString := r.URL.Query().Get(queryParamSize)
	if len(sizeString) > 0 {
		if !positiveIntegerRegex.MatchString(sizeString) {
			w.WriteHeader(http.StatusBadRequest)
			errorString := "size doesn't match regex: " + positiveIntegerRegex.String()
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		}
		size, _ = strconv.Atoi(sizeString) // can't fail thanks to the regexp
	}

	// increasing memory usage
	l.Infof("increasing memory usage with %s (%d Bytes)", SizeToHumanReadable(float64(size)), size)
	memStats := e.leak(size)
	l.Info(memStats)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(memStats))
}

func (e *RAMEndpoints) Decrease(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	l.Debug("parsing query variables")
	// parsing size
	size := size1MiB
	sizeString := r.URL.Query().Get(queryParamSize)
	if len(sizeString) > 0 {
		if !positiveIntegerRegex.MatchString(sizeString) {
			w.WriteHeader(http.StatusBadRequest)
			errorString := "size doesn't match regex: " + positiveIntegerRegex.String()
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		}
		size, _ = strconv.Atoi(sizeString) // can't fail thanks to the regexp
	}

	// decreasing memory usage
	l.Infof("decreasing memory usage of %s (%d Bytes)", SizeToHumanReadable(float64(size)), size)
	e.lock.Lock()
	remainingSize := size
	for i := len(e.leaks) - 1; i >= 0; i-- { // starting from the end
		if len(e.leaks[i]) < remainingSize {
			remainingSize = remainingSize - len(e.leaks[i])
			e.leaks = e.leaks[:i]
		} else {
			diffSize := len(e.leaks[i]) - remainingSize
			e.leaks[i] = e.leaks[i][:diffSize]
			remainingSize = 0
			break
		}
	}
	e.lock.Unlock()

	// display memory stats
	memStats := e.gc()
	l.Info(memStats)

	// not enough memory was released
	if remainingSize > 0 {
		errorString := fmt.Sprintf("server was not able to release all %s asked (%d Bytes), but only %s (%d Bytes)", SizeToHumanReadable(float64(size)), size, SizeToHumanReadable(float64(size-remainingSize)), size-remainingSize)
		l.Warn(errorString)
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte(errorString + "\n" + memStats))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(memStats))
}

func (e *RAMEndpoints) Leak(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	l.Debug("parsing query variables")
	// parsing size
	size := size1MiB
	sizeString := r.URL.Query().Get(queryParamSize)
	if len(sizeString) > 0 {
		if !positiveIntegerRegex.MatchString(sizeString) {
			w.WriteHeader(http.StatusBadRequest)
			errorString := "size doesn't match regex: " + positiveIntegerRegex.String()
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		}
		size, _ = strconv.Atoi(sizeString) // can't fail thanks to the regexp
	}

	// parsing frequency
	leakFrequencyString := r.URL.Query().Get(ramQueryParamFrequency)
	var err error
	var leakFrequency time.Duration
	if len(leakFrequencyString) > 0 {
		leakFrequency, err = time.ParseDuration(leakFrequencyString)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			errorString := "leak frequency is incorrect: " + err.Error()
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		} else if leakFrequency < 0 {
			w.WriteHeader(http.StatusBadRequest)
			errorString := "leak frequency is inferior to zero: " + leakFrequency.String()
			w.Write([]byte(errorString))
			l.Warn(errorString)
			return
		}
	}

	// stop
	stopCh := make(chan int)
	e.lock.Lock()
	e.stopFuncs = append(e.stopFuncs, func() { stopCh <- 0 })
	e.workerID++
	e.lock.Unlock()

	// leak worker
	e.workers.Add(1)
	go func() {
		defer e.workers.Done()

		// logger
		l := log.WithField("leak_worker_id", e.workerID)
		if leakFrequency > 0 {
			l = l.WithField("frequency", leakFrequency.String())
		}

		// worker loop
		for {
			select {
			case <-stopCh:
				return

			default:
				l.Info(e.leak(size))
				if leakFrequency > 0 {
					time.Sleep(leakFrequency)
				}
			}
		}
	}()
	l.Infof("starting memory leak with frequency of %s/%s", SizeToHumanReadable(float64(size)), leakFrequency.String())
	w.WriteHeader(http.StatusOK)
}

func (e *RAMEndpoints) leak(size int) string {
	leak := make([]byte, size)
	for i := range leak {
		leak[i] = 0x00
	}
	e.lock.Lock()
	e.leaks = append(e.leaks, leak)
	e.lock.Unlock()

	// display memory stats
	return e.memoryStats()
}

func (e *RAMEndpoints) Reset(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	// stopping leak workers
	e.lock.Lock()
	for _, stopFunc := range e.stopFuncs {
		go stopFunc()
	}
	e.stopFuncs = []func(){}
	e.lock.Unlock()
	e.workers.Wait()

	// clearing leaks
	e.lock.Lock()
	e.leaks = [][]byte{}
	e.lock.Unlock()

	// display memory stats
	memStats := e.gc()
	l.Info(memStats)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(memStats))
}

func (e *RAMEndpoints) Status(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	memStats := e.memoryStats()
	l.Info(memStats)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(memStats))
}

func (e RAMEndpoints) gc() string {
	runtime.GC()
	return e.memoryStats()
}
func (RAMEndpoints) memoryStats() string {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	return fmt.Sprintf("memory status: Alloc: %s", SizeToHumanReadable(float64(memory.Alloc)))
}
