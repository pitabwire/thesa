package model

import (
	"context"
	"testing"
)

func TestRequestContext_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rc      *RequestContext
		wantErr bool
	}{
		{
			name: "valid context",
			rc: &RequestContext{
				SubjectID: "user-1",
				TenantID:  "tenant-1",
			},
			wantErr: false,
		},
		{
			name: "missing SubjectID",
			rc: &RequestContext{
				TenantID: "tenant-1",
			},
			wantErr: true,
		},
		{
			name: "missing TenantID",
			rc: &RequestContext{
				SubjectID: "user-1",
			},
			wantErr: true,
		},
		{
			name:    "missing both",
			rc:      &RequestContext{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rc.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequestContext_HasRole(t *testing.T) {
	rc := &RequestContext{
		Roles: []string{"admin", "editor"},
	}
	if !rc.HasRole("admin") {
		t.Error("HasRole(admin) = false, want true")
	}
	if !rc.HasRole("editor") {
		t.Error("HasRole(editor) = false, want true")
	}
	if rc.HasRole("viewer") {
		t.Error("HasRole(viewer) = true, want false")
	}
}

func TestRequestContext_HasRole_empty(t *testing.T) {
	rc := &RequestContext{}
	if rc.HasRole("admin") {
		t.Error("HasRole(admin) on empty roles = true, want false")
	}
}

func TestRequestContext_Claim(t *testing.T) {
	rc := &RequestContext{
		Claims: map[string]any{
			"email": "user@example.com",
			"count": 42,
		},
	}
	if got := rc.Claim("email"); got != "user@example.com" {
		t.Errorf("Claim(email) = %v, want user@example.com", got)
	}
	if got := rc.Claim("count"); got != 42 {
		t.Errorf("Claim(count) = %v, want 42", got)
	}
	if got := rc.Claim("missing"); got != nil {
		t.Errorf("Claim(missing) = %v, want nil", got)
	}
}

func TestRequestContext_Claim_nil_map(t *testing.T) {
	rc := &RequestContext{}
	if got := rc.Claim("any"); got != nil {
		t.Errorf("Claim(any) on nil claims = %v, want nil", got)
	}
}

func TestWithRequestContext_and_RequestContextFrom(t *testing.T) {
	rctx := &RequestContext{
		SubjectID: "user-1",
		TenantID:  "tenant-1",
	}
	ctx := WithRequestContext(context.Background(), rctx)
	got := RequestContextFrom(ctx)
	if got != rctx {
		t.Errorf("RequestContextFrom() = %v, want %v", got, rctx)
	}
}

func TestRequestContextFrom_absent(t *testing.T) {
	got := RequestContextFrom(context.Background())
	if got != nil {
		t.Errorf("RequestContextFrom(empty context) = %v, want nil", got)
	}
}

func TestMustRequestContext_present(t *testing.T) {
	rctx := &RequestContext{
		SubjectID: "user-1",
		TenantID:  "tenant-1",
	}
	ctx := WithRequestContext(context.Background(), rctx)
	got := MustRequestContext(ctx)
	if got != rctx {
		t.Errorf("MustRequestContext() = %v, want %v", got, rctx)
	}
}

func TestMustRequestContext_absent_panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRequestContext(empty context) did not panic")
		}
	}()
	MustRequestContext(context.Background())
}
