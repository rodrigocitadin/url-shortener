package repository

import "gorm.io/gorm"

type Factory interface {
	URLS(shardingKey string) *URLRepository
}

type UnitOfWork interface {
	ExecuteTx(shardingKey string, fn func(Factory) error) error
	URLS(shardingKey string) *URLRepository
}

type unitOfWork struct {
	shardManager *ShardManager
	db           *gorm.DB
}

type factory struct {
	shardManager *ShardManager
	db           *gorm.DB
}

func NewUnitOfWork(sm *ShardManager) UnitOfWork {
	return &unitOfWork{shardManager: sm}
}

func (f *unitOfWork) ExecuteTx(shardingKey string, fn func(Factory) error) error {
	db := f.shardManager.GetShard(shardingKey)
	return db.Transaction(func(tx *gorm.DB) error {
		txFactory := &factory{db: tx}
		return fn(txFactory)
	})
}

func (f *unitOfWork) URLS(shardingKey string) *URLRepository {
	db := f.shardManager.GetShard(shardingKey)
	return NewURLRepository(db)
}

func (f *factory) URLS(shardingKey string) *URLRepository {
	return NewURLRepository(f.db)
}
