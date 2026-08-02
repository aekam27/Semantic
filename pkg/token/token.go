// Package token issues and verifies the API's JWTs.
package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// signingMethod is pinned so a token presented with a different algorithm is
// rejected outright. Parsing without this check is what makes algorithm
// confusion attacks possible.
var signingMethod = jwt.SigningMethodHS256

// ErrInvalidToken is returned for any token that fails verification.
var ErrInvalidToken = errors.New("invalid token")

// Claims is the payload carried by an issued token.
type Claims struct {
	Email      string `json:"email"`
	Authorized bool   `json:"authorized"`
	jwt.RegisteredClaims
}

// Issuer signs and verifies tokens with a single secret.
type Issuer struct {
	secret []byte
	ttl    time.Duration
}

// NewIssuer builds an Issuer. The secret must be non-empty.
func NewIssuer(secret []byte, ttl time.Duration) (*Issuer, error) {
	if len(secret) == 0 {
		return nil, errors.New("token: signing secret must not be empty")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &Issuer{secret: secret, ttl: ttl}, nil
}

// Issue returns a signed token for email along with its expiry.
func (i *Issuer) Issue(email string) (string, time.Time, error) {
	expiresAt := time.Now().Add(i.ttl)
	claims := Claims{
		Email:      email,
		Authorized: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	signed, err := jwt.NewWithClaims(signingMethod, claims).SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

// Parse verifies a token's signature, algorithm and expiry, and returns its
// claims.
func (i *Issuer) Parse(raw string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(*jwt.Token) (interface{}, error) { return i.secret, nil },
		jwt.WithValidMethods([]string{signingMethod.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidToken, err)
	}
	return claims, nil
}
