package redis

import "github.com/go-redis/redis/v8"

type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func NewRedisClient(conf *Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     conf.Host + ":" + conf.Port,
		Password: conf.Password,
		DB:       conf.DB,
	})

	ctx := client.Context()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return client, nil
}
