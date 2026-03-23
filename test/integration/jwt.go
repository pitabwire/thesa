package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/pitabwire/frame/security"
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
// It implements config.ConfigurationJWTVerification so it can be used directly
// with Frame's openid.NewJwtTokenAuthenticator.
type tokenIssuer struct {
	privateKey *rsa.PrivateKey
	jwksServer *httptest.Server
	issuer     string
	audience   string
	jwkData    string
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
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}

	jwkSetData, _ := json.Marshal(map[string]any{
		"keys": []map[string]any{jwk},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwkSetData)
	}))
	t.Cleanup(srv.Close)

	return &tokenIssuer{
		privateKey: key,
		jwksServer: srv,
		issuer:     "https://auth.test.thesa.dev",
		audience:   "service_thesa",
		jwkData:    string(jwkSetData),
	}
}

// --- config.ConfigurationJWTVerification implementation ---

func (ti *tokenIssuer) GetOauth2WellKnownJwk() string {
	return ti.jwksServer.URL
}

func (ti *tokenIssuer) GetOauth2WellKnownJwkData() string {
	return ti.jwkData
}

func (ti *tokenIssuer) GetVerificationAudience() []string {
	return []string{ti.audience}
}

func (ti *tokenIssuer) GetVerificationIssuer() string {
	return ti.issuer
}

// GenerateToken creates a valid, signed JWT token with the given claims.
// Tokens use Frame's AuthenticationClaims structure (typed fields + ext map).
func (ti *tokenIssuer) GenerateToken(claims TestClaims) string {
	now := time.Now()

	authClaims := &security.AuthenticationClaims{
		TenantID: claims.TenantID,
		Ext:      map[string]any{"email": claims.Email},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ti.issuer,
			Audience:  jwt.ClaimStrings{ti.audience},
			Subject:   claims.SubjectID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
		},
	}

	if len(claims.Roles) > 0 {
		authClaims.Roles = claims.Roles
	}

	for k, v := range claims.Extra {
		authClaims.Ext[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, authClaims)
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

	authClaims := &security.AuthenticationClaims{
		TenantID: claims.TenantID,
		Ext:      map[string]any{"email": claims.Email},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ti.issuer,
			Audience:  jwt.ClaimStrings{ti.audience},
			Subject:   claims.SubjectID,
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, authClaims)
	token.Header["kid"] = testKeyID

	signed, err := token.SignedString(ti.privateKey)
	if err != nil {
		panic("sign JWT: " + err.Error())
	}
	return signed
}

// GenerateMapClaimsToken creates a JWT using MapClaims for tests that need
// non-standard token structures (e.g. security controls tests).
func (ti *tokenIssuer) GenerateMapClaimsToken(claims jwt.MapClaims) string {
	// Ensure defaults.
	if _, ok := claims["iss"]; !ok {
		claims["iss"] = ti.issuer
	}
	if _, ok := claims["aud"]; !ok {
		claims["aud"] = ti.audience
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
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
