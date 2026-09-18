package cache

import (
	"context"
	"testing"
	"time"

	"usermanagementsystem/internal/model"
)

func TestRedisUserCache(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     "6380",
		Password: "",
		DB:       1, // Use DB 1 for testing
	}

	userCache, err := NewRedisUserCache(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	ctx := context.Background()

	// 1. Check Ping
	if err := userCache.Ping(ctx); err != nil {
		t.Fatalf("Redis Ping failed: %v", err)
	}

	testUser := &model.User{
		ID:     999,
		Name:   "Redis Test User",
		Email:  "redis.test@company.com",
		Status: "active",
	}

	// 2. Test Cache Miss on non-existent key
	_, err = userCache.GetUser(ctx, testUser.ID)
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss, got: %v", err)
	}

	// 3. Test SetUser with TTL
	ttl := 2 * time.Second
	if err := userCache.SetUser(ctx, testUser, ttl); err != nil {
		t.Fatalf("Failed to cache user: %v", err)
	}

	// 4. Test Cache Hit
	cached, err := userCache.GetUser(ctx, testUser.ID)
	if err != nil {
		t.Fatalf("Expected Cache Hit, got error: %v", err)
	}
	if cached.ID != testUser.ID || cached.Email != testUser.Email {
		t.Errorf("Cached user mismatch: %+v vs %+v", cached, testUser)
	}

	// 5. Test Cache Invalidation (DeleteUser)
	if err := userCache.DeleteUser(ctx, testUser.ID); err != nil {
		t.Fatalf("Failed to invalidate cache: %v", err)
	}

	// 6. Test Cache Miss after invalidation
	_, err = userCache.GetUser(ctx, testUser.ID)
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss after deletion, got: %v", err)
	}

	t.Log(" All Redis Cache operations (Hit, Miss, Set, TTL, Invalidation) passed!")
}
