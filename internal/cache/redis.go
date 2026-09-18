package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"usermanagementsystem/internal/model"
)

// ErrCacheMiss is returned when the key is not found in Redis.
var ErrCacheMiss = errors.New("cache miss")

// UserCache defines caching operations for user entities.
type UserCache interface {
	GetUser(ctx context.Context, id int) (*model.User, error)
	SetUser(ctx context.Context, user *model.User, ttl time.Duration) error
	DeleteUser(ctx context.Context, id int) error
	Ping(ctx context.Context) error
}

type redisUserCache struct {
	client *redis.Client
}

// Config specifies connection settings for Redis.
type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// NewRedisUserCache instantiates a new Redis client and checks connectivity.
func NewRedisUserCache(cfg Config) (UserCache, error) {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", addr, err)
	}

	return &redisUserCache{client: client}, nil
}

func (c *redisUserCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *redisUserCache) userKey(id int) string {
	return fmt.Sprintf("user:%d", id)
}

func (c *redisUserCache) GetUser(ctx context.Context, id int) (*model.User, error) {
	key := c.userKey(id)
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			fmt.Printf("[CACHE] MISS %s\n", key)
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	fmt.Printf("[CACHE] HIT %s (Returning from RAM)\n", key)
	var user model.User
	if err := json.Unmarshal([]byte(val), &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached user: %w", err)
	}

	return &user, nil
}

func (c *redisUserCache) SetUser(ctx context.Context, user *model.User, ttl time.Duration) error {
	if user == nil {
		return nil
	}

	key := c.userKey(user.ID)
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user for caching: %w", err)
	}

	fmt.Printf("[CACHE] SET %s (TTL: %v)\n", key, ttl)
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *redisUserCache) DeleteUser(ctx context.Context, id int) error {
	key := c.userKey(id)
	fmt.Printf("[CACHE] INVALIDATE %s\n", key)
	return c.client.Del(ctx, key).Err()
}
