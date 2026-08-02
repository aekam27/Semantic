package token

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("a-test-secret-that-is-long-enough-32")

func newTestIssuer(t *testing.T, ttl time.Duration) *Issuer {
	t.Helper()
	issuer, err := NewIssuer(testSecret, ttl)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	return issuer
}

func TestNewIssuerRejectsEmptySecret(t *testing.T) {
	if _, err := NewIssuer(nil, time.Minute); err == nil {
		t.Fatal("expected an error for an empty secret")
	}
}

func TestIssueAndParseRoundTrip(t *testing.T) {
	issuer := newTestIssuer(t, 15*time.Minute)

	signed, expiresAt, err := issuer.Issue("user@example.com")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	claims, err := issuer.Parse(signed)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("email = %q, want user@example.com", claims.Email)
	}
	if !claims.Authorized {
		t.Error("authorized = false, want true")
	}
	if claims.ExpiresAt == nil {
		t.Fatal("expiry claim is missing")
	}
	if got := claims.ExpiresAt.Unix(); got != expiresAt.Unix() {
		t.Errorf("expiry claim = %d, want %d", got, expiresAt.Unix())
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	issuer := newTestIssuer(t, time.Minute)
	signed, _, err := issuer.Issue("user@example.com")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	other, err := NewIssuer([]byte("a-different-secret-also-long-enough!"), time.Minute)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	if _, err := other.Parse(signed); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

// A token presented with alg "none" must be rejected. Parsing without pinning
// the signing method is what allows this class of attack.
func TestParseRejectsNoneAlgorithm(t *testing.T) {
	issuer := newTestIssuer(t, time.Minute)

	claims := Claims{
		Email:      "attacker@example.com",
		Authorized: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("build none-signed token: %v", err)
	}

	if _, err := issuer.Parse(unsigned); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	issuer := newTestIssuer(t, time.Minute)

	// A negative TTL is normalised to the 15 minute default, so the expired
	// token has to be built by hand.
	claims := Claims{
		Email: "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(testSecret)
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	if _, err := issuer.Parse(expired); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestParseRejectsTokenWithoutExpiry(t *testing.T) {
	issuer := newTestIssuer(t, time.Minute)

	claims := Claims{Email: "user@example.com"}
	noExpiry, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(testSecret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := issuer.Parse(noExpiry); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	issuer := newTestIssuer(t, time.Minute)
	for _, raw := range []string{"", "not-a-token", "a.b.c"} {
		if _, err := issuer.Parse(raw); err == nil {
			t.Errorf("Parse(%q) succeeded, want an error", raw)
		}
	}
}
