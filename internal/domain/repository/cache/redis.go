package cache

import (
	"context"
	"encoding/json"
	"time"

	"order-service/config"
	"order-service/internal/domain/entities"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisCache struct {
	rdb    *redis.Client
	ttl    time.Duration
	prefix string
	log    *zap.SugaredLogger
}

func NewRedisCache(cfg *config.ConfigModel, l *zap.Logger) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	log := l.Named("redis.cache").Sugar()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Warnw("redis ping failed", "addr", cfg.Redis.Addr, "error", err)
	} else {
		log.Infow("redis connected", "addr", cfg.Redis.Addr, "db", cfg.Redis.DB)
	}

	var ttl time.Duration
	if cfg.Redis.TTLSeconds > 0 {
		ttl = time.Duration(cfg.Redis.TTLSeconds) * time.Second
	}

	prefix := cfg.Redis.KeyPrefix
	if prefix == "" {
		prefix = "orders:"
	}

	return &RedisCache{
		rdb:    rdb,
		ttl:    ttl,
		prefix: prefix,
		log:    log,
	}
}

func (c *RedisCache) key(id string) string { return c.prefix + id }


func (c *RedisCache) Get(id string) (*entities.Order, bool) {
	ctx := context.Background()
	data, err := c.rdb.Get(ctx, c.key(id)).Bytes()
	if err != nil {
		if err != redis.Nil {
			c.log.Warnw("redis get failed", "order_uid", id, "error", err)
		} else {
			c.log.Debugw("cache miss", "order_uid", id)
		}
		return nil, false
	}

	var o entities.Order
	if err := json.Unmarshal(data, &o); err != nil {
		c.log.Errorw("unmarshal failed", "order_uid", id, "error", err)
		return nil, false
	}
	c.log.Debugw("cache hit", "order_uid", id)
	return &o, true
}

func (c *RedisCache) Set(id string, order *entities.Order) {
	b, err := json.Marshal(order)
	if err != nil {
		c.log.Errorw("marshal failed", "order_uid", id, "error", err)
		return
	}
	ctx := context.Background()
	if err := c.rdb.Set(ctx, c.key(id), b, c.ttl).Err(); err != nil {
		c.log.Errorw("redis set failed", "order_uid", id, "error", err)
		return
	}
	c.log.Infow("cache set", "order_uid", id)
}
