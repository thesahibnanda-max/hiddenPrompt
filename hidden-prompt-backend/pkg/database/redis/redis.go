package redisRepo

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Repository interface {
	Set(ctx context.Context, key string, value string, exp ...time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
	// FlushDB wipes every key in the logical Redis DB in use - everything
	// here is ephemeral cache data (OTPs, rate-limit counters), never a
	// system of record, so a full flush is safe. Note it also resets every
	// in-flight rate-limit window for every user/IP, not just OTPs.
	FlushDB(ctx context.Context) error
}

func NewRepository(redisClient redis.Cmdable, defaultTTL time.Duration) (Repository, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("redisClient is nil")
	}
	return &repoImpl{
		redisClient: redisClient,
		defaultTTL:  defaultTTL,
	}, nil
}

type repoImpl struct {
	redisClient redis.Cmdable
	defaultTTL  time.Duration
}

func (r *repoImpl) Set(ctx context.Context, key string, value string, exp ...time.Duration) error {
	ttl := r.defaultTTL
	if len(exp) > 0 {
		ttl = exp[0]
	}

	return r.redisClient.Set(ctx, key, value, ttl).Err()
}

func (r *repoImpl) Get(ctx context.Context, key string) (string, error) {
	return r.redisClient.Get(ctx, key).Result()
}

func (r *repoImpl) Delete(ctx context.Context, key string) error {
	return r.redisClient.Del(ctx, key).Err()
}

func (r *repoImpl) FlushDB(ctx context.Context) error {
	// FLUSHDB has no key to hash, so a *redis.ClusterClient routes it to a
	// single arbitrary master node rather than the whole cluster (confirmed
	// against go-redis v9's cmdNode routing) - calling it directly would
	// silently leave every other shard's data untouched. Detect the cluster
	// case and fan out to every master explicitly; a plain *redis.Client
	// (single node) needs no special handling.
	if cluster, ok := r.redisClient.(*redis.ClusterClient); ok {
		return cluster.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			return client.FlushDB(ctx).Err()
		})
	}
	return r.redisClient.FlushDB(ctx).Err()
}
