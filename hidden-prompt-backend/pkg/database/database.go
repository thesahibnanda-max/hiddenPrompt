package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hidden-prompt-backend/pkg/config"
	musicRepo "hidden-prompt-backend/pkg/database/mongo/music"
	usersRepo "hidden-prompt-backend/pkg/database/mysql/users"
	puzzlesRepo "hidden-prompt-backend/pkg/database/psql/puzzles"
	redisRepo "hidden-prompt-backend/pkg/database/redis"
	"hidden-prompt-backend/pkg/utils"
	"sync"

	"github.com/bsm/redislock"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	redisRate "github.com/go-redis/redis_rate/v10"
)

const (
	mongoDatabaseName    = "defaultdb"
	mongoMusicCollection = "music"
)

type PingContextError struct {
	MySQLPingErr  error
	PostgreSQLErr error
	RedisErr      error
	MongoErr      error
}

type DatabaseParams struct {
	UsersRepository                      usersRepo.Repository
	PuzzlesRepository                    puzzlesRepo.Repository
	RedisRepository                      redisRepo.Repository
	MusicRepository                      musicRepo.Repository
	RedisRateLimiter                     *redisRate.Limiter
	RedisLocker                          *redislock.Client
	PingContextFunction                  func(ctx context.Context) *PingContextError
	DeleteAllTablesInRelationalDatabases func(ctx context.Context) error
	ListRemainingTables                  func(ctx context.Context) (mysqlTables []string, postgresTables []string, err error)
}

func NewDatabase() (*DatabaseParams, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	pgPoolConfig, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, err
	}
	*pgPoolConfig.ConnConfig = *cfg.PostgreSQLConfig

	pgPool, err := pgxpool.NewWithConfig(ctx, pgPoolConfig)
	if err != nil {
		return nil, err
	}

	mysqlDB, err := sql.Open("mysql", cfg.MySQLConfig.FormatDSN())
	if err != nil {
		return nil, err
	}
	redisDB := cfg.RedisClient

	mongoClient, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoDBURI).SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)))
	if err != nil {
		return nil, err
	}

	if pingErr := mongoClient.Ping(ctx, readpref.Primary()); pingErr != nil {
		return nil, pingErr
	}
	if pingErr := pgPool.Ping(ctx); pingErr != nil {
		return nil, pingErr
	}
	if pingErr := mysqlDB.PingContext(ctx); pingErr != nil {
		return nil, pingErr
	}
	if pingErr := redisDB.Ping(ctx).Err(); pingErr != nil {
		return nil, pingErr
	}

	uRepo, err := usersRepo.NewRepository(mysqlDB)
	if err != nil {
		return nil, err
	}

	pRepo, err := puzzlesRepo.NewRepo(pgPool)
	if err != nil {
		return nil, err
	}
	rRepo, err := redisRepo.NewRepository(redisDB, 0)
	if err != nil {
		return nil, err
	}
	mRepo, err := musicRepo.NewRepository(ctx, mongoClient.Database(mongoDatabaseName).Collection(mongoMusicCollection))
	if err != nil {
		return nil, err
	}

	rh := &rawHandles{mysqlDB: mysqlDB, pgPool: pgPool, redisDB: redisDB, mongoClient: mongoClient}

	return &DatabaseParams{
		MusicRepository:   mRepo,
		UsersRepository:   uRepo,
		PuzzlesRepository: pRepo,
		RedisRepository:   rRepo,
		RedisRateLimiter:  redisRate.NewLimiter(redisDB),
		RedisLocker:       redislock.New(redisDB),
		PingContextFunction: func(ctx context.Context) *PingContextError {
			return rh.pingAll(ctx)
		},
		DeleteAllTablesInRelationalDatabases: func(ctx context.Context) error {
			return rh.deleteAllTablesAndVerify(ctx)
		},
		ListRemainingTables: func(ctx context.Context) (mysqlTables []string, postgresTables []string, err error) {
			postgresTables, err = rh.listPostgresTables(ctx)
			if err != nil {
				return nil, nil, err
			}
			mysqlTables, err = rh.listMySQLTables(ctx)
			if err != nil {
				return nil, nil, err
			}
			return mysqlTables, postgresTables, nil
		},
	}, nil
}

type rawHandles struct {
	mysqlDB     *sql.DB
	pgPool      *pgxpool.Pool
	redisDB     redis.Cmdable
	mongoClient *mongo.Client
}

