// Package auth_service issues API tokens.
package auth_service

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/aekam27/Semantic/pkg/cache_service"
	"github.com/aekam27/Semantic/pkg/token"
)

// maxBodyBytes caps the credential payload we are willing to read.
const maxBodyBytes = 1 << 16 // 64 KiB

// Handler serves the token endpoint.
type Handler struct {
	issuer *token.Issuer
	store  *cache_service.TokenStore

	// onIssue, when set, is called after a token is persisted so the
	// authenticator can pick it up straight away. Without it a caller has to
	// wait for the next refresh tick before the token it was just handed
	// actually works.
	onIssue func(context.Context) error
}

// NewHandler builds the token endpoint handler. onIssue may be nil.
func NewHandler(
	issuer *token.Issuer,
	store *cache_service.TokenStore,
	onIssue func(context.Context) error,
) *Handler {
	return &Handler{issuer: issuer, store: store, onIssue: onIssue}
}

// Creds is the request payload for a token.
type Creds struct {
	Email string `json:"email,omitempty"`
	Name  string `json:"name,omitempty"`
}

// GenerateToken issues a token for the supplied email.
func (h *Handler) GenerateToken(w http.ResponseWriter, r *http.Request) {
	creds, err := decodeCreds(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(creds.Email) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email is required"})
		return
	}

	signed, expiresAt, err := h.issuer.Issue(creds.Email)
	if err != nil {
		log.Printf("auth: issue token: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	record := cache_service.TokenRecord{
		Email:     creds.Email,
		Token:     signed,
		ExpiresAt: expiresAt.Unix(),
	}
	if err := h.store.Add(r.Context(), record); err != nil {
		// The middleware authorises against the stored set, so a token we
		// could not persist would be rejected on every subsequent request.
		// Failing here is clearer than handing back a token that never works.
		log.Printf("auth: persist token: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if h.onIssue != nil {
		// Best effort: the periodic refresh will catch up regardless, so a
		// failure here costs latency rather than correctness.
		if err := h.onIssue(r.Context()); err != nil {
			log.Printf("auth: refresh after issue: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": signed})
}

func decodeCreds(r *http.Request) (Creds, error) {
	var creds Creds
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&creds); err != nil {
		return Creds{}, err
	}
	return creds, nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("auth: write response: %v", err)
	}
}
