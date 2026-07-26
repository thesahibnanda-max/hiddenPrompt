package config

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	goxJsonUtils "github.com/devlibx/gox-base/v2/serialization/utils/json"
	"github.com/go-viper/mapstructure/v2"
)

func decodeToStruct[T any](b64String string, validateFns ...func(t T) (T, error)) (T, error) {
	var zero T

	keyS, ok := os.LookupEnv("CONFIG_DECODING_KEY")
	if !ok || strings.TrimSpace(keyS) == "" {
		return zero, fmt.Errorf("key is not present in env vars")
	}

	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(keyS))
	if err != nil {
		return zero, err
	}

	b64String = strings.TrimSpace(b64String)
	data, err := base64.StdEncoding.DecodeString(b64String)
	if err != nil {
		return zero, err
	}

	if len(data) < 12 {
		return zero, fmt.Errorf("less than 12")
	}

	nonce := data[:12]
	ciphertext := data[12:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return zero, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return zero, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return zero, err
	}

	stringObjectMap, err := goxJsonUtils.StringToStringObjectMap(string(plaintext))
	if err != nil {
		return zero, err
	}

	var cfg T

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  &cfg,
		TagName: "json",
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
	})
	if err != nil {
		return zero, err
	}

	if err := decoder.Decode(stringObjectMap); err != nil {
		return zero, err
	}

	if len(validateFns) > 0 {
		for _, validateFn := range validateFns {
			if validateFn != nil {
				newConfig, valiDateErr := validateFn(cfg)
				if valiDateErr != nil {
					return zero, valiDateErr
				}
				cfg = newConfig
			}
		}
	}

	return cfg, nil
}
