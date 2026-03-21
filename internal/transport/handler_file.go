package transport

import (
	"io"
	"net/http"

	"github.com/pitabwire/thesa/internal/config"
	"github.com/pitabwire/thesa/model"
)

// handleUpload proxies multipart file uploads to the files backend service.
// The uploaded file is streamed directly to the backend without buffering
// the entire file in memory.
func handleUpload(filesSvc config.ServiceConfig) http.HandlerFunc {
	client := &http.Client{Timeout: filesSvc.Timeout}

	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}

		if filesSvc.BaseURL == "" {
			WriteError(w, model.NewBadRequestError("file service not configured"))
			return
		}

		// Forward the multipart request to the files service.
		proxyURL := filesSvc.BaseURL + "/upload"

		proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, proxyURL, r.Body)
		if err != nil {
			WriteError(w, model.NewInternalError())
			return
		}

		// Copy content-type (preserves multipart boundary).
		proxyReq.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		if rctx.Token != "" {
			proxyReq.Header.Set("Authorization", "Bearer "+rctx.Token)
		}
		proxyReq.Header.Set("X-Tenant-Id", rctx.TenantID)
		proxyReq.Header.Set("X-Partition-Id", rctx.PartitionID)
		proxyReq.Header.Set("X-Correlation-Id", rctx.CorrelationID)

		resp, err := client.Do(proxyReq)
		if err != nil {
			WriteError(w, model.NewBackendUnavailableError())
			return
		}
		defer resp.Body.Close()

		// Stream the response back to the client.
		for _, key := range []string{"Content-Type", "Content-Length"} {
			if v := resp.Header.Get(key); v != "" {
				w.Header().Set(key, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

// handleDownload proxies file download requests to the files backend service.
// The file content is streamed directly to the client.
func handleDownload(filesSvc config.ServiceConfig) http.HandlerFunc {
	client := &http.Client{Timeout: filesSvc.Timeout}

	return func(w http.ResponseWriter, r *http.Request) {
		rctx := model.RequestContextFrom(r.Context())
		if rctx == nil {
			WriteError(w, model.NewUnauthorizedError("missing request context"))
			return
		}

		if filesSvc.BaseURL == "" {
			WriteError(w, model.NewBadRequestError("file service not configured"))
			return
		}

		fileID := r.PathValue("fileId")
		proxyURL := filesSvc.BaseURL + "/download/" + fileID

		proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, proxyURL, nil)
		if err != nil {
			WriteError(w, model.NewInternalError())
			return
		}

		if rctx.Token != "" {
			proxyReq.Header.Set("Authorization", "Bearer "+rctx.Token)
		}
		proxyReq.Header.Set("X-Tenant-Id", rctx.TenantID)
		proxyReq.Header.Set("X-Partition-Id", rctx.PartitionID)
		proxyReq.Header.Set("X-Correlation-Id", rctx.CorrelationID)

		resp, err := client.Do(proxyReq)
		if err != nil {
			WriteError(w, model.NewBackendUnavailableError())
			return
		}
		defer resp.Body.Close()

		// Stream the file response back to the client.
		for _, key := range []string{"Content-Type", "Content-Length", "Content-Disposition"} {
			if v := resp.Header.Get(key); v != "" {
				w.Header().Set(key, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}
