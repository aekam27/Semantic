package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/cache/v9"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/aekam27/Semantic/pkg/auth_service"
	"github.com/aekam27/Semantic/pkg/cache_service"
	"github.com/aekam27/Semantic/pkg/dao"
	"github.com/aekam27/Semantic/pkg/middleware"
	"github.com/aekam27/Semantic/pkg/semantic_search_service"
	"github.com/aekam27/Semantic/pkg/token"
)

var testSecret = []byte("a-test-secret-that-is-long-enough-32")

// stubDAO returns a fixed result without touching MongoDB.
type stubDAO struct {
	results []bson.M
	err     error
}

func (s stubDAO) VectorFind(context.Context, dao.VectorSearch) ([]bson.M, error) {
	return s.results, s.err
}

func newTestServer(t *testing.T, mongoDAO dao.MongoDAO, embeddingURL string) (http.Handler, *middleware.Authenticator) {
	t.Helper()

	issuer, err := token.NewIssuer(testSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	store := cache_service.NewTokenStore(cache.New(&cache.Options{
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	}), time.Hour)

	auth := middleware.NewAuthenticator(issuer, store)
	handler := NewRouter(
		auth,
		auth_service.NewHandler(issuer, store, auth.Refresh),
		semantic_search_service.NewHandler(mongoDAO, embeddingURL, http.DefaultClient),
	)
	return handler, auth
}

func TestHealthEndpoint(t *testing.T) {
	handler, _ := newTestServer(t, stubDAO{}, "")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

func TestSearchRequiresAuthentication(t *testing.T) {
	handler, _ := newTestServer(t, stubDAO{}, "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/mget", strings.NewReader(`{"text":"hi"}`))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestGenerateTokenIsPublic(t *testing.T) {
	handler, _ := newTestServer(t, stubDAO{}, "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/generateToken",
		strings.NewReader(`{"email":"user@example.com"}`))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["token"] == "" {
		t.Error("expected a token in the response")
	}
}

func TestGenerateTokenRejectsMissingEmail(t *testing.T) {
	handler, _ := newTestServer(t, stubDAO{}, "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/generateToken", strings.NewReader(`{}`))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestGenerateTokenRejectsMalformedBody(t *testing.T) {
	handler, _ := newTestServer(t, stubDAO{}, "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/generateToken", strings.NewReader(`not json`))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// End to end over the real router: mint a token, then use it to search against
// a stub embedding service and a stub DAO.
func TestAuthenticatedSearchRoundTrip(t *testing.T) {
	embedding := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string][]float64{"vector_embedding": {0.1, 0.2, 0.3}})
	}))
	defer embedding.Close()

	want := []bson.M{{"title": "a result"}}
	handler, _ := newTestServer(t, stubDAO{results: want}, embedding.URL)

	// Mint a token through the public endpoint.
	tokenRec := httptest.NewRecorder()
	handler.ServeHTTP(tokenRec, httptest.NewRequest(http.MethodPost, "/v1/generateToken",
		strings.NewReader(`{"email":"user@example.com"}`)))
	if tokenRec.Code != http.StatusOK {
		t.Fatalf("token status = %d: %s", tokenRec.Code, tokenRec.Body.String())
	}
	var tokenBody map[string]string
	if err := json.Unmarshal(tokenRec.Body.Bytes(), &tokenBody); err != nil {
		t.Fatalf("decode token body: %v", err)
	}

	// No manual refresh here on purpose: issuing a token triggers one, so the
	// token has to work on the very next request.
	searchRec := httptest.NewRecorder()
	searchReq := httptest.NewRequest(http.MethodPost, "/v1/mget",
		strings.NewReader(`{"searchIndex":"idx","path":"embedding","text":"hello","limit":5}`))
	searchReq.Header.Set("Authorization", "Bearer "+tokenBody["token"])
	handler.ServeHTTP(searchRec, searchReq)

	if searchRec.Code != http.StatusOK {
		t.Fatalf("search status = %d, want 200: %s", searchRec.Code, searchRec.Body.String())
	}
	if got := searchRec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Errorf("Content-Encoding = %q, want gzip", got)
	}
}
