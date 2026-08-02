// Package util holds small HTTP helpers shared by the services.
package util

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// maxResponseBytes caps how much an upstream service can make us buffer.
const maxResponseBytes = 32 << 20 // 32 MiB

// PostJSON posts doc as JSON to url and returns the response body.
//
// The caller's context governs the request, so a slow embedding service cannot
// pin a request goroutine indefinitely.
func PostJSON(ctx context.Context, client *http.Client, url string, doc interface{}) ([]byte, error) {
	requestBytes, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("call %s: unexpected status %s", url, res.Status)
	}

	return io.ReadAll(io.LimitReader(res.Body, maxResponseBytes))
}

// GZIPResp writes respData to w as a gzipped JSON response.
func GZIPResp(w http.ResponseWriter, respData []byte) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(respData); err != nil {
		http.Error(w, "Failed to compress the content", http.StatusInternalServerError)
		return
	}
	if err := gz.Close(); err != nil {
		http.Error(w, "Failed to close the gzip writer", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "gzip")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(buf.Bytes()); err != nil {
		// The header is already out; nothing left to do but stop.
		return
	}
}
