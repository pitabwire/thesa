package command

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/pitabwire/thesa/model"
)

func testCommandResponse() model.CommandResponse {
	return model.CommandResponse{
		Success: true,
		Message: "Order cancelled",
		Result: map[string]any{
			"id":     "ord-123",
			"status": "cancelled",
		},
	}
}

// --- MemoryIdempotencyStore ---

func TestMemoryIdempotencyStore_CheckNotFound(t *testing.T) {
	store := NewMemoryIdempotencyStore()

	result, found, err := store.Check(context.Background(), "idem:cmd:key1", "hash-abc")
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if found {
		t.Error("found = true, want false")
	}
	if result != nil {
		t.Errorf("result = %+v, want nil", result)
	}
}

func TestMemoryIdempotencyStore_StoreAndCheck(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	ctx := context.Background()
	key := "idem:orders.cancel:key1"
	hash := "hash-abc"
	resp := testCommandResponse()

	err := store.Store(ctx, key, hash, resp, 5*time.Minute)
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}

	result, found, err := store.Check(ctx, key, hash)
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if !result.Success {
		t.Error("result.Success = false")
	}
	if result.Message != "Order cancelled" {
		t.Errorf("result.Message = %q", result.Message)
	}
	if result.Result["id"] != "ord-123" {
		t.Errorf("result.Result[id] = %v", result.Result["id"])
	}
}

func TestMemoryIdempotencyStore_ConflictOnHashMismatch(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	ctx := context.Background()
	key := "idem:orders.cancel:key1"

	err := store.Store(ctx, key, "hash-abc", testCommandResponse(), 5*time.Minute)
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}

	// Same key, different hash → conflict.
	_, found, err := store.Check(ctx, key, "hash-different")
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !found {
		t.Error("found = false, want true (key exists)")
	}

	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T, want *model.ErrorEnvelope", err)
	}
	if envErr.Code != model.ErrConflict {
		t.Errorf("error code = %s, want %s", envErr.Code, model.ErrConflict)
	}
}

func TestMemoryIdempotencyStore_TTLExpiry(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	ctx := context.Background()
	key := "idem:cmd:key1"

	// Store with very short TTL.
	err := store.Store(ctx, key, "hash-abc", testCommandResponse(), 1*time.Millisecond)
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}

	// Wait for TTL to expire.
	time.Sleep(5 * time.Millisecond)

	result, found, err := store.Check(ctx, key, "hash-abc")
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if found {
		t.Error("found = true, want false (expired)")
	}
	if result != nil {
		t.Errorf("result = %+v, want nil (expired)", result)
	}
}

func TestMemoryIdempotencyStore_OverwriteExistingKey(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	ctx := context.Background()
	key := "idem:cmd:key1"

	resp1 := model.CommandResponse{Success: true, Message: "first"}
	resp2 := model.CommandResponse{Success: true, Message: "second"}

	_ = store.Store(ctx, key, "hash-1", resp1, 5*time.Minute)
	_ = store.Store(ctx, key, "hash-2", resp2, 5*time.Minute)

	result, found, err := store.Check(ctx, key, "hash-2")
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if !found {
		t.Fatal("found = false")
	}
	if result.Message != "second" {
		t.Errorf("result.Message = %q, want second", result.Message)
	}
}

func TestMemoryIdempotencyStore_Len(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	ctx := context.Background()

	if store.Len() != 0 {
		t.Errorf("Len() = %d, want 0", store.Len())
	}

	_ = store.Store(ctx, "key1", "h1", testCommandResponse(), 5*time.Minute)
	_ = store.Store(ctx, "key2", "h2", testCommandResponse(), 5*time.Minute)

	if store.Len() != 2 {
		t.Errorf("Len() = %d, want 2", store.Len())
	}
}

func TestMemoryIdempotencyStore_ExpiredEntryRemovedOnCheck(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	ctx := context.Background()
	key := "idem:cmd:key1"

	_ = store.Store(ctx, key, "hash-abc", testCommandResponse(), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	// Check should clean up the expired entry.
	_, _, _ = store.Check(ctx, key, "hash-abc")

	if store.Len() != 0 {
		t.Errorf("Len() = %d, want 0 (expired entry removed)", store.Len())
	}
}

// --- RedisIdempotencyStore ---

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return mr, client
}

func TestRedisIdempotencyStore_CheckNotFound(t *testing.T) {
	_, client := newTestRedis(t)
	store := NewRedisIdempotencyStore(client)

	result, found, err := store.Check(context.Background(), "idem:cmd:key1", "hash-abc")
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if found {
		t.Error("found = true, want false")
	}
	if result != nil {
		t.Errorf("result = %+v, want nil", result)
	}
}

