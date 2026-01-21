package repository

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Factory interface {
	URLS(shardingKey string) URLRepository
}

type UnitOfWork interface {
	ExecuteTx(shardingKey string, fn func(Factory) error) error
	URLS(shardingKey string) URLRepository
}

type unitOfWork struct {
	shardManager *ShardManager
	redisClient  *redis.Client
	amqpChannel  *amqp.Channel
}

type factory struct {
	redisClient *redis.Client
	db          *gorm.DB
}

func NewUnitOfWork(sm *ShardManager, rdb *redis.Client, ch *amqp.Channel) UnitOfWork {
	return &unitOfWork{
		shardManager: sm,
		redisClient:  rdb,
		amqpChannel:  ch,
	}
}

func (f *unitOfWork) ExecuteTx(shardingKey string, fn func(Factory) error) error {
	db := f.shardManager.GetShard(f.shardManager.GetShardIndex(shardingKey))
	return db.Transaction(func(tx *gorm.DB) error {
		txFactory := &factory{
			db:          tx,
			redisClient: f.redisClient,
		}
		return fn(txFactory)
	})
}

func (f *unitOfWork) URLS(shardingKey string) URLRepository {
	db := f.shardManager.GetShard(f.shardManager.GetShardIndex(shardingKey))
	pgRepo := NewDatabaseURLRepository(db)

	var finalRepo URLRepository = pgRepo

	if f.amqpChannel != nil {
		finalRepo = NewQueueURLRepository(f.amqpChannel, pgRepo)
	}

	return NewCachedURLRepository(finalRepo, f.redisClient)
}

func (f *factory) URLS(shardingKey string) URLRepository {
	pgRepo := NewDatabaseURLRepository(f.db)
	return NewCachedURLRepository(pgRepo, f.redisClient)
}
