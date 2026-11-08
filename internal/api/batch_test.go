package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkBatchLookupLarge(b *testing.B) {
	srv := setupTestServer(&testing.T{})

	// Generate 10,000 realistic IPs
	const count = 10000
	ips := make([]string, count)
	for i := 0; i < count; i++ {
		ips[i] = fmt.Sprintf("88.198.%d.%d", (i/250)%255, i%255)
	}

	payload, err := json.Marshal(BatchLookupRequest{IPs: ips})
	if err != nil {
		b.Fatalf("Failed to marshal payload: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/batch/lookup", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}