// pingAll checks all four backing stores concurrently rather than one
// after another - they're fully independent connections, so total
// latency is max(...) instead of sum(...). Each goroutine only ever
// writes to its own dedicated field of pingErr, so no synchronization is
// needed beyond the WaitGroup itself.
func (rh *rawHandles) pingAll(ctx context.Context) *PingContextError {
	pingErr := &PingContextError{}

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		var ignored int
		pingErr.MySQLPingErr = errors.Join(rh.mysqlDB.PingContext(ctx), rh.mysqlDB.QueryRowContext(ctx, "SELECT 1").Scan(&ignored))
	}()
	go func() {
		defer wg.Done()
		var ignored int
		pingErr.PostgreSQLErr = errors.Join(rh.pgPool.Ping(ctx), rh.pgPool.QueryRow(ctx, "SELECT 1").Scan(&ignored))
	}()
	go func() {
		defer wg.Done()
		id := "ignored:" + utils.GenerateValidULID()
		setErr := rh.redisDB.Set(ctx, id, utils.GenerateValidULID(), 0).Err()
		delErr := rh.redisDB.Del(ctx, id).Err()
		pingErr.RedisErr = errors.Join(rh.redisDB.Ping(ctx).Err(), setErr, delErr)
	}()
	go func() {
		defer wg.Done()
		pingErr.MongoErr = rh.mongoClient.Ping(ctx, readpref.Primary())
	}()

	wg.Wait()

	if pingErr.MySQLPingErr == nil && pingErr.PostgreSQLErr == nil && pingErr.RedisErr == nil && pingErr.MongoErr == nil {
		return nil
	}
	return pingErr
}

func (rh *rawHandles) deleteAllTablesAndVerify(ctx context.Context) error {
	if err := rh.dropAllPostgresTables(ctx); err != nil {
		return err
	}
	if err := rh.dropAllMySQLTables(ctx); err != nil {
		return err
	}

	postgresTables, err := rh.listPostgresTables(ctx)
	if err != nil {
		return fmt.Errorf("verify postgres tables dropped: %w", err)
	}
	if len(postgresTables) > 0 {
		return fmt.Errorf("postgres tables still present after drop: %v", postgresTables)
	}

	mysqlTables, err := rh.listMySQLTables(ctx)
	if err != nil {
		return fmt.Errorf("verify mysql tables dropped: %w", err)
	}
	if len(mysqlTables) > 0 {
		return fmt.Errorf("mysql tables still present after drop: %v", mysqlTables)
	}

	return nil
}

func (rh *rawHandles) dropAllPostgresTables(ctx context.Context) error {
	const pgDrop = `
DO $$
DECLARE
	r RECORD;
BEGIN
	FOR r IN
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
	LOOP
		EXECUTE format('DROP TABLE IF EXISTS %I CASCADE', r.tablename);
	END LOOP;
END $$;
`
	_, err := rh.pgPool.Exec(ctx, pgDrop)
	return err
}

func (rh *rawHandles) dropAllMySQLTables(ctx context.Context) error {
	if _, err := rh.mysqlDB.ExecContext(ctx, `SET FOREIGN_KEY_CHECKS = 0;`); err != nil {
		return err
	}

	tables, err := rh.listMySQLTables(ctx)
	if err != nil {
		return err
	}

	for _, tableName := range tables {
		if _, err := rh.mysqlDB.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName)); err != nil {
			return err
		}
	}

	if _, err := rh.mysqlDB.ExecContext(ctx, `SET FOREIGN_KEY_CHECKS = 1;`); err != nil {
		return err
	}

	return nil
}

func (rh *rawHandles) listPostgresTables(ctx context.Context) ([]string, error) {
	rows, err := rh.pgPool.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname = 'public'`)
	if err != nil {
		return nil, fmt.Errorf("list postgres tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("list postgres tables: %w", err)
		}
		tables = append(tables, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list postgres tables: %w", err)
	}

	return tables, nil
}

func (rh *rawHandles) listMySQLTables(ctx context.Context) ([]string, error) {
	rows, err := rh.mysqlDB.QueryContext(ctx, `
SELECT table_name
FROM information_schema.tables
WHERE table_schema = DATABASE()
`)
	if err != nil {
		return nil, fmt.Errorf("list mysql tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("list mysql tables: %w", err)
		}
		tables = append(tables, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list mysql tables: %w", err)
	}

	return tables, nil
}
