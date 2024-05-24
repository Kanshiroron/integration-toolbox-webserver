package main

import (
	"database/sql"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	// databases drivers
	"github.com/go-sql-driver/mysql"    // mysql
	_ "github.com/lib/pq"               // postgresql
	_ "github.com/microsoft/go-mssqldb" // Microsoft SQL server
)

const (
	// engines
	databaseEngineMSSQL      string = "sqlserver"
	databaseEngineMySQL      string = "mysql"
	databaseEnginePostgreSQL string = "postgres"

	// default ports
	databaseEngineMSSQLDefaultPort      int = 1433
	databaseEngineMySQLDefaultPort      int = 3306
	databaseEnginePostgreSQLDefaultPort int = 5432

	// form data
	databaseFormDataEngine   string = "engine"
	databaseFormDataHost     string = "host"
	databaseFormDataPort     string = "port"
	databaseFormDataUsername string = "username"
	databaseFormDataPassword string = "password"
	databaseFormDataDBName   string = "db_name"
	databaseFormDataTLSMode  string = "ssl_mode"
	databaseFormDataQuery    string = "query"
)

type DatabaseEndpoints struct{}

func NewDatabaseEndpoints() *DatabaseEndpoints {
	return &DatabaseEndpoints{}
}

func (e *DatabaseEndpoints) Connect(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	// do we have a POST method? (mandatory according to RFC)
	if r.Method != http.MethodPost {
		l.Warnf("only the POST method is allowed for posting forms, according to RFC 1867 (%s used)", r.Method)
	}

	// connection config
	config, err := parseDBConfigFromFormData(l, r)
	if err != nil {
		errorString := "failed to parse connection config"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}
	if err = config.Validate(); err != nil {
		errorString := "invalid connection config"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}

	// writing certificates on disks
	if config.needCertsAsFile() {
		err = config.tlsConfig.WriteCertificatesOnDisk(l)
		defer config.tlsConfig.DeleteCertificatesFromDisk(l)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			l.Error(err.Error())
			return
		}
	}

	// opening db connection
	db := e.open(l, w, config)
	if db == nil {
		return // error already sent
	}
	defer db.Close()

	// testing connection
	if err = db.Ping(); err != nil {
		errorString := "failed to ping the database"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (e *DatabaseEndpoints) Query(l *log.Entry, w http.ResponseWriter, r *http.Request) {
	// do we have a POST method? (mandatory according to RFC)
	if r.Method != http.MethodPost {
		l.Warnf("only the POST method is allowed for posting forms, according to RFC 1867 (%s used)", r.Method)
	}

	// connection config
	config, err := parseDBConfigFromFormData(l, r)
	if err != nil {
		errorString := "failed to parse connection config"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}
	if err = config.Validate(); err != nil {
		errorString := "invalid connection config"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}
	// query
	query := strings.TrimSpace(r.FormValue(databaseFormDataQuery))
	if len(query) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		errorString := "empty SQL query"
		w.Write([]byte(errorString))
		l.Warn(errorString)
		return
	}

	// writing certificates on disks
	if config.needCertsAsFile() {
		err = config.tlsConfig.WriteCertificatesOnDisk(l)
		defer config.tlsConfig.DeleteCertificatesFromDisk(l)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			l.Error(err.Error())
			return
		}
	}

	// opening db connection
	db := e.open(l, w, config)
	if db == nil {
		return // error already sent
	}
	defer db.Close()

	// running query
	l.Debugf("running query: %s", query)
	row, err := db.Query(query)
	if err != nil {
		errorString := "failed to run the query against the database"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return
	}
	defer row.Close()

	// TO DO return result?

	w.WriteHeader(http.StatusOK)
}

func (d *DatabaseEndpoints) open(l *log.Entry, w http.ResponseWriter, config dbConfig) *sql.DB {
	// datasource
	var dataSource string
	switch config.engine {
	case databaseEngineMSSQL: // Microsoft SQL server
		dataSource = config.getMSSQLDataSource()
	case databaseEngineMySQL: // MySQL
		var err error
		dataSource, err = config.getMySQLDataSource(l)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			l.Warn(err.Error())
			return nil
		}
	case databaseEnginePostgreSQL: // PostgreSQL
		dataSource = config.getPSQLDataSource()
	}

	// opening connection
	l.Debugf("opening connection to database: %s", dataSource)
	db, err := sql.Open(config.engine, dataSource)
	if err != nil {
		errorString := "failed to open database connection"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorString + ": " + err.Error()))
		l.WithError(err).Warn(errorString)
		return nil
	}
	return db
}

type dbConfig struct {
	engine    string
	host      string
	port      int
	username  string
	password  string
	dbName    string
	tlsConfig TLSConfig
	tlsMode   string
}

func parseDBConfigFromFormData(l *log.Entry, r *http.Request) (c dbConfig, err error) {
	l.Debug("parsing connection configuration")
	// parse form
	if err = r.ParseMultipartForm(MaxFormSize); err != nil {
		return
	}

	c.engine = strings.TrimSpace(r.FormValue(databaseFormDataEngine))
	c.host = strings.TrimSpace(r.FormValue(databaseFormDataHost))
	c.username = strings.TrimSpace(r.FormValue(databaseFormDataUsername))
	c.password = strings.TrimSpace(r.FormValue(databaseFormDataPassword))
	c.dbName = strings.TrimSpace(r.FormValue(databaseFormDataDBName))

	// port
	portString := strings.TrimSpace(r.FormValue(databaseFormDataPort))
	if len(portString) > 0 {
		if c.port, err = strconv.Atoi(portString); err != nil {
			return c, errors.WithMessage(err, "failed to parse port to int")
		}
	}

	// tls
	c.tlsConfig, err = ParseTLSConfigFromFormData(l, r)
	if err != nil {
		return
	}
	// ssl mode
	if c.tlsConfig.Enabled {
		c.tlsMode = strings.TrimSpace(r.FormValue(databaseFormDataTLSMode))
	}
	return
}

