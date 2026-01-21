package repository

import (
	"fmt"
	"hash/fnv"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ShardManager struct {
	shards []*gorm.DB
}

func NewShardManager(dsns []string) (*ShardManager, error) {
	var conns []*gorm.DB

	for i, dsn := range dsns {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to shard %d: %w", i, err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("failed to get sql.DB from shard %d: %w", i, err)
		}

		if err := sqlDB.Ping(); err != nil {
			return nil, fmt.Errorf("shard %d unreachable: %w", i, err)
		}

		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)

		conns = append(conns, db)
		log.Printf("connected to shard %d (GORM)", i)
	}

	return &ShardManager{shards: conns}, nil
}

func (sm *ShardManager) GetShard(idx int) *gorm.DB {
	return sm.shards[idx]
}

func (sm *ShardManager) GetShardIndex(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	hashValue := h.Sum32()

	return int(hashValue % uint32(len(sm.shards)))
}
