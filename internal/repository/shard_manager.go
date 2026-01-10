package repository

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"log"

	_ "github.com/lib/pq"
)

type ShardManager struct {
	shards []*sql.DB // Slice of connections: [db0, db1, db2]
}

func NewShardManager(dsns []string) (*ShardManager, error) {
	var conns []*sql.DB

	for i, dsn := range dsns {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to the shard %d: %w", i, err)
		}

		if err := db.Ping(); err != nil {
			return nil, fmt.Errorf("shard %d unreacheble: %w", i, err)
		}

		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)

		conns = append(conns, db)
		log.Printf("connected to the shard %d", i)
	}

	return &ShardManager{shards: conns}, nil
}

func (sm *ShardManager) GetShard(key string) *sql.DB {
	h := fnv.New32a()
	h.Write([]byte(key))
	hashValue := h.Sum32()

	shardIndex := hashValue % uint32(len(sm.shards))

	return sm.shards[shardIndex]
}
