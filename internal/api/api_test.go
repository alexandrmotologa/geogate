package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/geogate/internal/config"
	"github.com/alexandrmotologa/geogate/internal/geoip"
	"github.com/alexandrmotologa/geogate/internal/proxy"
)

func setupTestServer(t *testing.T) *Server {
	eng, err := geoip.NewEngine(geoip.Config{})
	if err != nil {
		t.Fatalf("Failed to init engine: %v", err)
	}
	hub := proxy.NewEventHub(10)
	cfg := &config.Config{
		DefaultCountry: "US",
		EnableCORS:     true,
		MaxBatchSize:   10000,
	}
	return NewServer(eng, hub, cfg, nil)
}

func TestSingleLookupEndpoint(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/lookup/8.8.8.8", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp LookupResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Record.CountryCode != "US" {
		t.Errorf("Expected country US, got %s", resp.Record.CountryCode)
	}
	if resp.Record.ASN.Number != 15169 {
		t.Errorf("Expected ASN 15169, got %d", resp.Record.ASN.Number)
	}
}

func TestBatchLookupEndpoint(t *testing.T) {
	srv := setupTestServer(t)

	ips := []string{"8.8.8.8", "88.198.24.1", "86.120.10.5", "133.242.1.1"}
	payload, _ := json.Marshal(BatchLookupRequest{IPs: ips})

	req := httptest.NewRequest(http.MethodPost, "/v1/batch/lookup", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	t.Logf("Response body: %s", w.Body.String())

	var resp BatchLookupResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode batch response: %v", err)
	}

	if resp.TotalProcessed != len(ips) {
		t.Errorf("TotalProcessed = %d, want %d", resp.TotalProcessed, len(ips))
	}
	if resp.Results[0].CountryCode != "US" {
		t.Errorf("IP 0 country = %s, want US", resp.Results[0].CountryCode)
	}
	if resp.Results[1].CountryCode != "DE" {
		t.Errorf("IP 1 country = %s, want DE", resp.Results[1].CountryCode)
	}
	if resp.ThroughputIPsPerSec <= 0 {
		t.Errorf("Throughput should be positive, got %f", resp.ThroughputIPsPerSec)
	}
}

func TestGeofenceEvaluationEndpoint(t *testing.T) {
	srv := setupTestServer(t)

	reqPayload := GeofenceEvaluationRequest{
		Policy: GeofencePolicy{
			Name:             "EU Compliance Only",
			RequireEU:        true,
			BlockedCountries: []string{"CN", "RU"},
		},
		IPs: []string{"88.198.24.1", "8.8.8.8"}, // DE (EU) should pass, US should fail requireEU
	}

	data, _ := json.Marshal(reqPayload)
	req := httptest.NewRequest(http.MethodPost, "/v1/geofence/evaluate", bytes.NewReader(data))
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp GeofenceEvaluationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode geofence response: %v", err)
	}

	if resp.Allowed != 1 || resp.Blocked != 1 {
		t.Errorf("Expected 1 allowed and 1 blocked, got allowed=%d, blocked=%d", resp.Allowed, resp.Blocked)
	}

	// 88.198.24.1 is DE (EU) -> Allowed
	if !resp.Results[0].Allowed {
		t.Errorf("Expected DE IP to be allowed under RequireEU")
	}

	// 8.8.8.8 is US -> Blocked (RequireEU)
	if resp.Results[1].Allowed {
		t.Errorf("Expected US IP to be blocked under RequireEU")
	}
}
