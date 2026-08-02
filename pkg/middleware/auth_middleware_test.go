package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/cache/v9"

	"github.com/aekam27/Semantic/pkg/cache_service"
	"github.com/aekam27/Semantic/pkg/token"
)

var testSecret = []byte("a-test-secret-that-is-long-enough-32")

func newTestAuth(t *testing.T) (*Authenticator, *token.Issuer, *cache_service.TokenStore) {
	t.Helper()
	issuer, err := token.NewIssuer(testSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	store := cache_service.NewTokenStore(cache.New(&cache.Options{
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	}), time.Hour)
	return NewAuthenticator(issuer, store), issuer, store
}

// issue mints a token and records it in the store, mirroring what the token
// endpoint does.
func issue(t *testing.T, issuer *token.Issuer, store *cache_service.TokenStore, email string) string {
	t.Helper()
	signed, expiresAt, err := issuer.Issue(email)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	record := cache_service.TokenRecord{Email: email, Token: signed, ExpiresAt: expiresAt.Unix()}
	if err := store.Add(context.Background(), record); err != nil {
		t.Fatalf("store.Add: %v", err)
	}
	return signed
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestMiddlewareAllowsValidToken(t *testing.T) {
	auth, issuer, store := newTestAuth(t)
	signed := issue(t, issuer, store, "user@example.com")
	if err := auth.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/mget", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()

	auth.Middleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestMiddlewareRejectsMalformedHeaders(t *testing.T) {
	auth, _, _ := newTestAuth(t)

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"absent", "", http.StatusUnauthorized},
		{"no scheme", "sometoken", http.StatusUnauthorized},
		{"empty credential", "Bearer ", http.StatusUnauthorized},
		{"wrong scheme", "Basic abcdef", http.StatusUnauthorized},
		{"unknown token", "Bearer not-a-real-token", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/mget", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()

			auth.Middleware(okHandler()).ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Errorf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

// A token signed with a different secret must not pass, even if its raw string
// somehow appears in the cached set.
func TestMiddlewareRejectsForeignSignature(t *testing.T) {
	auth, _, store := newTestAuth(t)

	other, err := token.NewIssuer([]byte("a-different-secret-also-long-enough!"), time.Minute)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	signed, expiresAt, err := other.Issue("attacker@example.com")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	err = store.Add(context.Background(), cache_service.TokenRecord{
		Email: "attacker@example.com", Token: signed, ExpiresAt: expiresAt.Unix(),
	})
	if err != nil {
		t.Fatalf("store.Add: %v", err)
	}
	if err := auth.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/mget", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()

	auth.Middleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRefreshSkipsExpiredAndHotlistedTokens(t *testing.T) {
	auth, _, store := newTestAuth(t)
	ctx := context.Background()

	records := []cache_service.TokenRecord{
		{Email: "a@example.com", Token: "expired", ExpiresAt: time.Now().Add(-time.Hour).Unix()},
		{Email: "b@example.com", Token: "hotlisted", Hotlisted: true, ExpiresAt: time.Now().Add(time.Hour).Unix()},
		{Email: "c@example.com", Token: "live", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	for _, r := range records {
		if err := store.Add(ctx, r); err != nil {
			t.Fatalf("store.Add: %v", err)
		}
	}

	if err := auth.Refresh(ctx); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	if _, found := auth.lookup("live"); !found {
		t.Error("live token should be present")
	}
	if _, found := auth.lookup("hotlisted"); found {
		t.Error("hotlisted token should be filtered out")
	}
	if _, found := auth.lookup("expired"); found {
		t.Error("expired token should be filtered out")
	}
}

// Run with -race. Before the mutex, the refresh goroutine writing the token map
// while request goroutines read it aborted the process with "concurrent map
// read and map write".
func TestConcurrentRefreshAndLookup(t *testing.T) {
	auth, issuer, store := newTestAuth(t)
	signed := issue(t, issuer, store, "user@example.com")
	ctx := context.Background()
	if err := auth.Refresh(ctx); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				req := httptest.NewRequest(http.MethodPost, "/v1/mget", nil)
				req.Header.Set("Authorization", "Bearer "+signed)
				auth.Middleware(okHandler()).ServeHTTP(httptest.NewRecorder(), req)
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 200; j++ {
			if err := auth.Refresh(ctx); err != nil {
				t.Errorf("refresh: %v", err)
				return
			}
		}
	}()
	wg.Wait()
}

func TestStartRefreshLoadsImmediately(t *testing.T) {
	auth, issuer, store := newTestAuth(t)
	signed := issue(t, issuer, store, "user@example.com")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	auth.StartRefresh(ctx)

	// No sleep: the first load is synchronous, so the token is usable as soon
	// as StartRefresh returns.
	if _, found := auth.lookup(signed); !found {
		t.Fatal("token should be available immediately after StartRefresh")
	}
}
