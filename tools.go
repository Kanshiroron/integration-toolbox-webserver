package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	sizePowers []string = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
)

// SizeToHumanReadable returns a human readable of a binary size.
// This works with both positive and negative sizes (if ever needed).
// It has a decimal precision of 2 digits.
func SizeToHumanReadable(size float64) string {
	prefix := ""
	if size < 0 {
		prefix = "-"
		size = -size
	}
	return prefix + sizeToHumanReadable(size, 2, 0)
}

func sizeToHumanReadable(size float64, decimals, power int) string {
	if (size >= 1024) && (power < len(sizePowers)-1) {
		power++
		return sizeToHumanReadable(size/1024, decimals, power)
	}
	return fmt.Sprintf("%."+strconv.Itoa(decimals)+"f %s", size, sizePowers[power])
}

var (
	uuidLetters = []rune("0123456789abcdef")
)

// GenerateUUID generates a random UUID and return it in the string format.
func GenerateUUID() string {
	b := make([]rune, 32)
	for i := range b {
		b[i] = uuidLetters[rand.Intn(len(uuidLetters))]
	}
	return string(b[:8]) + "-" + string(b[8:12]) + "-" + string(b[12:16]) + "-" + string(b[16:20]) + "-" + string(b[20:])
}

// FormValueOrFormFile tries to first read the key from form values, and if
// the value is empty it tries to read the key from one form files. An error
// is returned if it fails to read the form file. No error is returned if the
// key has not been found neither in form value nor in form files (by the value
// will be empty).
func FormValueOrFormFile(key string, r *http.Request) (value string, err error) {
	// form value
	value = strings.TrimSpace(r.FormValue(key))
	if len(value) > 0 {
		return
	}

	// form file
	file, fileHeader, err := r.FormFile(key)
	if err != nil {
		// we don't want to return an error if not found
		if errors.Is(err, http.ErrMissingFile) {
			return "", nil
		}
		return
	}
	fileContent := make([]byte, fileHeader.Size)
	if _, err = file.Read(fileContent); err != nil {
		return
	}
	return string(fileContent), nil
}

const (
	tlsFormDataTLSEnabled       string = "tls_enabled"
	tlsFormDataTLSInsecure      string = "tls_insecure"
	tlsFormDataTLSCA            string = "tls_ca"
	tlsFormDataTLSClientCert    string = "tls_user_cert"
	tlsFormDataTLSClientCertKey string = "tls_user_key"
)

// TLSConfig contains the common TLS configuration used by many endpoints.
type TLSConfig struct {
	Enabled       bool
	Insecure      bool
	CA            string
	ClientCert    string
	ClientCertKey string
}

// ParseTLSConfigFromFormData parses the TLS configuration from the form request.
// An error is return if one of the variable failed to be parsed to the correct
// type, or if one of the certificate file failed to be read (file too long for
// example).
func ParseTLSConfigFromFormData(l *log.Entry, r *http.Request) (c TLSConfig, err error) {
	l.Debug("parsing TLS configuration")
	// tls
	tlsEnabledString := strings.TrimSpace(r.FormValue(tlsFormDataTLSEnabled))
	if len(tlsEnabledString) > 0 {
		// TLS enabled
		c.Enabled, err = strconv.ParseBool(tlsEnabledString)
		if err != nil {
			return c, errors.WithMessage(err, "failed to parse TLS enabled to boolean")
		}
		if c.Enabled {
			l.Debug("TLS is enabled")
			// tls inscure
			tlsInsecureString := strings.TrimSpace(r.FormValue(tlsFormDataTLSInsecure))
			if len(tlsInsecureString) > 0 {
				c.Insecure, err = strconv.ParseBool(tlsInsecureString)
				if err != nil {
					return c, errors.WithMessage(err, "failed to parse TLS insecure to boolean")
				}
			}
			if c.Insecure {
				l.Debug("TLS is set to be insecure")
			} else {
				// CA file
				c.CA, err = FormValueOrFormFile(tlsFormDataTLSCA, r)
				if err != nil {
					return c, errors.WithMessage(err, "failed to read the CA file")
				}
			}
			// client cert
			c.ClientCert, err = FormValueOrFormFile(tlsFormDataTLSClientCert, r)
			if err != nil {
				return c, errors.WithMessage(err, "failed to read the client cert file")
			}
			// client cert key
			c.ClientCertKey, err = FormValueOrFormFile(tlsFormDataTLSClientCertKey, r)
			if err != nil {
				return c, errors.WithMessage(err, "failed to read the client cert key file")
			}
		}
	}

	if !c.Enabled {
		l.Debug("TLS is disabled")
	}

	return
}

