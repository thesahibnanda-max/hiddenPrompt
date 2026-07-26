package config

import (
	"sync"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	GroqAPIKeys      []string
	GeminiAPIKeys    []string
	PostgreSQLConfig *pgx.ConnConfig
	MySQLConfig      *mysql.Config
	RedisClient      redis.Cmdable
	MailConnections  []MailConnectionConfig
	MongoDBURI       string
}

var (
	once    sync.Once = sync.Once{}
	config  Config    = Config{}
	onceErr error     = nil
)

func GetConfig() (Config, error) {
	once.Do(func() {
		groqKeys, groqErr := groqConfigKeys()
		if groqErr != nil {
			onceErr = groqErr
			return
		} else {
			config.GroqAPIKeys = groqKeys
		}
		geminiKeys, geminiErr := geminiConfigKeys()
		if geminiErr != nil {
			onceErr = geminiErr
			return
		} else {
			config.GeminiAPIKeys = geminiKeys
		}
		postgreSQLCfg, postErr := postgresqlConfig()
		if postErr != nil {
			onceErr = postErr
			return
		} else {
			config.PostgreSQLConfig = postgreSQLCfg
		}
		mysqlCfg, mysqlErr := mySQLConfigInternal()
		if mysqlErr != nil {
			onceErr = mysqlErr
			return
		} else {
			config.MySQLConfig = mysqlCfg
		}
		redisCl, redisErr := redisClient()
		if redisErr != nil {
			onceErr = redisErr
			return
		} else {
			config.RedisClient = redisCl
		}
		mailCfg, mailErr := mailConnConfig()
		if mailErr != nil {
			onceErr = mailErr
			return
		} else {
			config.MailConnections = mailCfg
		}
		mongoDBStringURI, mongoErr := getMongoClientURI()
		if mongoErr != nil {
			onceErr = mongoErr
			return
		} else {
			config.MongoDBURI = mongoDBStringURI
		}
	})
	return config, onceErr
}
