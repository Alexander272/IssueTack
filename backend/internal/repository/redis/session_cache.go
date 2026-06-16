package redis_repo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/go-redis/redis/v8"
)

type SessionCacheRepo struct {
	client      *redis.Client
	ttl         time.Duration
	serviceName string
}

func NewSessionCacheRepo(client *redis.Client, ttl time.Duration) *SessionCacheRepo {
	return &SessionCacheRepo{
		client:      client,
		ttl:         ttl,
		serviceName: os.Getenv("SERVICE_ID"),
	}
}

type SessionCache interface {
	Get(ctx context.Context, userId string) map[string][]string
	Set(ctx context.Context, userId string, perms map[string][]string)
	Del(ctx context.Context, userId string)
	Flush(ctx context.Context)
}

func (c *SessionCacheRepo) key(userId string) string {
	return fmt.Sprintf("%s:session:permissions:%s", c.serviceName, userId)
}

func (c *SessionCacheRepo) Get(ctx context.Context, userId string) map[string][]string {
	data, err := c.client.Get(ctx, c.key(userId)).Bytes()
	if err != nil {
		if err != redis.Nil {
			logger.Warn("redis get error", logger.StringAttr("key", c.key(userId)), logger.ErrAttr(err))
		}
		return nil
	}

	var perms map[string][]string
	if err := json.Unmarshal(data, &perms); err != nil {
		logger.Warn("redis unmarshal error", logger.StringAttr("key", c.key(userId)), logger.ErrAttr(err))
		return nil
	}
	return perms
}

func (c *SessionCacheRepo) Set(ctx context.Context, userId string, perms map[string][]string) {
	data, err := json.Marshal(perms)
	if err != nil {
		logger.Warn("redis marshal error", logger.StringAttr("key", c.key(userId)), logger.ErrAttr(err))
		return
	}
	if err := c.client.Set(ctx, c.key(userId), data, c.ttl).Err(); err != nil {
		logger.Warn("redis set error", logger.StringAttr("key", c.key(userId)), logger.ErrAttr(err))
	}
}

func (c *SessionCacheRepo) Del(ctx context.Context, userId string) {
	if err := c.client.Del(ctx, c.key(userId)).Err(); err != nil {
		logger.Warn("redis del error", logger.StringAttr("key", c.key(userId)), logger.ErrAttr(err))
	}
}

func (c *SessionCacheRepo) Flush(ctx context.Context) {
	key := fmt.Sprintf("%s:session:permissions:*", c.serviceName)

	iter := c.client.Scan(ctx, 0, key, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			logger.Warn("redis flush del error", logger.StringAttr("key", iter.Val()), logger.ErrAttr(err))
		}
	}
	if err := iter.Err(); err != nil {
		logger.Warn("redis flush scan error", logger.ErrAttr(err))
	}
}