// Validate makes sure the configuration is correct and ready to be consummed.
func (c TLSConfig) Validate() error {
	// tls
	if c.Enabled {
		// CA & Insecure
		if c.Insecure && (len(c.CA) > 0) {
			return errors.New("the CA has been defined while the TLS connection is set to be insecure, please remove one of the two")
		}
		// client certificates
		if (len(c.ClientCert) > 0) != (len(c.ClientCertKey) > 0) {
			return errors.New("both the client certificate and the client certificate key must be defined, or none of the two")
		}
	}

	return nil
}

// GetTLSConfig return the TLS configuration to be used in requests. An error
// is returned if one of the certificate failed to be parsed. nil is returned
// is TLS has been deactivated.
//
// This function is incompatible with a previous call of the WriteCertificatesOnDisk
// function as certificates contents will be replace with file paths (this function
// will always return an error).
func (c TLSConfig) GetTLSConfig(l *log.Entry) (*tls.Config, error) {
	if !c.Enabled {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.Insecure,
	}
	// ca
	if len(c.CA) > 0 {
		tlsConfig.RootCAs = x509.NewCertPool()
		if !tlsConfig.RootCAs.AppendCertsFromPEM([]byte(c.CA)) {
			return nil, errors.New("was not able to add the server CA certificate")
		}
		l.Debug("TLS config has a CA")
	}
	// client certs
	if len(c.ClientCert) > 0 {
		var err error
		tlsConfig.Certificates = make([]tls.Certificate, 1)
		tlsConfig.Certificates[0], err = tls.X509KeyPair([]byte(c.ClientCert), []byte(c.ClientCertKey))
		if err != nil {
			return nil, errors.WithMessage(err, "was not able to add client certificate and key")
		}
		l.Debug("TLS config has client certificate and key")
	}

	return tlsConfig, nil
}

// WriteCertificatesOnDisk writes certificate content in a temporary file
// on the disk. An error is returned if it fails to write one of the certificate
// on the disk. Once called, certificate contents (saved in parameters) will be
// replaced with the path of the temporary file.
// Once called, the DeleteCertificatesFromDisk function should be called, even
// though this function returns an error.
func (c *TLSConfig) WriteCertificatesOnDisk(l *log.Entry) (err error) {
	if !c.Enabled {
		return nil
	}

	// reseting variable as those will be replaced by file location
	// and that may cause an issue if an error occurs and the
	// "deleteCertificatesFromDisk" runs.
	caContent := c.CA
	clientCertContent := c.ClientCert
	clientKeyContent := c.ClientCertKey
	c.CA = ""
	c.ClientCert = ""
	c.ClientCertKey = ""

	// tls ca
	if len(caContent) > 0 {
		caFilePath := filepath.Join(TempFolderPath, GenerateUUID()+".crt")
		l.Debugf("writing TLS CA in temp file %s", caFilePath)
		if err = os.WriteFile(caFilePath, []byte(caContent), 0644); err != nil {
			return errors.WithMessagef(err, "failed to write TLS CA file to disk (path: %s)", caFilePath)
		}
		c.CA = caFilePath
	}
	// tls client cert
	if len(clientCertContent) > 0 {
		certFilePath := filepath.Join(TempFolderPath, GenerateUUID()+".crt")
		l.Debugf("writing user certificate in temp file %s", certFilePath)
		if err = os.WriteFile(certFilePath, []byte(clientCertContent), 0644); err != nil {
			return errors.WithMessagef(err, "failed to write TLS client cert file to disk (path: %s)", certFilePath)
		}
		c.ClientCert = certFilePath

		// tls client key
		keyFilePath := filepath.Join(TempFolderPath, GenerateUUID()+".key")
		l.Debugf("writing user certificate key in temp file %s", keyFilePath)
		if err = os.WriteFile(keyFilePath, []byte(clientKeyContent), 0644); err != nil {
			return errors.WithMessagef(err, "failed to write TLS client cert key file to disk (path: %s)", keyFilePath)
		}
		c.ClientCertKey = keyFilePath
	}
	return nil
}

// DeleteCertificatesFromDisk deletes certificates that have written on disk. It is
// safe to call this function even if certificates have not been saved on disk first,
// since the function checks it. If the file fails to be deleted, the error is not
// returned but logged as Error instead.
func (c *TLSConfig) DeleteCertificatesFromDisk(l *log.Entry) {
	if !c.Enabled {
		return
	}

	var err error
	for _, file := range []string{c.CA, c.ClientCert, c.ClientCertKey} {
		if (len(file) == 0) || !strings.HasPrefix(file, "/") {
			continue
		}

		// does file exists
		if _, err = os.Stat(file); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			} else {
				l.WithError(err).Warnf("failed to check if the file %s exists, will still try to delete it", file)
			}
		}

		// remove file
		l.Debugf("removing temporary file %s", file)
		if err = os.Remove(file); err != nil {
			l.WithError(err).Errorf("failed to remove file %s from disk", file)
		}
	}
}
