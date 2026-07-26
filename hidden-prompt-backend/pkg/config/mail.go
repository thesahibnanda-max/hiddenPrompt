package config

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed mail.txt
var mailConfigString string

type MailConnectionConfig struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

type mailConnectionConfig struct {
	MailConnections []MailConnectionConfig `json:"mail_connections"`
}

func mailConnConfig() ([]MailConnectionConfig, error) {
	m, err := decodeToStruct[mailConnectionConfig](mailConfigString, func(t mailConnectionConfig) (mailConnectionConfig, error) {
		if len(t.MailConnections) == 0 {
			return t, fmt.Errorf("no mail conns")
		}
		return t, nil
	}, func(t mailConnectionConfig) (mailConnectionConfig, error) {

		newMailConns := make([]MailConnectionConfig, 0)
		for _, ma := range t.MailConnections {
			ma.Host = strings.TrimSpace(ma.Host)
			ma.Username = strings.TrimSpace(ma.Username)
			ma.Password = strings.TrimSpace(ma.Password)

			if ma.Host == "" {
				return t, fmt.Errorf("host is error")
			}
			if ma.Password == "" {
				return t, fmt.Errorf("password is error")
			}
			if ma.Username == "" {
				return t, fmt.Errorf("username is error")
			}
			if ma.Port <= 0 {
				return t, fmt.Errorf("port is 0")
			}
			newMailConns = append(newMailConns, ma)
		}
		t.MailConnections = newMailConns
		return t, nil
	})
	if err != nil {
		return nil, err
	}

	return m.MailConnections, nil
}
