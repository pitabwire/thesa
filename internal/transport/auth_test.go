package transport

import (
	"crypto/ecdsa"
	"crypto/elliptic"
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

	"github.com/pitabwire/thesa/internal/config"
)

// --- test helpers ---

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return key
}

func generateECKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return key
}

func rsaKeyToJWK(kid string, pub *rsa.PublicKey) map[string]any {
	return map[string]any{
		"kid": kid,
		"kty": "RSA",
		"alg": "RS256",
		"use": "sig",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}

func ecKeyToJWK(kid string, pub *ecdsa.PublicKey) map[string]any {
	return map[string]any{
		"kid": kid,
		"kty": "EC",
		"crv": "P-256",
		"use": "sig",
		"x":   base64.RawURLEncoding.EncodeToString(pub.X.Bytes()),
		"y":   base64.RawURLEncoding.EncodeToString(pub.Y.Bytes()),
	}
}

func startJWKSServer(t *testing.T, keys ...map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"keys": keys})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func signJWT(t *testing.T, key any, method jwt.SigningMethod, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = kid
	s, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	return s
}

func testIdentityCfg() config.IdentityConfig {
	return config.IdentityConfig{
		Issuer:     "https://auth.example.com",
		Audience:   "thesa-bff",
		Algorithms: []string{"RS256", "ES256"},
		ClaimPaths: map[string]string{
			"subject_id": "sub",
			"tenant_id":  "tenant_id",
			"email":      "email",
			"roles":      "roles",
		},
	}
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub":       "user-1",
		"tenant_id": "tenant-1",
		"email":     "user@example.com",
		"roles":     []string{"admin"},
		"iss":       "https://auth.example.com",
		"aud":       "thesa-bff",
		"exp":       jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		"iat":       jwt.NewNumericDate(time.Now()),
	}
}

// --- JWKSClient tests ---

func TestJWKSClient_GetKey_RSA(t *testing.T) {
	rsaKey := generateRSAKey(t)
	jwks := startJWKSServer(t, rsaKeyToJWK("rsa-key-1", &rsaKey.PublicKey))

	client := NewJWKSClient(jwks.URL, 1*time.Hour)
	key, err := client.GetKey("rsa-key-1")
	if err != nil {
		t.Fatalf("GetKey: %v", err)
	}
	pubKey, ok := key.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("key type = %T, want *rsa.PublicKey", key)
	}
	if pubKey.N.Cmp(rsaKey.PublicKey.N) != 0 {
		t.Error("RSA modulus mismatch")
	}
}

func TestJWKSClient_GetKey_EC(t *testing.T) {
	ecKey := generateECKey(t)
	jwks := startJWKSServer(t, ecKeyToJWK("ec-key-1", &ecKey.PublicKey))

	client := NewJWKSClient(jwks.URL, 1*time.Hour)
	key, err := client.GetKey("ec-key-1")
	if err != nil {
		t.Fatalf("GetKey: %v", err)
	}
	pubKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("key type = %T, want *ecdsa.PublicKey", key)
	}
	if pubKey.X.Cmp(ecKey.PublicKey.X) != 0 {
		t.Error("EC X coordinate mismatch")
	}
}

func TestJWKSClient_GetKey_unknown(t *testing.T) {
	jwks := startJWKSServer(t) // empty JWKS
	client := NewJWKSClient(jwks.URL, 1*time.Hour)
	_, err := client.GetKey("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestJWKSClient_caching(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		rsaKey := generateRSAKey(t)
		keys := []map[string]any{rsaKeyToJWK("cached-key", &rsaKey.PublicKey)}
		json.NewEncoder(w).Encode(map[string]any{"keys": keys})
	}))
	defer srv.Close()

	client := NewJWKSClient(srv.URL, 1*time.Hour)
	client.minRefresh = 0 // allow rapid refresh for test

	client.GetKey("cached-key")
	client.GetKey("cached-key")

	if callCount != 1 {
		t.Errorf("JWKS fetched %d times, want 1 (should be cached)", callCount)
	}
}

