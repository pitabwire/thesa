package command

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pitabwire/thesa/model"
)

// IdempotencyStore provides deduplication for command execution.
// The key format is "idem:{commandId}:{key}".
type IdempotencyStore interface {
	// Check looks up a previous result by key. If the key exists and the
	// input hash matches, it returns the cached result. If the key exists
	// but the hash differs, it returns a 409 conflict error.
	Check(ctx context.Context, key string, inputHash string) (result *model.CommandResponse, found bool, err error)

	// Store saves a command result keyed by the idempotency key with a TTL.
	Store(ctx context.Context, key string, inputHash string, result model.CommandResponse, ttl time.Duration) error
}

// idempotencyEntry is the stored value for an idempotency key.
type idempotencyEntry struct {
	InputHash string               `json:"input_hash"`
	Result    model.CommandResponse `json:"result"`
}

// --- MemoryIdempotencyStore ---

// MemoryIdempotencyStore is an in-memory IdempotencyStore with TTL support.
// Suitable for testing and single-instance deployments.
type MemoryIdempotencyStore struct {
	mu      sync.RWMutex
	entries map[string]*memEntry
}

type memEntry struct {
	data      idempotencyEntry
	expiresAt time.Time
}

// NewMemoryIdempotencyStore creates a new in-memory idempotency store.
func NewMemoryIdempotencyStore() *MemoryIdempotencyStore {
	return &MemoryIdempotencyStore{
		entries: make(map[string]*memEntry),
	}
}

// Check looks up a cached result. Returns conflict error if input hash differs.
func (s *MemoryIdempotencyStore) Check(_ context.Context, key string, inputHash string) (*model.CommandResponse, bool, error) {
	s.mu.RLock()
	entry, exists := s.entries[key]
	s.mu.RUnlock()

	if !exists {
		return nil, false, nil
	}

	// Check TTL.
	if time.Now().After(entry.expiresAt) {
		s.mu.Lock()
		delete(s.entries, key)
		s.mu.Unlock()
		return nil, false, nil
	}

	// Input hash mismatch → conflict.
	if entry.data.InputHash != inputHash {
		return nil, true, model.NewConflictError(
			fmt.Sprintf("idempotency key %q already used with different input", key),
		)
	}

	result := entry.data.Result
	return &result, true, nil
}

// Store saves a result with TTL.
func (s *MemoryIdempotencyStore) Store(_ context.Context, key string, inputHash string, result model.CommandResponse, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[key] = &memEntry{
		data: idempotencyEntry{
			InputHash: inputHash,
			Result:    result,
		},
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// Len returns the number of entries (including expired ones). For testing.
func (s *MemoryIdempotencyStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// --- RedisIdempotencyStore ---

// RedisIdempotencyStore is a Redis-backed IdempotencyStore with TTL.
type RedisIdempotencyStore struct {
	client redis.Cmdable
}

// NewRedisIdempotencyStore creates a new Redis-backed idempotency store.
func NewRedisIdempotencyStore(client redis.Cmdable) *RedisIdempotencyStore {
	return &RedisIdempotencyStore{client: client}
}

// Check looks up a cached result in Redis. Returns conflict error if input hash differs.
func (s *RedisIdempotencyStore) Check(ctx context.Context, key string, inputHash string) (*model.CommandResponse, bool, error) {
	raw, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("redis get %q: %w", key, err)
	}

	var entry idempotencyEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, false, fmt.Errorf("unmarshal idempotency entry %q: %w", key, err)
	}

	// Input hash mismatch → conflict.
	if entry.InputHash != inputHash {
		return nil, true, model.NewConflictError(
			fmt.Sprintf("idempotency key %q already used with different input", key),
		)
	}

	return &entry.Result, true, nil
}

// Store saves a result in Redis with TTL.
func (s *RedisIdempotencyStore) Store(ctx context.Context, key string, inputHash string, result model.CommandResponse, ttl time.Duration) error {
	entry := idempotencyEntry{
		InputHash: inputHash,
		Result:    result,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal idempotency entry: %w", err)
	}

	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set %q: %w", key, err)
	}
	return nil
}

// FormatIdempotencyKey builds the standard idempotency key.
func FormatIdempotencyKey(commandID, key string) string {
	return fmt.Sprintf("idem:%s:%s", commandID, key)
}
