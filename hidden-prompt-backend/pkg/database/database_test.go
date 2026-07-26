package database_test

import (
	"context"
	"hidden-prompt-backend/pkg/config"
	"hidden-prompt-backend/pkg/database"
	"hidden-prompt-backend/pkg/utils"
	"os"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Test_NewDatabase(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)
	require.NotNil(t, db)

	pCtx := db.PingContextFunction(context.Background())
	require.Nilf(
		t,
		pCtx,
		"MySQL=%+v\nPostgreSQL=%+v\nRedis=%+v\nMongoDB=%+v",
		lo.TernaryF(pCtx == nil, func() error { return nil }, func() error { return pCtx.MySQLPingErr }),
		lo.TernaryF(pCtx == nil, func() error { return nil }, func() error { return pCtx.PostgreSQLErr }),
		lo.TernaryF(pCtx == nil, func() error { return nil }, func() error { return pCtx.RedisErr }),
		lo.TernaryF(pCtx == nil, func() error { return nil }, func() error { return pCtx.MongoErr }),
	)
	mysqlTables, postgresTables, err := db.ListRemainingTables(context.Background())
	require.NoError(t, err)
	t.Logf("MySQL Tables: %+v", mysqlTables)
	t.Logf("PostgreSQL Tables: %+v", postgresTables)

	require.NoError(t, db.DeleteAllTablesInRelationalDatabases(context.Background()))

	// Independent check: don't just trust the function's own return value —
	// separately re-query both catalogs ourselves and confirm nothing survived.
	mysqlTables, postgresTables, err = db.ListRemainingTables(context.Background())
	require.NoError(t, err)
	require.Emptyf(t, mysqlTables, "expected no MySQL tables to remain, found: %v", mysqlTables)
	require.Emptyf(t, postgresTables, "expected no Postgres tables to remain, found: %v", postgresTables)
}

func Test_RedisGet(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)
	require.NotNil(t, db)

	pCtx := db.PingContextFunction(context.Background())
	require.Nilf(
		t,
		pCtx,
		"MySQL=%+v\nPostgreSQL=%+v\nRedis=%+v\nMongoDB=%+v",
		lo.TernaryF(pCtx == nil, func() error { return nil }, func() error { return pCtx.MySQLPingErr }),
		lo.TernaryF(pCtx == nil, func() error { return nil }, func() error { return pCtx.PostgreSQLErr }),
		lo.TernaryF(pCtx == nil, func() error { return nil }, func() error { return pCtx.RedisErr }),
		lo.TernaryF(pCtx == nil, func() error { return nil }, func() error { return pCtx.MongoErr }),
	)

	// Set via REDIS_TEST_KEY so ad-hoc lookups (e.g. "otp:verification:<email>"
	// to grab a real OTP during manual QA) don't require editing this file.
	key := os.Getenv("REDIS_TEST_KEY")
	value, err := db.RedisRepository.Get(context.Background(), key)
	require.NoError(t, err)
	t.Logf("Value:%s", value)
}

// Test_RedisDelete clears a single Redis key during manual QA - e.g. a rate
// limit counter (redis_rate's go-redis/redis_rate/v10 stores its GCRA state
// under "rate:" + the key passed to Limiter.Allow, so clearing
// "rate:ratelimit:signup:<client ip>" resets the signup-per-IP limit)
// without needing to wait out the real window. Set via REDIS_TEST_KEY, same
// pattern as Test_RedisGet.
func Test_RedisDelete(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)
	require.NotNil(t, db)

	pCtx := db.PingContextFunction(context.Background())
	require.Nilf(
		t,
		pCtx,
		"MySQL=%+v\nPostgreSQL=%+v\nRedis=%+v\nMongoDB=%+v",
		lo.TernaryF(pCtx == nil, func() error { return nil }, func() error { return pCtx.MySQLPingErr }),
		lo.TernaryF(pCtx == nil, func() error { return nil }, func() error { return pCtx.PostgreSQLErr }),
		lo.TernaryF(pCtx == nil, func() error { return nil }, func() error { return pCtx.RedisErr }),
		lo.TernaryF(pCtx == nil, func() error { return nil }, func() error { return pCtx.MongoErr }),
	)

	key := os.Getenv("REDIS_TEST_KEY")
	require.NoError(t, db.RedisRepository.Delete(context.Background(), key))
	t.Logf("Deleted:%s", key)
}

// Test_RedisFlushDB wipes the whole logical Redis DB (OTPs, rate-limit
// counters, locks - everything RedisRepository stores is ephemeral cache,
// never a system of record per the comment on FlushDB) rather than a single
// key like Test_RedisDelete above. Run manually during QA to reset every
// user/IP's rate limit and clear stale OTPs in one shot.
func Test_RedisFlushDB(t *testing.T) {
	db, err := database.NewDatabase()
	require.NoError(t, err)
	require.NotNil(t, db)

	ctx := context.Background()

	// Sanity-check FlushDB actually did something, using only the
	// Repository interface (no raw client exposed to this test package).
	key := "test:flushdb:" + utils.GenerateValidULID()
	require.NoError(t, db.RedisRepository.Set(ctx, key, "value"))
	_, err = db.RedisRepository.Get(ctx, key)
	require.NoError(t, err, "sanity check: key must exist before flush")

	require.NoError(t, db.RedisRepository.FlushDB(ctx))

	_, err = db.RedisRepository.Get(ctx, key)
	require.Error(t, err, "expected key to be gone after FlushDB")
	t.Log("Redis FlushDB complete: all keys cleared")
}

// Test_CleanupLeakedTestMusicData removes music documents left behind by
// musicRepo's own tests (uniqueTitle in music_test.go stamps every fixture
// as "TEST_MUSIC_<nanos>_<suffix>" - see the comment on that helper: the
// music collection intentionally has no per-test cleanup hook). Those
// runs insert real documents into the shared Mongo collection and are
// never deleted, so they leak into GET /music. MusicRepository has no
// delete method by design (only GetAllMusicDetails/GetMusicByID/
// InsertMusicDetails exist on purpose), so this connects to Mongo
// directly - same manual-cleanup pattern as Test_RedisDelete above - to
// purge them.
func Test_CleanupLeakedTestMusicData(t *testing.T) {
	cfg, err := config.GetConfig()
	require.NoError(t, err)

	ctx := context.Background()
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoDBURI).SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, client.Disconnect(ctx))
	}()

	// "defaultdb"/"music" mirror the unexported mongoDatabaseName/
	// mongoMusicCollection constants in database.go - this test package
	// can't reference those directly.
	collection := client.Database("defaultdb").Collection("music")
	filter := bson.M{"title": bson.M{"$regex": "^TEST_MUSIC_"}}

	cursor, err := collection.Find(ctx, filter)
	require.NoError(t, err)
	var leaked []bson.M
	require.NoError(t, cursor.All(ctx, &leaked))
	for _, doc := range leaked {
		t.Logf("deleting leaked test music doc: id=%v title=%v", doc["_id"], doc["title"])
	}

	result, err := collection.DeleteMany(ctx, filter)
	require.NoError(t, err)
	t.Logf("deleted %d leaked TEST_MUSIC_* documents", result.DeletedCount)

	remaining, err := collection.CountDocuments(ctx, filter)
	require.NoError(t, err)
	require.Zerof(t, remaining, "expected no TEST_MUSIC_* documents to remain after cleanup")
}
