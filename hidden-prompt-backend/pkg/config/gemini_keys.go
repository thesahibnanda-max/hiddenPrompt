package config

import (
	_ "embed"
	"fmt"
	"math/rand/v2"
)

//go:embed gemini_keys.txt
var geminiKeysString string

func geminiConfigKeys() ([]string, error) {
	type geminiKeys struct {
		GeminiKeys []string `json:"gemini_keys"`
	}
	ans, err := decodeToStruct(geminiKeysString, func(t geminiKeys) (geminiKeys, error) {
		if len(t.GeminiKeys) == 0 {
			return t, fmt.Errorf("0 keys available ")
		}
		return t, nil
	}, func(t geminiKeys) (geminiKeys, error) {
		rand.Shuffle(len(t.GeminiKeys), func(i, j int) {
			t.GeminiKeys[i], t.GeminiKeys[j] = t.GeminiKeys[j], t.GeminiKeys[i]
		})
		return t, nil
	})
	if err != nil {
		return make([]string, 0), err
	}
	return ans.GeminiKeys, nil
}
