package transport

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/util"

	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/model"
)

// Context keys for middleware-injected values.
type correlationIDKey struct{}
type capabilitiesKey struct{}

// CorrelationIDFrom extracts the correlation ID from the request context.
func CorrelationIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(correlationIDKey{}).(string)
	return id
}

// CapabilitiesFrom extracts the CapabilitySet from the context.
func CapabilitiesFrom(ctx context.Context) model.CapabilitySet {
	caps, _ := ctx.Value(capabilitiesKey{}).(model.CapabilitySet)
	return caps
}

// Recovery catches panics in downstream handlers, logs them, and returns
// a 500 JSON error response.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				util.Log(r.Context()).Error("panic recovered",
					"error", rec,
					"method", r.Method,
					"path", r.URL.Path,
				)
				WriteError(w, model.NewInternalError())
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS returns middleware that handles Cross-Origin Resource Sharing based
// on the provided configuration.
func CORS(cfg config.CORSConfig) func(http.Handler) http.Handler {
	origins := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		origins[o] = true
	}
	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")
	maxAge := fmt.Sprintf("%d", cfg.MaxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && origins[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", methods)
				w.Header().Set("Access-Control-Allow-Headers", headers)
				w.Header().Set("Access-Control-Max-Age", maxAge)
				w.Header().Set("Access-Control-Expose-Headers", "X-Correlation-Id")
				w.Header().Set("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestID reads X-Correlation-Id from the request header or generates a
// new one, then stores it in the context and sets the response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Correlation-Id")
		if id == "" {
			id = util.IDString()
		}
		ctx := context.WithValue(r.Context(), correlationIDKey{}, id)
		w.Header().Set("X-Correlation-Id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SecurityHeaders sets standard security response headers on all responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// BuildRequestContextMiddleware returns middleware that constructs a
// model.RequestContext from Frame's security.AuthenticationClaims (set by
// Frame's AuthenticationMiddleware) and standard request headers.
// Extra claim paths (e.g. "email") are looked up in AuthenticationClaims.Ext.
func BuildRequestContextMiddleware(extraClaimPaths map[string]string) func(http.Handler) http.Handler {
	emailPath := "email"
	if p, ok := extraClaimPaths["email"]; ok {
		emailPath = p
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authClaims := security.ClaimsFromContext(r.Context())

			rctx := &model.RequestContext{
				DeviceID:      r.Header.Get("X-Device-Id"),
				Timezone:      r.Header.Get("X-Timezone"),
				Locale:        r.Header.Get("Accept-Language"),
				CorrelationID: CorrelationIDFrom(r.Context()),
				Token:         security.JwtFromContext(r.Context()),
			}

			if authClaims != nil {
				rctx.SubjectID = authClaims.GetProfileID()
				rctx.TenantID = authClaims.GetTenantID()
				rctx.PartitionID = authClaims.GetPartitionID()
				rctx.Roles = authClaims.GetRoles()
				rctx.SessionID = authClaims.GetSessionID()
				rctx.Email = extString(authClaims.Ext, emailPath)
				rctx.Claims = authClaims.Ext
			}

			ctx := model.WithRequestContext(r.Context(), rctx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extString extracts a string value from the Ext claims map.
func extString(ext map[string]any, key string) string {
	if ext == nil {
		return ""
	}
	v, _ := ext[key].(string)
	return v
}

// ResolveCapabilities returns middleware that eagerly resolves capabilities
// for the current user and stores them in the context. If the authorization
// service is unavailable the request fails with 502 so the frontend can
// display a meaningful error instead of rendering an empty navigation tree.
func ResolveCapabilities(resolver model.CapabilityResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if resolver != nil {
				rctx := model.RequestContextFrom(r.Context())
				if rctx != nil {
					caps, err := resolver.Resolve(r.Context(), rctx)
					if err != nil {
						util.Log(r.Context()).Error("capability resolution failed",
							"error", err,
							"subject_id", rctx.SubjectID,
						)
						WriteError(w, model.NewBackendUnavailableError())
						return
					}
					ctx := context.WithValue(r.Context(), capabilitiesKey{}, caps)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// HandlerTimeout returns middleware that sets a context deadline on requests.
func HandlerTimeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if d <= 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestLogging logs each request with method, path, status, and duration.
func RequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		util.Log(r.Context()).Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"duration", time.Since(start),
			"correlation_id", CorrelationIDFrom(r.Context()),
		)
	})
}

// --- helpers ---

// statusWriter wraps http.ResponseWriter to capture the written status code.
type statusWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.written {
		w.status = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}
