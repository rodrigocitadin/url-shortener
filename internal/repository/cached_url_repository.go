package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rodrigocitadin/url-shortener/internal/entities"
)

type cachedURLRepository struct {
	next  URLRepository
	redis *redis.Client
}

const (
	cacheTTL = 1
	cacheKey = "url:"
)

func NewCachedURLRepository(next URLRepository, redisClient *redis.Client) URLRepository {
	return &cachedURLRepository{
		next:  next,
		redis: redisClient,
	}
}

func (r *cachedURLRepository) Find(shortCode string) (*entities.URLEntity, error) {
	ctx := context.Background() // remove this later
	key := cacheKey + shortCode

	val, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var entity entities.URLEntity
		if json.Unmarshal([]byte(val), &entity) == nil {
			return &entity, nil
		}
	}

	entity, err := r.next.Find(shortCode)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(entity); err == nil {
		r.redis.Set(ctx, key, data, cacheTTL*time.Hour)
	}

	return entity, nil
}

func (r *cachedURLRepository) Save(urlEntity *entities.URLEntity) error {
	if err := r.next.Save(urlEntity); err != nil {
		return err
	}

	ctx := context.Background() // remove this later
	key := cacheKey + urlEntity.Shortcode

	if data, err := json.Marshal(urlEntity); err == nil {
		r.redis.Set(ctx, key, data, cacheTTL*time.Hour)
	}

	return nil
}
