package model

import (
	"context"
	"errors"
	"fmt"
)

// RequestContext carries all identity, tenancy, and tracing information for the
// lifetime of an authenticated request. It is immutable after construction and
// safe for concurrent reads.
type RequestContext struct {
	SubjectID     string
	Email         string
	TenantID      string
	PartitionID   string
	Roles         []string
	Claims        map[string]any
	SessionID     string
	DeviceID      string
	CorrelationID string
	TraceID       string
	SpanID        string
	Locale        string
	Timezone      string
}

// Validate checks that all mandatory fields are present.
// SubjectID and TenantID must be non-empty.
func (rc *RequestContext) Validate() error {
	var errs []error
	if rc.SubjectID == "" {
		errs = append(errs, fmt.Errorf("SubjectID is required"))
	}
	if rc.TenantID == "" {
		errs = append(errs, fmt.Errorf("TenantID is required"))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// HasRole returns true if the RequestContext contains the given role.
func (rc *RequestContext) HasRole(role string) bool {
	for _, r := range rc.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// Claim returns the value of the given claim key, or nil if not present.
func (rc *RequestContext) Claim(key string) any {
	if rc.Claims == nil {
		return nil
	}
	return rc.Claims[key]
}

type contextKey struct{}

// WithRequestContext attaches a RequestContext to the given context.
func WithRequestContext(ctx context.Context, rctx *RequestContext) context.Context {
	return context.WithValue(ctx, contextKey{}, rctx)
}

// RequestContextFrom extracts the RequestContext from the context, or returns nil
// if not present.
func RequestContextFrom(ctx context.Context) *RequestContext {
	rctx, _ := ctx.Value(contextKey{}).(*RequestContext)
	return rctx
}

// MustRequestContext extracts the RequestContext from the context, panicking if
// it is not present. This is safe to call in handlers that are guaranteed to run
// behind the authentication middleware.
func MustRequestContext(ctx context.Context) *RequestContext {
	rctx := RequestContextFrom(ctx)
	if rctx == nil {
		panic("model: RequestContext not found in context")
	}
	return rctx
}
