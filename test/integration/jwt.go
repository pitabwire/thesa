package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"maps"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testKeyID = "test-key-1"

// TestClaims holds the configurable claims for generating test JWT tokens.
type TestClaims struct {
	SubjectID string
	TenantID  string
	Email     string
	Roles     []string
	Extra     map[string]any
}

// tokenIssuer holds an RSA key pair for signing JWTs and serves a JWKS endpoint.
type tokenIssuer struct {
	privateKey *rsa.PrivateKey
	jwksServer *httptest.Server
	issuer     string
	audience   string
}

// newTokenIssuer creates a token issuer with a fresh RSA key pair and
// a test JWKS server.
func newTokenIssuer(t *testing.T) *tokenIssuer {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	jwk := map[string]any{
		"kid": testKeyID,
		"kty": "RSA",
		"alg": "RS256",
		"use": "sig",
		"n":   base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{jwk},
		})
	}))
	t.Cleanup(srv.Close)

	return &tokenIssuer{
		privateKey: key,
		jwksServer: srv,
		issuer:     "https://auth.test.thesa.dev",
		audience:   "thesa-bff-test",
	}
}

// GenerateToken creates a valid, signed JWT token with the given claims.
func (ti *tokenIssuer) GenerateToken(claims TestClaims) string {
	now := time.Now()

	mapClaims := jwt.MapClaims{
		"iss":       ti.issuer,
		"aud":       ti.audience,
		"iat":       jwt.NewNumericDate(now),
		"exp":       jwt.NewNumericDate(now.Add(1 * time.Hour)),
		"sub":       claims.SubjectID,
		"tenant_id": claims.TenantID,
		"email":     claims.Email,
	}

	if len(claims.Roles) > 0 {
		// Store as []any to match JWT decode behavior.
		roles := make([]any, len(claims.Roles))
		for i, r := range claims.Roles {
			roles[i] = r
		}
		mapClaims["roles"] = roles
	}

	maps.Copy(mapClaims, claims.Extra)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, mapClaims)
	token.Header["kid"] = testKeyID

	signed, err := token.SignedString(ti.privateKey)
	if err != nil {
		panic("sign JWT: " + err.Error())
	}
	return signed
}

// GenerateExpiredToken creates a JWT token that expired in the past.
func (ti *tokenIssuer) GenerateExpiredToken(claims TestClaims) string {
	now := time.Now()

	mapClaims := jwt.MapClaims{
		"iss":       ti.issuer,
		"aud":       ti.audience,
		"iat":       jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		"exp":       jwt.NewNumericDate(now.Add(-1 * time.Hour)),
		"sub":       claims.SubjectID,
		"tenant_id": claims.TenantID,
		"email":     claims.Email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, mapClaims)
	token.Header["kid"] = testKeyID

	signed, err := token.SignedString(ti.privateKey)
	if err != nil {
		panic("sign JWT: " + err.Error())
	}
	return signed
}

// JWKSURL returns the URL of the JWKS endpoint served by this issuer.
func (ti *tokenIssuer) JWKSURL() string {
	return ti.jwksServer.URL
}

// Issuer returns the expected token issuer claim.
func (ti *tokenIssuer) Issuer() string {
	return ti.issuer
}

// Audience returns the expected token audience claim.
func (ti *tokenIssuer) Audience() string {
	return ti.audience
}
