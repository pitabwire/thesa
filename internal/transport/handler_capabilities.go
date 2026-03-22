package transport

import (
	"net/http"

	"github.com/pitabwire/thesa/model"
)

// capabilitiesResponse formats the CapabilitySet into the structure the frontend expects.
type capabilitiesResponse struct {
	Capabilities map[string]capabilityValue `json:"capabilities"`
	User         *userCapabilities          `json:"user,omitempty"`
	Tenant       *tenantCapabilities        `json:"tenant,omitempty"`
	App          *appCapabilities           `json:"app,omitempty"`
}

type capabilityValue struct {
	Enabled bool `json:"enabled"`
}

type userCapabilities struct {
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Features    []string `json:"features"`
}

type tenantCapabilities struct {
	TenantID string   `json:"tenantId,omitempty"`
	Features []string `json:"features"`
}

type appCapabilities struct {
	Version      string          `json:"version,omitempty"`
	FeatureFlags map[string]bool `json:"featureFlags,omitempty"`
}

func handleCapabilities(capResolver model.CapabilityResolver, appVersion string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}

		caps, err := capResolver.Resolve(r.Context(), rctx)
		if err != nil {
			WriteError(w, err)
			return
		}

		// Build the capabilities map.
		capMap := make(map[string]capabilityValue, len(caps))
		permissions := make([]string, 0, len(caps))
		for k, v := range caps {
			capMap[k] = capabilityValue{Enabled: v}
			if v {
				permissions = append(permissions, k)
			}
		}

		resp := capabilitiesResponse{
			Capabilities: capMap,
			User: &userCapabilities{
				Roles:       []string{},
				Permissions: permissions,
				Features:    []string{},
			},
			Tenant: &tenantCapabilities{
				TenantID: rctx.TenantID,
				Features: []string{},
			},
			App: &appCapabilities{
				Version: appVersion,
			},
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}
