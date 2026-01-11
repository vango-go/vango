package session

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type mockRedisStatusCmd struct{ err error }

func (c mockRedisStatusCmd) Err() error { return c.err }

type mockRedisStringCmd struct {
	data []byte
	err  error
}

func (c mockRedisStringCmd) Bytes() ([]byte, error) { return c.data, c.err }
func (c mockRedisStringCmd) Err() error             { return c.err }

type mockRedisIntCmd struct{ err error }

func (c mockRedisIntCmd) Err() error { return c.err }

type mockRedisBoolCmd struct{ err error }

func (c mockRedisBoolCmd) Err() error { return c.err }

type mockRedisPipeline struct {
	mu   sync.Mutex
	sets []mockRedisSetCall
	err  error
}

type mockRedisSetCall struct {
	key        string
	value      interface{}
	expiration time.Duration
}

func (p *mockRedisPipeline) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) RedisStatusCmd {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sets = append(p.sets, mockRedisSetCall{key: key, value: value, expiration: expiration})
	return mockRedisStatusCmd{}
}

func (p *mockRedisPipeline) Exec(ctx context.Context) ([]interface{}, error) {
	return nil, p.err
}

type mockRedisClient struct {
	mu sync.Mutex

	sets    []mockRedisSetCall
	gets    []string
	dels    [][]string
	expires []mockRedisExpireCall

	getResp map[string]mockRedisStringCmd

	pipe *mockRedisPipeline
}

type mockRedisExpireCall struct {
	key        string
	expiration time.Duration
}

func (c *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) RedisStatusCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sets = append(c.sets, mockRedisSetCall{key: key, value: value, expiration: expiration})
	return mockRedisStatusCmd{}
}

func (c *mockRedisClient) Get(ctx context.Context, key string) RedisStringCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gets = append(c.gets, key)
	if resp, ok := c.getResp[key]; ok {
		return resp
	}
	return mockRedisStringCmd{err: ErrRedisNil}
}

func (c *mockRedisClient) Del(ctx context.Context, keys ...string) RedisIntCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dels = append(c.dels, keys)
	return mockRedisIntCmd{}
}

func (c *mockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) RedisBoolCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.expires = append(c.expires, mockRedisExpireCall{key: key, expiration: expiration})
	return mockRedisBoolCmd{}
}

func (c *mockRedisClient) Pipeline() RedisPipeliner {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pipe == nil {
		c.pipe = &mockRedisPipeline{}
	}
	return c.pipe
}

func (c *mockRedisClient) Close() error { return nil }

func TestRedisStore_PrefixAndKeying(t *testing.T) {
	client := &mockRedisClient{}
	store := NewRedisStore(client, WithRedisPrefix("pfx:"))

	if store.Prefix() != "pfx:" {
		t.Fatalf("Prefix() got %q", store.Prefix())
	}
	if store.key("abc") != "pfx:abc" {
		t.Fatalf("key() got %q", store.key("abc"))
	}
}

func TestRedisStore_Save_ExpiredDeletes(t *testing.T) {
	client := &mockRedisClient{}
	store := NewRedisStore(client)

	err := store.Save(context.Background(), "s1", []byte("x"), time.Now().Add(-time.Second))
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.dels) != 1 {
		t.Fatalf("Del calls got %d want 1", len(client.dels))
	}
	if got := client.dels[0][0]; got != "vango:session:s1" {
		t.Fatalf("Del key got %q", got)
	}
	if len(client.sets) != 0 {
		t.Fatalf("Set calls got %d want 0", len(client.sets))
	}
}

func TestRedisStore_Load_MissingReturnsNilData(t *testing.T) {
	client := &mockRedisClient{
		getResp: map[string]mockRedisStringCmd{
			"vango:session:s1": {err: errors.New("redis: nil")},
		},
	}
	store := NewRedisStore(client)

	data, err := store.Load(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if data != nil {
		t.Fatalf("Load() got %v want nil", data)
	}
}

func TestRedisStore_Touch_ExpiredDeletes(t *testing.T) {
	client := &mockRedisClient{}
	store := NewRedisStore(client)

	err := store.Touch(context.Background(), "s1", time.Now().Add(-time.Second))
	if err != nil {
		t.Fatalf("Touch() error: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.dels) != 1 {
		t.Fatalf("Del calls got %d want 1", len(client.dels))
	}
	if len(client.expires) != 0 {
		t.Fatalf("Expire calls got %d want 0", len(client.expires))
	}
}

func TestRedisStore_SaveAll_SkipsExpiredAndPipelines(t *testing.T) {
	client := &mockRedisClient{}
	store := NewRedisStore(client)

	now := time.Now()
	err := store.SaveAll(context.Background(), map[string]SessionData{
		"alive":  {Data: []byte("a"), ExpiresAt: now.Add(time.Minute)},
		"stale1": {Data: []byte("b"), ExpiresAt: now.Add(-time.Second)},
	})
	if err != nil {
		t.Fatalf("SaveAll() error: %v", err)
	}

	p := client.pipe
	if p == nil {
		t.Fatal("Pipeline() was not used")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.sets) != 1 {
		t.Fatalf("pipeline sets got %d want 1", len(p.sets))
	}
	if p.sets[0].key != "vango:session:alive" {
		t.Fatalf("pipeline key got %q", p.sets[0].key)
	}
}

func TestRedisStore_Close_MakesOperationsFail(t *testing.T) {
	client := &mockRedisClient{}
	store := NewRedisStore(client)
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	if err := store.Save(context.Background(), "s", []byte("x"), time.Now().Add(time.Minute)); err == nil {
		t.Fatal("Save() expected error after Close, got nil")
	}
	if _, err := store.Load(context.Background(), "s"); err == nil {
		t.Fatal("Load() expected error after Close, got nil")
	}
	if err := store.Delete(context.Background(), "s"); err == nil {
		t.Fatal("Delete() expected error after Close, got nil")
	}
	if err := store.Touch(context.Background(), "s", time.Now().Add(time.Minute)); err == nil {
		t.Fatal("Touch() expected error after Close, got nil")
	}
	if err := store.SaveAll(context.Background(), map[string]SessionData{}); err == nil {
		t.Fatal("SaveAll() expected error after Close, got nil")
	}
}