func (c dbConfig) Validate() error {
	// engine
	if len(c.engine) == 0 {
		return errors.New("database engine not set")
	}
	switch c.engine {
	case databaseEngineMSSQL, databaseEngineMySQL, databaseEnginePostgreSQL:
	default:
		return errors.Errorf("unknwon database engine %q, must be one of: %s, %s, %s", c.engine, databaseEngineMSSQL, databaseEngineMySQL, databaseEnginePostgreSQL)
	}

	// password set and not the username
	if len(c.password) > 0 && len(c.username) == 0 {
		return errors.New("password is set while the username is not")
	}

	// tls
	if err := c.tlsConfig.Validate(); err != nil {
		return err
	}
	if c.tlsConfig.Enabled {
		// specific engine config
		switch c.engine {
		case databaseEngineMSSQL: // MS SQL
			if len(c.tlsConfig.ClientCert) > 0 {
				return errors.New("client certificate and key are not supported for Microsoft SQL server")
			}
		case databaseEnginePostgreSQL: // postgres
			if len(c.tlsMode) == 0 {
				return errors.New("TLS mode not set")
			}
			switch c.tlsMode {
			case "require", "verify-ca", "verify-full":
			default:
				return errors.Errorf("unknown TLS mode: %s", c.tlsMode)
			}
		}
	}

	return nil
}

func (c dbConfig) needCertsAsFile() bool {
	return c.tlsConfig.Enabled && (c.engine != databaseEngineMySQL)
}

func (c dbConfig) getMSSQLDataSource() string {
	// default options
	query := url.Values{}
	query.Add("app name", "intergration-test-server")
	query.Add("connection timeout", "20")
	// db name
	if len(c.dbName) > 0 {
		query.Add("database", c.dbName)
	}
	// tls
	if c.tlsConfig.Enabled {
		query.Add("encrypt", "mandatory")
		if c.tlsConfig.Insecure {
			query.Add("TrustServerCertificate", "true")
		} else if len(c.tlsConfig.CA) > 0 {
			query.Add("certificate", c.tlsConfig.CA)
		}
	}

	u := &url.URL{
		Scheme:   "sqlserver",
		RawQuery: query.Encode(),
	}
	// host
	if len(c.host) > 0 {
		u.Host = c.host
	}
	// port
	if c.port != 0 && c.port != databaseEngineMSSQLDefaultPort {
		u.Host = u.Host + ":" + strconv.Itoa(c.port)
	}
	// username
	if len(c.username) > 0 {
		u.User = url.UserPassword(c.username, c.password)
	}
	return u.String()
}

func (c dbConfig) getMySQLDataSource(l *log.Entry) (_ string, err error) {
	mySQLConfig := mysql.NewConfig()

	// host
	if len(c.host) > 0 {
		mySQLConfig.Addr = c.host
	}
	// port
	if c.port != 0 && c.port != databaseEngineMySQLDefaultPort {
		mySQLConfig.Addr = mySQLConfig.Addr + ":" + strconv.Itoa(c.port)
	}
	// username
	if len(c.username) > 0 {
		mySQLConfig.User = c.username
		// password
		if len(c.password) > 0 {
			mySQLConfig.Passwd = c.password
		}
	}
	// db name
	if len(c.dbName) > 0 {
		mySQLConfig.DBName = c.dbName
	}

	// tls
	mySQLConfig.TLS, err = c.tlsConfig.GetTLSConfig(l)
	if err != nil {
		return "", err
	}

	return mySQLConfig.FormatDSN(), nil
}

func (c dbConfig) getPSQLDataSource() string {
	// default options
	dataSourceComponents := []string{
		"fallback_application_name=intergration-test-server",
		"connect_timeout=20",
	}
	// host
	if len(c.host) > 0 {
		dataSourceComponents = append(dataSourceComponents, "host="+c.host)
	}
	// port
	if c.port != 0 && c.port != databaseEnginePostgreSQLDefaultPort {
		dataSourceComponents = append(dataSourceComponents, "port="+strconv.Itoa(c.port))
	}
	// username
	if len(c.username) > 0 {
		dataSourceComponents = append(dataSourceComponents, "user="+c.username)
		// password
		if len(c.password) > 0 {
			dataSourceComponents = append(dataSourceComponents, "password="+psqlEscapeConnectionString(c.password))
		}
	}
	// db name
	if len(c.dbName) > 0 {
		dataSourceComponents = append(dataSourceComponents, "dbname="+c.dbName)
	}

	if c.tlsConfig.Enabled {
		dataSourceComponents = append(dataSourceComponents, "sslmode="+c.tlsMode)
		if len(c.tlsConfig.CA) > 0 {
			dataSourceComponents = append(dataSourceComponents, "sslrootcert="+c.tlsConfig.CA)
		}
		if len(c.tlsConfig.ClientCert) > 0 {
			dataSourceComponents = append(dataSourceComponents, "sslcert="+c.tlsConfig.ClientCert)
			dataSourceComponents = append(dataSourceComponents, "sslkey="+c.tlsConfig.ClientCertKey)
		}
	}

	return strings.Join(dataSourceComponents, " ")
}

var (
	psqlEscapedChars = []string{" ", "'"}
)

func psqlEscapeConnectionString(s string) string {
	for _, escaped_char := range psqlEscapedChars {
		s = strings.ReplaceAll(s, escaped_char, "\\"+escaped_char)
	}
	return "'" + s + "'"
}
