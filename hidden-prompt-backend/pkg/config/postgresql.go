package config

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed postgresql.txt
var postgresqlString string

type postgreSQLConfig struct {
	PostgreSQL postgreSQLConnectionConfig `json:"postgre_sql_config"`
}

type postgreSQLConnectionConfig struct {
	Host                  string `json:"host"`
	Port                  uint16 `json:"port"`
	Username              string `json:"username"`
	Database              string `json:"database"`
	Password              string `json:"password"`
	InsecureTLSSkipVerify bool   `json:"insecure_tls_skip_verify"`
	CACertString          string `json:"ca_cert_string"`
}

func postgresqlConfig() (*pgx.ConnConfig, error) {
	cfgWrapper, err := decodeToStruct[postgreSQLConfig](
		postgresqlString,
		func(cfg postgreSQLConfig) (postgreSQLConfig, error) {
			cfg.PostgreSQL.Host = strings.TrimSpace(cfg.PostgreSQL.Host)
			cfg.PostgreSQL.Username = strings.TrimSpace(cfg.PostgreSQL.Username)
			cfg.PostgreSQL.Database = strings.TrimSpace(cfg.PostgreSQL.Database)
			cfg.PostgreSQL.Password = strings.TrimSpace(cfg.PostgreSQL.Password)
			cfg.PostgreSQL.CACertString = strings.TrimSpace(cfg.PostgreSQL.CACertString)

			if cfg.PostgreSQL.Host == "" {
				return cfg, fmt.Errorf("postgresql host is empty")
			}

			if cfg.PostgreSQL.Port == 0 {
				return cfg, fmt.Errorf("postgresql port is invalid")
			}

			if cfg.PostgreSQL.Username == "" {
				return cfg, fmt.Errorf("postgresql username is empty")
			}

			if cfg.PostgreSQL.Database == "" {
				return cfg, fmt.Errorf("postgresql database is empty")
			}

			if cfg.PostgreSQL.Password == "" {
				return cfg, fmt.Errorf("postgresql password is empty")
			}

			if cfg.PostgreSQL.CACertString == "" {
				return cfg, fmt.Errorf("ca_cert_string is required when insecure_tls_skip_verify is false")
			}

			return cfg, nil
		},
	)
	if err != nil {
		return nil, err
	}

	connConfig, err := pgx.ParseConfig("")
	if err != nil {
		return nil, err
	}

	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cfgWrapper.PostgreSQL.InsecureTLSSkipVerify,
	}
	if cfgWrapper.PostgreSQL.CACertString != "" {
		rootCAs := x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM([]byte(cfgWrapper.PostgreSQL.CACertString)) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsCfg.RootCAs = rootCAs
	}

	connConfig.Host = cfgWrapper.PostgreSQL.Host
	connConfig.Port = cfgWrapper.PostgreSQL.Port
	connConfig.Database = cfgWrapper.PostgreSQL.Database
	connConfig.User = cfgWrapper.PostgreSQL.Username
	connConfig.Password = cfgWrapper.PostgreSQL.Password
	connConfig.TLSConfig = tlsCfg

	return connConfig, nil
}
