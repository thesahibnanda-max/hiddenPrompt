package constants

import "time"

const (
	InMemoryTTLForMusicDetails = 30 * time.Minute

	RedisKeyForMusicDetails = "music:details"
	RedisTTLForMusicDetails = 12 * time.Hour

	MusicDBLimit  = 100
	MusicDBOffset = 0

	// RedisLockKeyForMusicDetails guards concurrent cache-miss DB fetches
	// (thundering herd) - a single static key is correct here since there
	// is exactly one resource being protected (the whole catalog), not a
	// per-item lock.
	RedisLockKeyForMusicDetails = "music:details:lock"
	// RedisLockTTLForMusicDetails is intentionally short: the query it
	// guards is a small, indexed, at-most-~20-row Mongo read that should
	// normally complete in milliseconds. A long TTL here just means a
	// stuck or crashed lock holder blocks every other request for that
	// long over what should be a fast operation.
	RedisLockTTLForMusicDetails = 5 * time.Second
	// RedisLockRetryBackoff/RedisLockMaxRetries bound how long a waiting
	// request retries for the lock (up to ~1s total) before giving up and
	// fetching directly - long enough to usually catch a fast lock holder
	// finishing, short enough to not add much latency under contention.
	RedisLockRetryBackoff = 50 * time.Millisecond
	RedisLockMaxRetries   = 20
)
