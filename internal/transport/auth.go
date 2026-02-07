package transport

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/model"
)

// JWKSClient fetches and caches JSON Web Key Sets from an identity provider.
type JWKSClient struct {
	mu         sync.RWMutex
	url        string
	keys       map[string]crypto.PublicKey
	lastFetch  time.Time
	ttl        time.Duration
	minRefresh time.Duration
	httpClient *http.Client
}

// NewJWKSClient creates a new JWKS client that fetches keys from the given
// URL and caches them for the given TTL.
func NewJWKSClient(url string, ttl time.Duration) *JWKSClient {
	return &JWKSClient{
		url:        url,
		keys:       make(map[string]crypto.PublicKey),
		ttl:        ttl,
		minRefresh: 5 * time.Minute,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetKey returns the public key for the given key ID. If the key is not
// cached or the cache is expired, the JWKS endpoint is fetched.
func (c *JWKSClient) GetKey(kid string) (crypto.PublicKey, error) {
	c.mu.RLock()
	key, ok := c.keys[kid]
	expired := time.Since(c.lastFetch) > c.ttl
	c.mu.RUnlock()

	if ok && !expired {
		return key, nil
	}

	if err := c.refresh(); err != nil {
		// Degraded mode: use cached key if available.
		c.mu.RLock()
		key, ok = c.keys[kid]
		c.mu.RUnlock()
		if ok {
			slog.Warn("jwks: refresh failed, using cached key", "error", err)
			return key, nil
		}
		return nil, fmt.Errorf("jwks: fetch failed: %w", err)
	}

	c.mu.RLock()
	key, ok = c.keys[kid]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("jwks: unknown signing key %q", kid)
	}
	return key, nil
}

func (c *JWKSClient) refresh() error {
	c.mu.RLock()
	tooSoon := time.Since(c.lastFetch) < c.minRefresh && len(c.keys) > 0
	c.mu.RUnlock()
	if tooSoon {
		return nil
	}

	resp, err := c.httpClient.Get(c.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	var jwks struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("jwks: parse error: %w", err)
	}

	keys := make(map[string]crypto.PublicKey, len(jwks.Keys))
	for _, raw := range jwks.Keys {
		var jwk map[string]any
		if err := json.Unmarshal(raw, &jwk); err != nil {
			continue
		}
		kid, _ := jwk["kid"].(string)
		if kid == "" {
			continue
		}
		kty, _ := jwk["kty"].(string)
		var key crypto.PublicKey
		switch kty {
		case "RSA":
			key, err = parseRSAKey(jwk)
		case "EC":
			key, err = parseECKey(jwk)
		default:
			continue
		}
		if err != nil {
			slog.Warn("jwks: failed to parse key", "kid", kid, "error", err)
			continue
		}
		keys[kid] = key
	}

	c.mu.Lock()
	c.keys = keys
	c.lastFetch = time.Now()
	c.mu.Unlock()

	return nil
}

func parseRSAKey(jwk map[string]any) (*rsa.PublicKey, error) {
	nStr, _ := jwk["n"].(string)
	eStr, _ := jwk["e"].(string)
	if nStr == "" || eStr == "" {
		return nil, fmt.Errorf("missing n or e")
	}
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}

func parseECKey(jwk map[string]any) (*ecdsa.PublicKey, error) {
	crv, _ := jwk["crv"].(string)
	xStr, _ := jwk["x"].(string)
	yStr, _ := jwk["y"].(string)
	if crv == "" || xStr == "" || yStr == "" {
		return nil, fmt.Errorf("missing crv, x, or y")
	}
	var curve elliptic.Curve
	switch crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported curve %q", crv)
	}
	xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
	if err != nil {
		return nil, fmt.Errorf("decode x: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
	if err != nil {
		return nil, fmt.Errorf("decode y: %w", err)
	}
	return &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

// JWTAuthenticator returns middleware that verifies JWT tokens from the
// Authorization header and stores verified claims in the request context.
func JWTAuthenticator(cfg config.IdentityConfig, jwks *JWKSClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				WriteError(w, model.NewUnauthorizedError("Missing authorization header"))
				return
			}
			if !strings.HasPrefix(auth, "Bearer ") {
				WriteError(w, model.NewUnauthorizedError("Invalid authorization header format"))
				return
			}
			tokenStr := auth[7:]

			token, err := jwt.Parse(tokenStr,
				func(token *jwt.Token) (any, error) {
					kid, _ := token.Header["kid"].(string)
					if kid == "" {
						return nil, fmt.Errorf("missing kid in token header")
					}
					return jwks.GetKey(kid)
				},
				jwt.WithValidMethods(cfg.Algorithms),
				jwt.WithIssuer(cfg.Issuer),
				jwt.WithAudience(cfg.Audience),
				jwt.WithLeeway(30*time.Second),
				jwt.WithExpirationRequired(),
			)
			if err != nil {
				WriteError(w, model.NewUnauthorizedError(classifyJWTError(err)))
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok || !token.Valid {
				WriteError(w, model.NewUnauthorizedError("Invalid token"))
				return
			}

			ctx := WithClaims(r.Context(), map[string]any(claims))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func classifyJWTError(err error) string {
	s := err.Error()
	switch {
	case strings.Contains(s, "expired"):
		return "Token expired"
	case strings.Contains(s, "issuer"):
		return "Invalid token issuer"
	case strings.Contains(s, "audience"):
		return "Invalid token audience"
	case strings.Contains(s, "signing method"):
		return "Disallowed signing algorithm"
	case strings.Contains(s, "kid"):
		return "Unknown signing key"
	case strings.Contains(s, "signature"):
		return "Invalid token signature"
	default:
		return "Invalid token"
	}
}