func TestJWKSClient_multipleKeys(t *testing.T) {
	rsaKey1 := generateRSAKey(t)
	rsaKey2 := generateRSAKey(t)
	jwks := startJWKSServer(t,
		rsaKeyToJWK("key-1", &rsaKey1.PublicKey),
		rsaKeyToJWK("key-2", &rsaKey2.PublicKey),
	)

	client := NewJWKSClient(jwks.URL, 1*time.Hour)

	k1, err := client.GetKey("key-1")
	if err != nil {
		t.Fatalf("GetKey(key-1): %v", err)
	}
	k2, err := client.GetKey("key-2")
	if err != nil {
		t.Fatalf("GetKey(key-2): %v", err)
	}
	if k1.(*rsa.PublicKey).N.Cmp(k2.(*rsa.PublicKey).N) == 0 {
		t.Error("keys should be different")
	}
}

// --- JWTAuthenticator tests ---

func TestJWTAuthenticator_validToken(t *testing.T) {
	rsaKey := generateRSAKey(t)
	jwksSrv := startJWKSServer(t, rsaKeyToJWK("test-key", &rsaKey.PublicKey))

	cfg := testIdentityCfg()
	jwksClient := NewJWKSClient(jwksSrv.URL, 1*time.Hour)

	handler := JWTAuthenticator(cfg, jwksClient)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFrom(r.Context())
		if claims == nil {
			t.Error("claims should be in context")
		}
		sub, _ := claims["sub"].(string)
		if sub != "user-1" {
			t.Errorf("sub = %q, want user-1", sub)
		}
		w.WriteHeader(200)
	}))

	tokenStr := signJWT(t, rsaKey, jwt.SigningMethodRS256, "test-key", validClaims())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestJWTAuthenticator_validToken_EC(t *testing.T) {
	ecKey := generateECKey(t)
	jwksSrv := startJWKSServer(t, ecKeyToJWK("ec-test", &ecKey.PublicKey))

	cfg := testIdentityCfg()
	jwksClient := NewJWKSClient(jwksSrv.URL, 1*time.Hour)

	handler := JWTAuthenticator(cfg, jwksClient)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	tokenStr := signJWT(t, ecKey, jwt.SigningMethodES256, "ec-test", validClaims())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200 for ES256 token", w.Code)
	}
}

func TestJWTAuthenticator_missingAuthHeader(t *testing.T) {
	cfg := testIdentityCfg()
	jwksClient := NewJWKSClient("http://unused", 1*time.Hour)
	handler := JWTAuthenticator(cfg, jwksClient)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestJWTAuthenticator_invalidFormat(t *testing.T) {
	cfg := testIdentityCfg()
	jwksClient := NewJWKSClient("http://unused", 1*time.Hour)
	handler := JWTAuthenticator(cfg, jwksClient)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestJWTAuthenticator_expiredToken(t *testing.T) {
	rsaKey := generateRSAKey(t)
	jwksSrv := startJWKSServer(t, rsaKeyToJWK("test-key", &rsaKey.PublicKey))

	cfg := testIdentityCfg()
	jwksClient := NewJWKSClient(jwksSrv.URL, 1*time.Hour)
	handler := JWTAuthenticator(cfg, jwksClient)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for expired token")
	}))

	claims := validClaims()
	claims["exp"] = jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)) // expired

	tokenStr := signJWT(t, rsaKey, jwt.SigningMethodRS256, "test-key", claims)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401 for expired token", w.Code)
	}
}

func TestJWTAuthenticator_wrongIssuer(t *testing.T) {
	rsaKey := generateRSAKey(t)
	jwksSrv := startJWKSServer(t, rsaKeyToJWK("test-key", &rsaKey.PublicKey))

	cfg := testIdentityCfg()
	jwksClient := NewJWKSClient(jwksSrv.URL, 1*time.Hour)
	handler := JWTAuthenticator(cfg, jwksClient)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for wrong issuer")
	}))

	claims := validClaims()
	claims["iss"] = "https://evil.example.com"

	tokenStr := signJWT(t, rsaKey, jwt.SigningMethodRS256, "test-key", claims)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401 for wrong issuer", w.Code)
	}
}

