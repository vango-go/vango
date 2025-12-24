package session

import (
	"context"
	"errors"
	"time"
)

// RedisClient defines the interface for Redis operations.
// This interface is compatible with github.com/redis/go-redis/v9.
type RedisClient interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) RedisStatusCmd
	Get(ctx context.Context, key string) RedisStringCmd
	Del(ctx context.Context, keys ...string) RedisIntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) RedisBoolCmd
	Pipeline() RedisPipeliner
	Close() error
}

// RedisStatusCmd represents a Redis status command result.
type RedisStatusCmd interface {
	Err() error
}

// RedisStringCmd represents a Redis string command result.
type RedisStringCmd interface {
	Bytes() ([]byte, error)
	Err() error
}

// RedisIntCmd represents a Redis int command result.
type RedisIntCmd interface {
	Err() error
}

// RedisBoolCmd represents a Redis bool command result.
type RedisBoolCmd interface {
	Err() error
}

// RedisPipeliner represents a Redis pipeline.
type RedisPipeliner interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) RedisStatusCmd
	Exec(ctx context.Context) ([]interface{}, error)
}

// ErrRedisNil is returned when a key doesn't exist in Redis.
// This should match redis.Nil from go-redis.
var ErrRedisNil = errors.New("redis: nil")

// RedisStore is a Redis-backed session store.
// It's suitable for multi-server deployments with shared session state.
type RedisStore struct {
	client RedisClient
	prefix string
	closed bool
}

// RedisStoreOption configures RedisStore behavior.
type RedisStoreOption func(*redisStoreConfig)

type redisStoreConfig struct {
	prefix string
}

// WithRedisPrefix sets the key prefix for session keys.
// Default: "vango:session:".
func WithRedisPrefix(prefix string) RedisStoreOption {
	return func(c *redisStoreConfig) {
		c.prefix = prefix
	}
}

// NewRedisStore creates a new Redis-backed session store.
func NewRedisStore(client RedisClient, opts ...RedisStoreOption) *RedisStore {
	cfg := &redisStoreConfig{
		prefix: "vango:session:",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return &RedisStore{
		client: client,
		prefix: cfg.prefix,
	}
}

// key returns the Redis key for a session ID.
func (r *RedisStore) key(sessionID string) string {
	return r.prefix + sessionID
}

// Save stores session data with an expiration time.
func (r *RedisStore) Save(ctx context.Context, sessionID string, data []byte, expiresAt time.Time) error {
	if r.closed {
		return ErrStoreClosed{}
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		// Already expired, delete instead
		return r.Delete(ctx, sessionID)
	}

	return r.client.Set(ctx, r.key(sessionID), data, ttl).Err()
}

// Load retrieves session data if it exists.
func (r *RedisStore) Load(ctx context.Context, sessionID string) ([]byte, error) {
	if r.closed {
		return nil, ErrStoreClosed{}
	}

	data, err := r.client.Get(ctx, r.key(sessionID)).Bytes()
	if err != nil {
		// Check for nil (key doesn't exist)
		if err.Error() == ErrRedisNil.Error() || err.Error() == "redis: nil" {
			return nil, nil
		}
		return nil, err
	}

	return data, nil
}

// Delete removes a session from Redis.
func (r *RedisStore) Delete(ctx context.Context, sessionID string) error {
	if r.closed {
		return ErrStoreClosed{}
	}

	return r.client.Del(ctx, r.key(sessionID)).Err()
}

// Touch updates the expiration time for a session.
func (r *RedisStore) Touch(ctx context.Context, sessionID string, expiresAt time.Time) error {
	if r.closed {
		return ErrStoreClosed{}
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return r.Delete(ctx, sessionID)
	}

	return r.client.Expire(ctx, r.key(sessionID), ttl).Err()
}

// SaveAll saves multiple sessions using a Redis pipeline.
func (r *RedisStore) SaveAll(ctx context.Context, sessions map[string]SessionData) error {
	if r.closed {
		return ErrStoreClosed{}
	}

	if len(sessions) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for id, sd := range sessions {
		ttl := time.Until(sd.ExpiresAt)
		if ttl > 0 {
			pipe.Set(ctx, r.key(id), sd.Data, ttl)
		}
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Close marks the store as closed.
// Note: This does not close the underlying Redis client,
// as it may be shared with other components.
func (r *RedisStore) Close() error {
	r.closed = true
	return nil
}

// Prefix returns the current key prefix.
// This is for testing/debugging purposes.
func (r *RedisStore) Prefix() string {
	return r.prefix
}
