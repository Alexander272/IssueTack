package redis_repo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

type SessionCacheRepo struct {
	client *redis.Client
	ttl    time.Duration
}

func NewSessionCacheRepo(client *redis.Client, ttl time.Duration) *SessionCacheRepo {
	return &SessionCacheRepo{client: client, ttl: ttl}
}

type SessionCache interface {
	Get(ctx context.Context, userId string) map[string][]string
	Set(ctx context.Context, userId string, perms map[string][]string)
	Del(ctx context.Context, userId string)
	Flush(ctx context.Context)
}

func (c *SessionCacheRepo) key(userId string) string {
	serviceName := os.Getenv("SERVICE_ID")
	return fmt.Sprintf("%s:session:permissions:%s", serviceName, userId)
}

func (c *SessionCacheRepo) Get(ctx context.Context, userId string) map[string][]string {
	data, err := c.client.Get(ctx, c.key(userId)).Bytes()
	if err != nil {
		return nil
	}

	var perms map[string][]string
	if err := json.Unmarshal(data, &perms); err != nil {
		return nil
	}
	return perms
}

func (c *SessionCacheRepo) Set(ctx context.Context, userId string, perms map[string][]string) {
	data, err := json.Marshal(perms)
	if err != nil {
		return
	}
	c.client.Set(ctx, c.key(userId), data, c.ttl)
}

func (c *SessionCacheRepo) Del(ctx context.Context, userId string) {
	c.client.Del(ctx, c.key(userId))
}

func (c *SessionCacheRepo) Flush(ctx context.Context) {
	serviceName := os.Getenv("SERVICE_ID")
	key := fmt.Sprintf("%s:session:permissions:*", serviceName)

	iter := c.client.Scan(ctx, 0, key, 0).Iterator()
	for iter.Next(ctx) {
		c.client.Del(ctx, iter.Val())
	}
}
