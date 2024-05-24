package main

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	// environment
	envBasicAuthUsername string = "BASIC_AUTH_USERNAME"
	envBasicAuthPassword string = "BASIC_AUTH_PASSWORD"
	envDebug             string = "DEBUG"
	envListenOn          string = "LISTEN_ON"
	envMaxFormSize       string = "MAX_FORM_SIZE"
	envServerTLSCert     string = "SERVER_TLS_FILE"
	envServerTLSCertKey  string = "SERVER_TLS_KEY"
	envStaticFolder      string = "STATIC_FOLDER"
	envTempFolder        string = "TEMP_FOLDER"
	// monitoring environment prefixes
	envMonitoringPrefixStartup   string = "STARTUP_PROBE_"
	envMonitoringPrefixLiveness  string = "LIVENESS_PROBE_"
	envMonitoringPrefixReadiness string = "READINESS_PROBE_"
	// monitoring environments
	envMonitoringStatusOk    string = "STATUS_OK"
	envMonitoringStatusError string = "STATUS_ERROR"
	envMonitoringFail        string = "FAIL"
	envMonitoringFailNumber  string = "FAIL_NB"
	envMonitoringDelay       string = "DELAY"

	// defaults
	defaultListenOn string = ":8080"
	//monitoring
	defaultMonitoringStatusOk    int = http.StatusOK
	defaultMonitoringStatusError int = http.StatusInternalServerError
)

type Config struct {
	BasicAuthUsername string
	BasicAuthPassword string
	Debug             bool
	ListenOn          string
	TLSCert           string
	TLSKey            string
	StaticFolder      string

	MonitoringConfig MonitoringConfig
}

var (
	TempFolderPath string = "/tmp/integration-toolbox-webserver"
	MaxFormSize    int64  = 100 * 1024 // 100KiB
)

func DefaultConfig() Config {
	return Config{
		ListenOn:         defaultListenOn,
		MonitoringConfig: DefaultMonitoringConfig(),
	}
}

func (c *Config) OverwriteFromEnv() (err error) {
	// basic auth username
	if authUsername, found := syscall.Getenv(envBasicAuthUsername); found {
		c.BasicAuthUsername = authUsername
	}
	// basic auth password
	if authPassword, found := syscall.Getenv(envBasicAuthPassword); found {
		c.BasicAuthPassword = authPassword
	}
	// debug
	if debugLogString, found := syscall.Getenv(envDebug); found {
		if c.Debug, err = strconv.ParseBool(debugLogString); err != nil {
			return errors.WithMessagef(err, "failed to parse the debug value to boolean (env variable: %s)", envDebug)
		}
	}
	// http server listen on
	if serverListenString, found := syscall.Getenv(envListenOn); found {
		c.ListenOn = serverListenString
	}
	// max form size
	if maxFormSizeString, found := syscall.Getenv(envMaxFormSize); found {
		maxFormSize, err := strconv.Atoi(maxFormSizeString)
		if err != nil {
			return errors.WithMessagef(err, "failed to parse the %s value to int", envMaxFormSize)
		}
		MaxFormSize = int64(maxFormSize)
	}
	// tls cert
	if tlsCert, found := syscall.Getenv(envServerTLSCert); found {
		c.TLSCert = tlsCert
	}
	// tls key
	if tlsKey, found := syscall.Getenv(envServerTLSCertKey); found {
		c.TLSKey = tlsKey
	}
	// static folder
	if staticFolder, found := syscall.Getenv(envStaticFolder); found {
		c.StaticFolder = staticFolder
	}
	// temp folder
	if tempFolder, found := syscall.Getenv(envTempFolder); found {
		TempFolderPath = tempFolder
	}

	// monitoring config
	return c.MonitoringConfig.OverwriteFromEnv()
}

