package config

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Explicit, short connection timeouts/retry bounds for the Redis client -
// go-redis v9's own defaults (DialTimeout=5s, MaxRetries=3) let a single
// stale pooled connection or transient network hiccup block a call for up
// to ~15s before erroring. Every current caller of this client that sits
// on a request's hot path (enforceRateLimit, the L2 music cache) already
// treats a Redis error as non-fatal/fail-open by design, so there is no
// reason a bad connection should ever be allowed to block a request
// anywhere near that long - failing fast costs nothing since the fallback
// path already exists.
const (
	redisDialTimeout  = 1 * time.Second
	redisReadTimeout  = 1 * time.Second
	redisWriteTimeout = 1 * time.Second
	redisMaxRetries   = 1
)

//go:embed redis.txt
var redisString string

type redisConfig struct {
	Redis redisConnectionConfig `json:"redis_config"`
}

type redisConnectionConfig struct {
	Addrs                 []string `json:"addrs"`
	Username              string   `json:"username"`
	Password              string   `json:"password"`
	Database              int      `json:"database"`
	InsecureTLSSkipVerify bool     `json:"insecure_tls_skip_verify"`
	CACertString          string   `json:"ca_cert_string"`
}

func redisClient() (redis.Cmdable, error) {
	cfgWrapper, err := decodeToStruct[redisConfig](
		redisString,
		func(cfg redisConfig) (redisConfig, error) {
			if len(cfg.Redis.Addrs) == 0 {
				return cfg, fmt.Errorf("redis addrs are empty")
			}

			for i, addr := range cfg.Redis.Addrs {
				addr = strings.TrimSpace(addr)
				if addr == "" {
					return cfg, fmt.Errorf("redis addr[%d] is empty", i)
				}
				cfg.Redis.Addrs[i] = addr
			}

			cfg.Redis.Username = strings.TrimSpace(cfg.Redis.Username)
			cfg.Redis.CACertString = strings.TrimSpace(cfg.Redis.CACertString)

			if cfg.Redis.Database < 0 {
				return cfg, fmt.Errorf("redis database is invalid")
			}

			if !cfg.Redis.InsecureTLSSkipVerify &&
				cfg.Redis.CACertString == "" {
				return cfg, fmt.Errorf("ca_cert_string is required when insecure_tls_skip_verify is false")
			}

			return cfg, nil
		},
	)
	if err != nil {
		return nil, err
	}

	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cfgWrapper.Redis.InsecureTLSSkipVerify,
	}

	if cfgWrapper.Redis.CACertString != "" {
		rootCAs := x509.NewCertPool()

		if !rootCAs.AppendCertsFromPEM([]byte(cfgWrapper.Redis.CACertString)) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsCfg.RootCAs = rootCAs
	}

	if len(cfgWrapper.Redis.Addrs) == 1 {
		return redis.NewClient(&redis.Options{
			Addr:         cfgWrapper.Redis.Addrs[0],
			Username:     cfgWrapper.Redis.Username,
			Password:     cfgWrapper.Redis.Password,
			DB:           cfgWrapper.Redis.Database,
			TLSConfig:    tlsCfg,
			DialTimeout:  redisDialTimeout,
			ReadTimeout:  redisReadTimeout,
			WriteTimeout: redisWriteTimeout,
			MaxRetries:   redisMaxRetries,
		}), nil
	}

	return redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        cfgWrapper.Redis.Addrs,
		Username:     cfgWrapper.Redis.Username,
		Password:     cfgWrapper.Redis.Password,
		TLSConfig:    tlsCfg,
		DialTimeout:  redisDialTimeout,
		ReadTimeout:  redisReadTimeout,
		WriteTimeout: redisWriteTimeout,
		MaxRetries:   redisMaxRetries,
	}), nil
}