func TestJWTAuthenticator_wrongAudience(t *testing.T) {
	rsaKey := generateRSAKey(t)
	jwksSrv := startJWKSServer(t, rsaKeyToJWK("test-key", &rsaKey.PublicKey))

	cfg := testIdentityCfg()
	jwksClient := NewJWKSClient(jwksSrv.URL, 1*time.Hour)
	handler := JWTAuthenticator(cfg, jwksClient)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for wrong audience")
	}))

	claims := validClaims()
	claims["aud"] = "wrong-audience"

	tokenStr := signJWT(t, rsaKey, jwt.SigningMethodRS256, "test-key", claims)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401 for wrong audience", w.Code)
	}
}

func TestJWTAuthenticator_disallowedAlgorithm(t *testing.T) {
	rsaKey := generateRSAKey(t)
	jwksSrv := startJWKSServer(t, rsaKeyToJWK("test-key", &rsaKey.PublicKey))

	cfg := testIdentityCfg()
	cfg.Algorithms = []string{"ES256"} // only allow ES256, not RS256
	jwksClient := NewJWKSClient(jwksSrv.URL, 1*time.Hour)
	handler := JWTAuthenticator(cfg, jwksClient)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for disallowed algorithm")
	}))

	tokenStr := signJWT(t, rsaKey, jwt.SigningMethodRS256, "test-key", validClaims())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401 for disallowed algorithm", w.Code)
	}
}

func TestJWTAuthenticator_unknownKid(t *testing.T) {
	rsaKey := generateRSAKey(t)
	jwksSrv := startJWKSServer(t, rsaKeyToJWK("known-key", &rsaKey.PublicKey))

	cfg := testIdentityCfg()
	jwksClient := NewJWKSClient(jwksSrv.URL, 1*time.Hour)
	jwksClient.minRefresh = 0 // allow refresh in test
	handler := JWTAuthenticator(cfg, jwksClient)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for unknown kid")
	}))

	tokenStr := signJWT(t, rsaKey, jwt.SigningMethodRS256, "unknown-key", validClaims())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401 for unknown kid", w.Code)
	}
}

func TestJWTAuthenticator_missingExpClaim(t *testing.T) {
	rsaKey := generateRSAKey(t)
	jwksSrv := startJWKSServer(t, rsaKeyToJWK("test-key", &rsaKey.PublicKey))

	cfg := testIdentityCfg()
	jwksClient := NewJWKSClient(jwksSrv.URL, 1*time.Hour)
	handler := JWTAuthenticator(cfg, jwksClient)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for missing exp")
	}))

	claims := validClaims()
	delete(claims, "exp") // remove expiration

	tokenStr := signJWT(t, rsaKey, jwt.SigningMethodRS256, "test-key", claims)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401 for missing exp claim", w.Code)
	}
}

func TestJWTAuthenticator_clockSkewTolerance(t *testing.T) {
	rsaKey := generateRSAKey(t)
	jwksSrv := startJWKSServer(t, rsaKeyToJWK("test-key", &rsaKey.PublicKey))

	cfg := testIdentityCfg()
	jwksClient := NewJWKSClient(jwksSrv.URL, 1*time.Hour)
	handler := JWTAuthenticator(cfg, jwksClient)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Token expired 15 seconds ago — within 30s leeway.
	claims := validClaims()
	claims["exp"] = jwt.NewNumericDate(time.Now().Add(-15 * time.Second))

	tokenStr := signJWT(t, rsaKey, jwt.SigningMethodRS256, "test-key", claims)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200 (token within clock skew tolerance)", w.Code)
	}
}

// --- extractClaim tests ---

func TestExtractClaim_dotNotation(t *testing.T) {
	claims := map[string]any{
		"realm_access": map[string]any{
			"roles": []any{"admin", "viewer"},
		},
		"sub": "user-1",
	}

	// Simple path
	if v := extractClaimString(claims, "sub"); v != "user-1" {
		t.Errorf("sub = %q, want user-1", v)
	}

	// Nested path
	roles := extractClaimStringSlice(claims, "realm_access.roles")
	if len(roles) != 2 || roles[0] != "admin" {
		t.Errorf("realm_access.roles = %v, want [admin viewer]", roles)
	}

	// Missing path
	if v := extractClaimString(claims, "nonexistent.path"); v != "" {
		t.Errorf("nonexistent.path = %q, want empty", v)
	}

	// Nil claims
	if v := extractClaimString(nil, "sub"); v != "" {
		t.Errorf("nil claims = %q, want empty", v)
	}
}