func (c Config) Validate() (err error) {
	// basic auth
	if (len(c.BasicAuthUsername) > 0) != (len(c.BasicAuthPassword) > 0) {
		return errors.Errorf("both %s and %s environment variables must be set or empty", envBasicAuthUsername, envBasicAuthPassword)
	}
	// static folder
	if len(c.StaticFolder) > 0 {
		if info, err := os.Stat(c.StaticFolder); err != nil {
			return errors.WithMessagef(err, "failed to verify if the static folder exists at location: %q", c.StaticFolder)
		} else if !info.IsDir() {
			return errors.Errorf("the static folder location %q is not a directory", c.StaticFolder)
		}
		if _, err := os.ReadDir(c.StaticFolder); err != nil {
			return errors.WithMessagef(err, "failed to list files in the static fodler (%s), please verify access rights", c.StaticFolder)
		}
	}
	// tls certificates
	if (len(c.TLSCert) > 0) != (len(c.TLSKey) > 0) {
		return errors.Errorf("both %s and %s environment variables must be set or empty", envServerTLSCert, envServerTLSCertKey)
	} else if len(c.TLSCert) > 0 {
		for _, file := range []string{c.TLSCert, c.TLSKey} {
			fileInfo, err := os.Stat(TempFolderPath)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return errors.WithMessagef(err, "failed to check if file %s exists", file)
				} else {
					return errors.Errorf("file %s does not exists", file)
				}
			}
			if fileInfo.IsDir() {
				return errors.Errorf("file %s is a directory", file)
			}
		}
	}
	// temp folder
	if len(TempFolderPath) == 0 {
		return errors.New("temporary folder path is not set")
	} else if !strings.HasPrefix(TempFolderPath, "/") {
		return errors.New("temporary folder path must be absolute")
	}
	if _, err = os.Stat(TempFolderPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return errors.WithMessagef(err, "failed to check if temp folder %s exists", TempFolderPath)
		}
		if err = os.MkdirAll(TempFolderPath, 0750); err != nil {
			return errors.WithMessagef(err, "failed to create temp folder %s", TempFolderPath)
		}
	}

	// monitoring
	return c.MonitoringConfig.Validate()
}

func (c Config) Log() {
	if len(c.BasicAuthUsername) > 0 {
		log.Debugf("CONFIG :: basic auth username: %s", c.BasicAuthUsername)
		log.Debugf("CONFIG :: basic auth password: %s", c.BasicAuthPassword)
	} else {
		log.Debug("CONFIG :: basic auth is not configured")
	}
	log.Debugf("CONFIG :: maximum form size: %s (%d bytes)", SizeToHumanReadable(float64(MaxFormSize)), MaxFormSize)
	if len(c.TLSCert) > 0 {
		log.Debugf("CONFIG :: server TLS certificate file: %s", c.TLSCert)
		log.Debugf("CONFIG :: server TLS certificate key file: %s", c.TLSKey)
	} else {
		log.Debug("CONFIG :: server TLS is not configured")
	}
	if len(c.StaticFolder) > 0 {
		log.Debugf("CONFIG :: static folder: %s", c.StaticFolder)
	} else {
		log.Debug("CONFIG :: no static folder set")
	}
	log.Debugf("CONFIG :: temp folder: %s", TempFolderPath)
	c.MonitoringConfig.Log()
}

type MonitoringConfig struct {
	Startup   MonitoringEndpointConfig
	Liveness  MonitoringEndpointConfig
	Readiness MonitoringEndpointConfig
}

func DefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		Startup:   DefaultMonitoringEndpointConfig(),
		Liveness:  DefaultMonitoringEndpointConfig(),
		Readiness: DefaultMonitoringEndpointConfig(),
	}
}

func (c *MonitoringConfig) OverwriteFromEnv() (err error) {
	// startup
	if err = c.Startup.OverwriteFromEnv(envMonitoringPrefixStartup); err != nil {
		return
	}
	// liveness
	if err = c.Liveness.OverwriteFromEnv(envMonitoringPrefixLiveness); err != nil {
		return
	}
	// readiness
	return c.Readiness.OverwriteFromEnv(envMonitoringPrefixReadiness)
}

