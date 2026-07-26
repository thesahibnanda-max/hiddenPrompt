package admin

import (
	"context"
	"crypto/subtle"
	"fmt"
	"hidden-prompt-backend/pkg/database"
	customErrors "hidden-prompt-backend/pkg/errors"
	"hidden-prompt-backend/pkg/service/music"
	"hidden-prompt-backend/pkg/utils"
	"os"
	"strings"
)

type Service interface {
	// ValidateAdminKey constant-time-compares key against the configured
	// ADMIN_KEY. The admin secret stays encapsulated here (the one place
	// that already reads it from env), mirroring how the user-token secret
	// stays encapsulated in pkg/utils rather than leaking into handlers.
	ValidateAdminKey(key string) bool
	ClearDataFromCache(ctx context.Context) error
	DeleteUserRelatedDataFromRelationalDBs(ctx context.Context, email string) error
}

type service struct {
	adminKey     string
	db           *database.DatabaseParams
	musicService music.Service
}

func NewService(db *database.DatabaseParams) (Service, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	adminKey, ok := os.LookupEnv("ADMIN_KEY")
	// Was `&&` - a set-but-empty ADMIN_KEY would slip through, and since
	// ValidateAdminKey does a byte-compare, an empty stored key would then
	// accept an empty X-Admin-Key header as valid. Must be `||`.
	if !ok || strings.TrimSpace(adminKey) == "" {
		return nil, fmt.Errorf("admin key not set in envs")
	}

	return &service{
		adminKey: strings.TrimSpace(adminKey),
		db:       db,
	}, nil
}

// ValidateAdminKey implements [Service].
func (s *service) ValidateAdminKey(key string) bool {
	// Defense in depth alongside the NewService guard: never let an empty
	// submitted key match an (in-principle-impossible-but-just-in-case)
	// empty stored key.
	if key == "" || s.adminKey == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(key), []byte(s.adminKey)) == 1
}

// ClearDataFromCache implements [Service]. Redis here is purely ephemeral
// (OTPs, rate-limit counters) - never a system of record - so a full flush
// is safe, but it does reset every in-flight rate-limit window for every
// user/IP, not just OTP-related keys.
func (s *service) ClearDataFromCache(ctx context.Context) error {
	return s.db.RedisRepository.FlushDB(ctx)
}

// DeleteUserRelatedDataFromRelationalDBs implements [Service]. Deletes the
// user's row from MySQL and all of their puzzles from Postgres - the only
// two tables in the system that reference an email at all.
//
// Deliberately sequential, not parallelized: the Postgres delete must
// never run once the MySQL delete has already failed (see
// Test_DeleteUserRelatedDataFromRelationalDBs_StopsAtFirstError) - a
// fail-fast contract that plain concurrency can't preserve, since by the
// time a concurrently-started MySQL delete's failure is observed, a
// concurrently-started Postgres delete may already be in flight or done.
func (s *service) DeleteUserRelatedDataFromRelationalDBs(ctx context.Context, email string) error {
	email, err := utils.ValidateEmail(email)
	if err != nil {
		return &customErrors.EMailIsInvalid{Err: err}
	}

	if err := s.db.UsersRepository.DeleteUserByEmail(ctx, email); err != nil {
		return err
	}

	return s.db.PuzzlesRepository.DeletePuzzlesByEmail(ctx, email)
}
