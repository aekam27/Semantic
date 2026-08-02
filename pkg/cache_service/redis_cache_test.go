package cache_service

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/cache/v9"
)

// newLocalStore builds a TokenStore backed only by the in-process cache, so
// these tests need no Redis server.
func newLocalStore(t *testing.T) *TokenStore {
	t.Helper()
	return NewTokenStore(cache.New(&cache.Options{
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	}), time.Hour)
}

func TestListReturnsNothingWhenNoTokensIssued(t *testing.T) {
	store := newLocalStore(t)

	records, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List on an empty store returned an error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("got %d records, want 0", len(records))
	}
}

func TestAddThenList(t *testing.T) {
	store := newLocalStore(t)
	ctx := context.Background()

	want := TokenRecord{
		Email:     "user@example.com",
		Token:     "tok-1",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	if err := store.Add(ctx, want); err != nil {
		t.Fatalf("Add: %v", err)
	}

	records, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].Email != want.Email || records[0].Token != want.Token {
		t.Errorf("got %+v, want %+v", records[0], want)
	}
}

func TestAddDropsExpiredRecords(t *testing.T) {
	store := newLocalStore(t)
	ctx := context.Background()

	stale := TokenRecord{Token: "stale", ExpiresAt: time.Now().Add(-time.Hour).Unix()}
	if err := store.Add(ctx, stale); err != nil {
		t.Fatalf("Add stale: %v", err)
	}
	fresh := TokenRecord{Token: "fresh", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	if err := store.Add(ctx, fresh); err != nil {
		t.Fatalf("Add fresh: %v", err)
	}

	records, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1 (the expired one should be pruned)", len(records))
	}
	if records[0].Token != "fresh" {
		t.Errorf("kept %q, want fresh", records[0].Token)
	}
}

func TestExpired(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		exp  int64
		want bool
	}{
		{"past", now.Add(-time.Second).Unix(), true},
		{"now", now.Unix(), true},
		{"future", now.Add(time.Hour).Unix(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := TokenRecord{ExpiresAt: tc.exp}.Expired(now)
			if got != tc.want {
				t.Errorf("Expired() = %v, want %v", got, tc.want)
			}
		})
	}
}