func (c MonitoringConfig) Validate() (err error) {
	// startup
	if err = c.Startup.Validate(); err != nil {
		return errors.WithMessage(err, "invalid startup configuration")
	}
	// liveness
	if err = c.Liveness.Validate(); err != nil {
		return errors.WithMessage(err, "invalid liveness configuration")
	}
	// readiness
	if err = c.Readiness.Validate(); err != nil {
		return errors.WithMessage(err, "invalid readiness configuration")
	}
	return nil
}

func (c MonitoringConfig) Log() {
	c.Startup.Log("startup")
	c.Liveness.Log("liveness")
	c.Readiness.Log("readyness")
}

type MonitoringEndpointConfig struct {
	StatusOk    int
	StatusError int
	Fail        bool
	FailNb      int
	Delay       time.Duration
}

func DefaultMonitoringEndpointConfig() MonitoringEndpointConfig {
	return MonitoringEndpointConfig{
		StatusOk:    defaultMonitoringStatusOk,
		StatusError: defaultMonitoringStatusError,
	}
}

func (c *MonitoringEndpointConfig) OverwriteFromEnv(prefix string) (err error) {
	// status ok
	envStatusOk := prefix + envMonitoringStatusOk
	if statusOkString, found := syscall.Getenv(envStatusOk); found {
		if !statusCodeRegex.MatchString(statusOkString) {
			return errors.Errorf("the %s env variable status code does not match regexp: %s (value: %s)", envStatusOk, statusCodeRegex.String(), statusOkString)
		}
		c.StatusOk, _ = strconv.Atoi(statusOkString) // error checked with regexp
	}
	// status error
	envStatusError := prefix + envMonitoringStatusError
	if statusErrorString, found := syscall.Getenv(envStatusError); found {
		if !statusCodeRegex.MatchString(statusErrorString) {
			return errors.Errorf("the %s env variable status code does not match regexp: %s (value: %s)", envStatusError, statusCodeRegex.String(), statusErrorString)
		}
		c.StatusError, _ = strconv.Atoi(statusErrorString) // error checked with regexp
	}
	// fail
	envFail := prefix + envMonitoringFail
	if failString, found := syscall.Getenv(envFail); found {
		if c.Fail, err = strconv.ParseBool(failString); err != nil {
			return errors.Errorf("failed to parse boolean from %s env variable (value: %s)", envFail, failString)
		}
	}
	// fail number
	envFailNb := prefix + envMonitoringFailNumber
	if failNbString, found := syscall.Getenv(envFailNb); found {
		if c.FailNb, err = strconv.Atoi(failNbString); err != nil {
			return errors.Errorf("failed to parse integer from %s env variable (value: %s)", envFailNb, failNbString)
		}
	}
	// delay
	envDelay := prefix + envMonitoringDelay
	if delayString, found := syscall.Getenv(envDelay); found {
		if c.Delay, err = time.ParseDuration(delayString); err != nil {
			return errors.Errorf("failed to parse Golang duration from %s env variable (value: %s)\nfor more information, please visit: https://pkg.go.dev/time#ParseDuration", envDelay, delayString)
		}
	}

	return
}

func (c MonitoringEndpointConfig) Validate() error {
	// fail number
	if c.FailNb < 0 {
		return errors.Errorf("number of failures inferior to zero (value: %d)", c.FailNb)
	}
	// delay
	if c.Delay < 0 {
		return errors.Errorf("delay inferior to zero (value: %s)", c.Delay.String())
	}

	return nil
}

func (c MonitoringEndpointConfig) Log(prefix string) {
	log.Debugf("CONFIG :: %s probe status code ok: %d", prefix, c.StatusOk)
	log.Debugf("CONFIG :: %s probe status code fail: %d", prefix, c.StatusError)
	log.Debugf("CONFIG :: %s probe set to fail: %t", prefix, c.Fail)
	log.Debugf("CONFIG :: %s probe number of failures: %d", prefix, c.FailNb)
	log.Debugf("CONFIG :: %s probe delay: %s", prefix, c.Delay.String())
}
