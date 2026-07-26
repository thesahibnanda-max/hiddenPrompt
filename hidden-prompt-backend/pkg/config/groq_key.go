package config

import (
	_ "embed"
	"fmt"
	"math/rand/v2"
)

//go:embed groq_keys.txt
var groqAPIsString string

func groqConfigKeys() ([]string, error) {

	type groqStruct struct {
		GroqKeys []string `json:"groq_keys"`
	}

	ans, err := decodeToStruct(groqAPIsString, func(t groqStruct) (groqStruct, error) {
		if len(t.GroqKeys) == 0 {
			return t, fmt.Errorf("no keys mentioned")
		}
		return t, nil
	}, func(t groqStruct) (groqStruct, error) {
		rand.Shuffle(len(t.GroqKeys), func(i, j int) {
			t.GroqKeys[i], t.GroqKeys[j] = t.GroqKeys[j], t.GroqKeys[i]
		})
		return t, nil
	})
	if err != nil {
		return make([]string, 0), err
	}
	return ans.GroqKeys, nil
}
