// Package middleware holds the HTTP middleware chain.
package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aekam27/Semantic/pkg/cache_service"
	"github.com/aekam27/Semantic/pkg/token"
)

// refreshInterval is how often the live token set is pulled from Redis.
const refreshInterval = 10 * time.Second

// Authenticator validates bearer tokens against a locally cached copy of the
// issued token set.
type Authenticator struct {
	issuer *token.Issuer
	store  *cache_service.TokenStore

	// mu guards tokens. The refresh goroutine writes it while request
	// goroutines read it; without this the Go runtime aborts the process on
	// "concurrent map read and map write".
	mu     sync.RWMutex
	tokens map[string]cache_service.TokenRecord
}

// NewAuthenticator builds an Authenticator with an empty token set.
func NewAuthenticator(issuer *token.Issuer, store *cache_service.TokenStore) *Authenticator {
	return &Authenticator{
		issuer: issuer,
		store:  store,
		tokens: make(map[string]cache_service.TokenRecord),
	}
}

// StartRefresh loads the token set immediately and then keeps it up to date
// until ctx is cancelled.
//
// The initial synchronous load matters: previously the first refresh only
// happened one tick after boot, so every authenticated request in the first ten
// seconds of a process's life was rejected.
func (a *Authenticator) StartRefresh(ctx context.Context) {
	if err := a.Refresh(ctx); err != nil {
		log.Printf("auth: initial token load failed: %v", err)
	}

	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := a.Refresh(ctx); err != nil {
					// A failed refresh is transient: Redis may be briefly
					// unavailable. Keep serving the tokens already loaded and
					// try again on the next tick rather than stopping forever.
					log.Printf("auth: token refresh failed: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Refresh replaces the local token set with what is currently in the store.
//
// Callers can invoke this directly to pull a change forward rather than
// waiting for the next tick — the token endpoint does exactly that, so a
// token works on the very next request instead of up to refreshInterval later.
func (a *Authenticator) Refresh(ctx context.Context) error {
	records, err := a.store.List(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	fresh := make(map[string]cache_service.TokenRecord, len(records))
	for _, record := range records {
		if record.Expired(now) || record.Hotlisted {
			continue
		}
		fresh[record.Token] = record
	}

	a.mu.Lock()
	a.tokens = fresh
	a.mu.Unlock()
	return nil
}

// lookup returns the stored record for a raw token.
func (a *Authenticator) lookup(raw string) (cache_service.TokenRecord, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	record, found := a.tokens[raw]
	return record, found
}

// Middleware rejects any request without a valid, non-revoked bearer token.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := bearerToken(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		record, found := a.lookup(raw)
		if !found {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		claims, err := a.issuer.Parse(raw)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// The signature already covers these, but the cached record is the
		// revocation source of truth: if it disagrees with the token, refuse.
		if record.Email != claims.Email {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if claims.ExpiresAt == nil || record.ExpiresAt != claims.ExpiresAt.Unix() {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if record.Expired(time.Now()) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// bearerToken extracts the credential from an Authorization header.
func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	scheme, credential, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || credential == "" {
		return "", false
	}
	return credential, true
}
