package admin_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"hidden-prompt-backend/pkg/database"
	usersRepo "hidden-prompt-backend/pkg/database/mysql/users"
	puzzlesRepo "hidden-prompt-backend/pkg/database/psql/puzzles"
	redisRepo "hidden-prompt-backend/pkg/database/redis"
	"hidden-prompt-backend/pkg/service/admin"

	"github.com/stretchr/testify/require"
)

// fakeUsersRepo/fakePuzzlesRepo/fakeRedisRepo are minimal in-memory stand-ins
// for the repository interfaces, following this repo's established fake
// pattern (see pkg/chat/chat_test.go) rather than a mocking library.
type fakeUsersRepo struct {
	usersRepo.Repository
	deleteCalledWith string
	deleteErr        error
}

func (f *fakeUsersRepo) DeleteUserByEmail(_ context.Context, email string) error {
	f.deleteCalledWith = email
	return f.deleteErr
}

type fakePuzzlesRepo struct {
	puzzlesRepo.Repository
	deleteCalledWith string
	deleteErr        error
}

func (f *fakePuzzlesRepo) DeletePuzzlesByEmail(_ context.Context, email string) error {
	f.deleteCalledWith = email
	return f.deleteErr
}

type fakeRedisRepo struct {
	redisRepo.Repository
	flushCalled bool
	flushErr    error
}

func (f *fakeRedisRepo) FlushDB(context.Context) error {
	f.flushCalled = true
	return f.flushErr
}

func withAdminKey(t *testing.T, key string) {
	t.Helper()
	t.Setenv("ADMIN_KEY", key)
}

func Test_NewService_NilDB(t *testing.T) {
	withAdminKey(t, "secret")
	svc, err := admin.NewService(nil)
	require.Error(t, err)
	require.Nil(t, svc)
}

func Test_NewService_MissingAdminKey(t *testing.T) {
	require.NoError(t, os.Unsetenv("ADMIN_KEY"))
	svc, err := admin.NewService(&database.DatabaseParams{})
	require.Error(t, err)
	require.Nil(t, svc)
}

func Test_NewService_EmptyAdminKey(t *testing.T) {
	withAdminKey(t, "")
	svc, err := admin.NewService(&database.DatabaseParams{})
	require.Error(t, err, "a set-but-empty ADMIN_KEY must not silently construct a service")
	require.Nil(t, svc)
}

func Test_ValidateAdminKey(t *testing.T) {
	withAdminKey(t, "correct-horse-battery-staple")
	svc, err := admin.NewService(&database.DatabaseParams{})
	require.NoError(t, err)

	require.True(t, svc.ValidateAdminKey("correct-horse-battery-staple"))
	require.False(t, svc.ValidateAdminKey("wrong"))
	require.False(t, svc.ValidateAdminKey(""))
}

func Test_ClearDataFromCache(t *testing.T) {
	withAdminKey(t, "secret")
	redis := &fakeRedisRepo{}
	svc, err := admin.NewService(&database.DatabaseParams{RedisRepository: redis})
	require.NoError(t, err)

	require.NoError(t, svc.ClearDataFromCache(context.Background()))
	require.True(t, redis.flushCalled)
}

func Test_ClearDataFromCache_PropagatesError(t *testing.T) {
	withAdminKey(t, "secret")
	redis := &fakeRedisRepo{flushErr: errors.New("redis is down")}
	svc, err := admin.NewService(&database.DatabaseParams{RedisRepository: redis})
	require.NoError(t, err)

	require.Error(t, svc.ClearDataFromCache(context.Background()))
}

func Test_DeleteUserRelatedDataFromRelationalDBs(t *testing.T) {
	withAdminKey(t, "secret")
	users := &fakeUsersRepo{}
	puzzles := &fakePuzzlesRepo{}
	svc, err := admin.NewService(&database.DatabaseParams{UsersRepository: users, PuzzlesRepository: puzzles})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteUserRelatedDataFromRelationalDBs(context.Background(), "Player@Example.com"))
	require.Equal(t, "player@example.com", users.deleteCalledWith, "email must be normalized before use")
	require.Equal(t, "player@example.com", puzzles.deleteCalledWith)
}

func Test_DeleteUserRelatedDataFromRelationalDBs_InvalidEmail(t *testing.T) {
	withAdminKey(t, "secret")
	users := &fakeUsersRepo{}
	puzzles := &fakePuzzlesRepo{}
	svc, err := admin.NewService(&database.DatabaseParams{UsersRepository: users, PuzzlesRepository: puzzles})
	require.NoError(t, err)

	err = svc.DeleteUserRelatedDataFromRelationalDBs(context.Background(), "not-an-email")
	require.Error(t, err)
	require.Empty(t, users.deleteCalledWith, "must not touch either DB when the email itself is invalid")
	require.Empty(t, puzzles.deleteCalledWith)
}

func Test_DeleteUserRelatedDataFromRelationalDBs_StopsAtFirstError(t *testing.T) {
	withAdminKey(t, "secret")
	users := &fakeUsersRepo{deleteErr: errors.New("mysql is down")}
	puzzles := &fakePuzzlesRepo{}
	svc, err := admin.NewService(&database.DatabaseParams{UsersRepository: users, PuzzlesRepository: puzzles})
	require.NoError(t, err)

	err = svc.DeleteUserRelatedDataFromRelationalDBs(context.Background(), "player@example.com")
	require.Error(t, err)
	require.Empty(t, puzzles.deleteCalledWith, "postgres delete must not run once the mysql delete already failed")
}
