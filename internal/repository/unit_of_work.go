package repository

import (
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
}

type factory struct {
	redisClient *redis.Client
	db          *gorm.DB
}

func NewUnitOfWork(sm *ShardManager, rdb *redis.Client) UnitOfWork {
	return &unitOfWork{
		shardManager: sm,
		redisClient:  rdb,
	}
}

func (f *unitOfWork) ExecuteTx(shardingKey string, fn func(Factory) error) error {
	db := f.shardManager.GetShard(shardingKey)
	return db.Transaction(func(tx *gorm.DB) error {
		txFactory := &factory{
			db:          tx,
			redisClient: f.redisClient,
		}
		return fn(txFactory)
	})
}

func (f *unitOfWork) URLS(shardingKey string) URLRepository {
	db := f.shardManager.GetShard(shardingKey)
	pgRepo := NewPostgresURLRepository(db)
	return NewCachedURLRepository(pgRepo, f.redisClient)
}

func (f *factory) URLS(shardingKey string) URLRepository {
	pgRepo := NewPostgresURLRepository(f.db)
	return NewCachedURLRepository(pgRepo, f.redisClient)
}