func TestRedisIdempotencyStore_StoreAndCheck(t *testing.T) {
	_, client := newTestRedis(t)
	store := NewRedisIdempotencyStore(client)
	ctx := context.Background()
	key := "idem:orders.cancel:key1"
	hash := "hash-abc"
	resp := testCommandResponse()

	err := store.Store(ctx, key, hash, resp, 5*time.Minute)
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}

	result, found, err := store.Check(ctx, key, hash)
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if !result.Success {
		t.Error("result.Success = false")
	}
	if result.Message != "Order cancelled" {
		t.Errorf("result.Message = %q", result.Message)
	}
}

func TestRedisIdempotencyStore_ConflictOnHashMismatch(t *testing.T) {
	_, client := newTestRedis(t)
	store := NewRedisIdempotencyStore(client)
	ctx := context.Background()
	key := "idem:orders.cancel:key1"

	err := store.Store(ctx, key, "hash-abc", testCommandResponse(), 5*time.Minute)
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}

	_, found, err := store.Check(ctx, key, "hash-different")
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !found {
		t.Error("found = false, want true")
	}

	envErr, ok := err.(*model.ErrorEnvelope)
	if !ok {
		t.Fatalf("error type = %T, want *model.ErrorEnvelope", err)
	}
	if envErr.Code != model.ErrConflict {
		t.Errorf("error code = %s, want %s", envErr.Code, model.ErrConflict)
	}
}

func TestRedisIdempotencyStore_TTLExpiry(t *testing.T) {
	mr, client := newTestRedis(t)
	store := NewRedisIdempotencyStore(client)
	ctx := context.Background()
	key := "idem:cmd:key1"

	err := store.Store(ctx, key, "hash-abc", testCommandResponse(), 1*time.Second)
	if err != nil {
		t.Fatalf("Store error: %v", err)
	}

	// Fast-forward miniredis time past TTL.
	mr.FastForward(2 * time.Second)

	result, found, err := store.Check(ctx, key, "hash-abc")
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if found {
		t.Error("found = true, want false (expired)")
	}
	if result != nil {
		t.Errorf("result = %+v, want nil", result)
	}
}

func TestRedisIdempotencyStore_OverwriteExistingKey(t *testing.T) {
	_, client := newTestRedis(t)
	store := NewRedisIdempotencyStore(client)
	ctx := context.Background()
	key := "idem:cmd:key1"

	resp1 := model.CommandResponse{Success: true, Message: "first"}
	resp2 := model.CommandResponse{Success: true, Message: "second"}

	_ = store.Store(ctx, key, "hash-1", resp1, 5*time.Minute)
	_ = store.Store(ctx, key, "hash-2", resp2, 5*time.Minute)

	result, found, err := store.Check(ctx, key, "hash-2")
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if !found {
		t.Fatal("found = false")
	}
	if result.Message != "second" {
		t.Errorf("result.Message = %q, want second", result.Message)
	}
}

func TestRedisIdempotencyStore_PreservesResultFields(t *testing.T) {
	_, client := newTestRedis(t)
	store := NewRedisIdempotencyStore(client)
	ctx := context.Background()
	key := "idem:cmd:key1"

	resp := model.CommandResponse{
		Success: true,
		Message: "Done",
		Result:  map[string]any{"id": "123", "status": "ok"},
		Errors:  []model.FieldError{{Field: "name", Code: "REQUIRED", Message: "Name is required"}},
	}

	_ = store.Store(ctx, key, "hash", resp, 5*time.Minute)
	result, _, _ := store.Check(ctx, key, "hash")

	if result.Message != "Done" {
		t.Errorf("Message = %q", result.Message)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("len(Errors) = %d, want 1", len(result.Errors))
	}
	if result.Errors[0].Field != "name" {
		t.Errorf("Errors[0].Field = %q", result.Errors[0].Field)
	}
}

// --- FormatIdempotencyKey ---

func TestFormatIdempotencyKey(t *testing.T) {
	key := FormatIdempotencyKey("orders.cancel", "user-key-123")
	want := "idem:orders.cancel:user-key-123"
	if key != want {
		t.Errorf("key = %q, want %q", key, want)
	}
}

func TestFormatIdempotencyKey_specialChars(t *testing.T) {
	key := FormatIdempotencyKey("cmd.with.dots", "key/with/slashes")
	want := "idem:cmd.with.dots:key/with/slashes"
	if key != want {
		t.Errorf("key = %q, want %q", key, want)
	}
}
