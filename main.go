package main

import (
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func init() {
	// logger
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})
}

func main() {
	log.Info("starting Integration Toolbox WebServer")

	// configuration
	config := DefaultConfig()
	var err error
	if err = config.OverwriteFromEnv(); err != nil {
		log.WithError(err).Fatal("failed to parse configuration from environment variables")
	}
	if err = config.Validate(); err != nil {
		log.WithError(err).Fatal("invalid configuration")
	}

	// debug log level
	if config.Debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug log level enabled")
		config.Log() // config is display with the "debug" log level
	}

	// basic auth
	basicAuthMiddleware := NewBasicAuthMiddleWare(config.BasicAuthUsername, config.BasicAuthPassword)

	// routing endpoints
	http.HandleFunc("/crash", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(crash)))))
	http.HandleFunc("/download", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(download)))))
	http.HandleFunc("/echo", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(echo)))))
	http.HandleFunc("/echo/form", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(echoForm)))))
	http.HandleFunc("/echo/raw", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(LogMiddleware(echoRaw))))
	http.HandleFunc("/ping", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(ping)))))
	http.HandleFunc("/request", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(request)))))
	http.HandleFunc("/sleep", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(sleep)))))
	http.HandleFunc("/status_code", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(statusCode)))))
	http.HandleFunc("/tcp", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(tcp)))))
	http.HandleFunc("/upload", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(upload)))))
	// databases
	databaseEndpoints := NewDatabaseEndpoints()
	http.HandleFunc("/database/connect", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(databaseEndpoints.Connect)))))
	http.HandleFunc("/database/query", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(databaseEndpoints.Query)))))
	// CPU
	cpuEndpoints := NewCPUEndpoints()
	http.HandleFunc("/cpu/load", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(cpuEndpoints.Load)))))
	http.HandleFunc("/cpu/reset", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(cpuEndpoints.Reset)))))
	// RAM
	ramEndpoints := NewRAMEndpoints()
	http.HandleFunc("/ram/increase", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(ramEndpoints.Increase)))))
	http.HandleFunc("/ram/decrease", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(ramEndpoints.Decrease)))))
	http.HandleFunc("/ram/leak", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(ramEndpoints.Leak)))))
	http.HandleFunc("/ram/reset", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(ramEndpoints.Reset)))))
	http.HandleFunc("/ram/status", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(HeadersMiddleWare(LogMiddleware(ramEndpoints.Status)))))
	// ui
	http.HandleFunc("/ui/", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(http.StripPrefix("/ui/", http.FileServer(http.Dir("./ui"))).ServeHTTP))) // trailing '/' in the path is needed
	// static folder
	if len(config.StaticFolder) > 0 {
		http.HandleFunc("/static/", LogRequestMiddleWare(basicAuthMiddleware.MiddleWare(http.StripPrefix("/static/", http.FileServer(http.Dir(config.StaticFolder))).ServeHTTP))) // trailing '/' in the path is needed
	}
	// monitoring
	monitoringEndpoints := NewMonitoringEndpoints(config.MonitoringConfig)
	http.HandleFunc("/started", LogRequestMiddleWare(HeadersMiddleWare(LogMiddleware(monitoringEndpoints.Startup))))
	http.HandleFunc("/alive", LogRequestMiddleWare(HeadersMiddleWare(LogMiddleware(monitoringEndpoints.Liveness))))
	http.HandleFunc("/ready", LogRequestMiddleWare(HeadersMiddleWare(LogMiddleware(monitoringEndpoints.Readiness))))

	// HTTP server
	log.Infof("server is now listening on: %s", config.ListenOn)
	if len(config.TLSCert) > 0 {
		err = http.ListenAndServeTLS(config.ListenOn, config.TLSCert, config.TLSKey, nil)
	} else {
		err = http.ListenAndServe(config.ListenOn, nil)
	}
	if errors.Is(err, http.ErrServerClosed) {
		log.Debug("HTTP server closed")
	} else if err != nil {
		log.WithError(err).Fatal("failed to start the HTTP server")
	}
	log.Info("Integration Toolbox WebServer stopped")
}
