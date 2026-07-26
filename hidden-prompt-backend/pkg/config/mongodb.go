package config

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed mongodb.txt
var mongoDBString string

type mongoConfig struct {
	MongoString string `json:"mongo_string"`
}

func getMongoClientURI() (string, error) {
	m, err := decodeToStruct(mongoDBString, func(t mongoConfig) (mongoConfig, error) {
		t.MongoString = strings.TrimSpace(t.MongoString)
		if t.MongoString == "" {
			return t, fmt.Errorf("empty MongoString")
		}
		return t, nil
	})
	if err != nil {
		return "", err
	}

	return m.MongoString, nil
}
