package utils

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/miscreant/miscreant.go"
)

const cryptoKey = "67ad40d952ee9a1c0046d43669058368fbf58e549ab6b382ffdf7c3cd48ccfbe"

var (
	cipherPool sync.Pool
	initErr    error
	initOnce   sync.Once
)

func initCrypto() error {
	initOnce.Do(func() {
		key, err := hex.DecodeString(cryptoKey)
		if err != nil {
			initErr = fmt.Errorf("decode crypto key: %w", err)
			return
		}

		if len(key) != 32 {
			initErr = fmt.Errorf("invalid AES-256 key length: got %d bytes", len(key))
			return
		}

		// Verify the key can actually construct a cipher.
		testCipher, err := miscreant.NewAESCMACSIV(key)
		if err != nil {
			initErr = fmt.Errorf("initialize AES-SIV cipher: %w", err)
			return
		}

		// Paranoid sanity check.
		if testCipher == nil {
			initErr = fmt.Errorf("miscreant returned nil cipher")
			return
		}

		cipherPool = sync.Pool{
			New: func() any {
				c, err := miscreant.NewAESCMACSIV(key)
				if err != nil || c == nil {
					// This should never happen because we already
					// validated the key above. Returning nil lets
					// callers fail gracefully instead of panicking.
					return nil
				}
				return c
			},
		}
	})

	return initErr
}

func getCipher() (*miscreant.Cipher, error) {
	if err := initCrypto(); err != nil {
		return nil, err
	}

	v := cipherPool.Get()
	if v == nil {
		return nil, fmt.Errorf("cipher pool returned nil")
	}

	c, ok := v.(*miscreant.Cipher)
	if !ok {
		return nil, fmt.Errorf("cipher pool returned unexpected type %T", v)
	}

	if c == nil {
		return nil, fmt.Errorf("cipher pool returned nil cipher")
	}

	return c, nil
}

func putCipher(c *miscreant.Cipher) {
	if c != nil {
		cipherPool.Put(c)
	}
}

func EncryptString(plaintext string) (string, error) {
	c, err := getCipher()
	if err != nil {
		return "", err
	}
	defer putCipher(c)

	ct, err := c.Seal(nil, []byte(plaintext))
	if err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}

	return base64.StdEncoding.EncodeToString(ct), nil
}

func DecryptString(ciphertext string) (string, error) {
	c, err := getCipher()
	if err != nil {
		return "", err
	}
	defer putCipher(c)

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	pt, err := c.Open(nil, raw)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(pt), nil
}
