package config

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/go-sql-driver/mysql"
)

//go:embed mysql.txt
var mysqlString string

type mySQLConfig struct {
	MySQL mySQLConnectionConfig `json:"mysql_config"`
}

type mySQLConnectionConfig struct {
	Host                  string `json:"host"`
	Port                  uint16 `json:"port"`
	Username              string `json:"username"`
	Database              string `json:"database"`
	Password              string `json:"password"`
	InsecureTLSSkipVerify bool   `json:"insecure_tls_skip_verify"`
	CACertString          string `json:"ca_cert_string"`
}

var (
	mysqlTLSRegisterOnce sync.Once
	mysqlTLSRegisterErr  error
)

func mySQLConfigInternal() (*mysql.Config, error) {
	cfgWrapper, err := decodeToStruct(
		mysqlString,
		func(cfg mySQLConfig) (mySQLConfig, error) {
			cfg.MySQL.Host = strings.TrimSpace(cfg.MySQL.Host)
			cfg.MySQL.Username = strings.TrimSpace(cfg.MySQL.Username)
			cfg.MySQL.Database = strings.TrimSpace(cfg.MySQL.Database)
			cfg.MySQL.CACertString = strings.TrimSpace(cfg.MySQL.CACertString)

			if cfg.MySQL.Host == "" {
				return cfg, fmt.Errorf("mysql host is empty")
			}

			if cfg.MySQL.Port == 0 {
				return cfg, fmt.Errorf("mysql port is invalid")
			}

			if cfg.MySQL.Username == "" {
				return cfg, fmt.Errorf("mysql username is empty")
			}

			if cfg.MySQL.Database == "" {
				return cfg, fmt.Errorf("mysql database is empty")
			}

			if cfg.MySQL.Password == "" {
				return cfg, fmt.Errorf("mysql password is empty")
			}

			if !cfg.MySQL.InsecureTLSSkipVerify && cfg.MySQL.CACertString == "" {
				return cfg, fmt.Errorf("ca_cert_string is required when insecure_tls_skip_verify is false")
			}

			return cfg, nil
		},
	)
	if err != nil {
		return nil, err
	}

	cfg := mysql.NewConfig()

	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", cfgWrapper.MySQL.Host, cfgWrapper.MySQL.Port)
	cfg.User = cfgWrapper.MySQL.Username
	cfg.Passwd = cfgWrapper.MySQL.Password
	cfg.DBName = cfgWrapper.MySQL.Database
	cfg.ParseTime = true

	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cfgWrapper.MySQL.InsecureTLSSkipVerify,
		ServerName:         cfgWrapper.MySQL.Host,
	}

	if cfgWrapper.MySQL.CACertString != "" {
		rootCAs := x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM([]byte(cfgWrapper.MySQL.CACertString)) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsCfg.RootCAs = rootCAs
	}

	const tlsConfigName = "hidden-prompt-mysql"

	mysqlTLSRegisterOnce.Do(func() {
		mysqlTLSRegisterErr = mysql.RegisterTLSConfig(tlsConfigName, tlsCfg)
	})

	if mysqlTLSRegisterErr != nil {
		return nil, mysqlTLSRegisterErr
	}

	cfg.TLSConfig = tlsConfigName

	return cfg, nil
}
