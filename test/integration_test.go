package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/geogate/internal/api"
	"github.com/alexandrmotologa/geogate/internal/config"
	"github.com/alexandrmotologa/geogate/internal/geoip"
	"github.com/alexandrmotologa/geogate/internal/proxy"
	"github.com/alexandrmotologa/geogate/web"
)

func TestFullIntegrationSuite(t *testing.T) {
	// 1. Setup mock upstream local app (e.g. Next.js / Express)
	var capturedHeaders http.Header
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upstream":"handled","ok":true}`))
	}))
	defer mockUpstream.Close()

	// 2. Initialize GeoGate subsystems
	cfg := &config.Config{
		Port:           8080,
		UpstreamURL:    mockUpstream.URL,
		DefaultCountry: "US",
		EnableCORS:     true,
		MaxBatchSize:   50000,
	}

	engine, err := geoip.NewEngine(geoip.Config{})
	if err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}
	defer engine.Close()

	webFS, err := web.GetFS()
	if err != nil {
		t.Fatalf("Failed to read webFS: %v", err)
	}

	hub := proxy.NewEventHub(50)
	resolver := proxy.NewResolver(engine, cfg.DefaultCountry)
	apiServer := api.NewServer(engine, hub, cfg, webFS)

	proxyHandler, err := proxy.NewProxyHandler(cfg.UpstreamURL, resolver, hub, apiServer)
	if err != nil {
		t.Fatalf("Failed to create proxy handler: %v", err)
	}

	gatewayServer := httptest.NewServer(proxyHandler)
	defer gatewayServer.Close()

	client := gatewayServer.Client()

	// Test 1: Proxy request with simulation override ?simulate_country=JP
	t.Run("Proxy simulation ?simulate_country=JP", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, gatewayServer.URL+"/api/catalog?simulate_country=JP", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
		}

		if capturedHeaders.Get("CF-IPCountry") != "JP" {
			t.Errorf("CF-IPCountry = %q, want JP", capturedHeaders.Get("CF-IPCountry"))
		}
		if capturedHeaders.Get("X-Geo-City") != "Tokyo" {
			t.Errorf("X-Geo-City = %q, want Tokyo", capturedHeaders.Get("X-Geo-City"))
		}
		if capturedHeaders.Get("X-Geo-EU") != "0" {
			t.Errorf("X-Geo-EU = %q, want 0", capturedHeaders.Get("X-Geo-EU"))
		}
		if capturedHeaders.Get("X-ASN") != "AS9370" {
			t.Errorf("X-ASN = %q, want AS9370", capturedHeaders.Get("X-ASN"))
		}
		if capturedHeaders.Get("X-GeoGate-Simulated") != "true" {
			t.Errorf("X-GeoGate-Simulated = %q, want true", capturedHeaders.Get("X-GeoGate-Simulated"))
		}
	})

	// Test 2: Proxy request with header override X-Simulate-Country: RO
	t.Run("Proxy simulation X-Simulate-Country: RO", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, gatewayServer.URL+"/api/vat", nil)
		req.Header.Set("X-Simulate-Country", "RO")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if capturedHeaders.Get("CF-IPCountry") != "RO" {
			t.Errorf("CF-IPCountry = %q, want RO", capturedHeaders.Get("CF-IPCountry"))
		}
		if capturedHeaders.Get("X-Geo-City") != "Bucharest" {
			t.Errorf("X-Geo-City = %q, want Bucharest", capturedHeaders.Get("X-Geo-City"))
		}
		if capturedHeaders.Get("X-Geo-EU") != "1" {
			t.Errorf("X-Geo-EU = %q, want 1", capturedHeaders.Get("X-Geo-EU"))
		}
		if capturedHeaders.Get("X-Geo-Currency") != "RON" {
			t.Errorf("X-Geo-Currency = %q, want RON", capturedHeaders.Get("X-Geo-Currency"))
		}
		if capturedHeaders.Get("X-Geo-PPP-Factor") != "0.48" {
			t.Errorf("X-Geo-PPP-Factor = %q, want 0.48", capturedHeaders.Get("X-Geo-PPP-Factor"))
		}
		if capturedHeaders.Get("X-Geo-PPP-Discount") != "52" {
			t.Errorf("X-Geo-PPP-Discount = %q, want 52", capturedHeaders.Get("X-Geo-PPP-Discount"))
		}
		if capturedHeaders.Get("X-Geo-Connection-Type") != "residential" {
			t.Errorf("X-Geo-Connection-Type = %q, want residential", capturedHeaders.Get("X-Geo-Connection-Type"))
		}
	})

	// Test 3: Proxy request with cookie override
	t.Run("Proxy simulation Cookie geogate_simulate_country=DE", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, gatewayServer.URL+"/dashboard", nil)
		req.AddCookie(&http.Cookie{Name: "geogate_simulate_country", Value: "DE"})
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if capturedHeaders.Get("CF-IPCountry") != "DE" {
			t.Errorf("CF-IPCountry = %q, want DE", capturedHeaders.Get("CF-IPCountry"))
		}
		if capturedHeaders.Get("X-Geo-City") != "Berlin" {
			t.Errorf("X-Geo-City = %q, want Berlin", capturedHeaders.Get("X-Geo-City"))
		}
		if capturedHeaders.Get("X-Geo-EU") != "1" {
			t.Errorf("X-Geo-EU = %q, want 1", capturedHeaders.Get("X-Geo-EU"))
		}
	})

	// Test 4: API Single Lookup /v1/lookup/8.8.8.8
	t.Run("API Single Lookup", func(t *testing.T) {
		resp, err := client.Get(gatewayServer.URL + "/v1/lookup/8.8.8.8")
		if err != nil {
			t.Fatalf("Get lookup failed: %v", err)
		}
		defer resp.Body.Close()

		var res api.LookupResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}
		if res.Record.CountryCode != "US" {
			t.Errorf("Country = %s, want US", res.Record.CountryCode)
		}
		if res.Record.ASN.Number != 15169 {
			t.Errorf("ASN = %d, want 15169", res.Record.ASN.Number)
		}
	})

	// Test 5: High-Throughput Batch Lookup with 1,000 IPs
	t.Run("API Batch Lookup 1000 IPs", func(t *testing.T) {
		const numIPs = 1000
		ips := make([]string, numIPs)
		for i := 0; i < numIPs; i++ {
			ips[i] = fmt.Sprintf("88.198.%d.%d", (i/250)%255, i%255)
		}

		payload, _ := json.Marshal(api.BatchLookupRequest{IPs: ips})
		resp, err := client.Post(gatewayServer.URL+"/v1/batch/lookup", "application/json", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("Batch lookup failed: %v", err)
		}
		defer resp.Body.Close()

		var res api.BatchLookupResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}

		if res.TotalProcessed != numIPs {
			t.Errorf("Processed = %d, want %d", res.TotalProcessed, numIPs)
		}
		t.Logf("1,000 IPs batch duration: %d us, throughput: %.0f IPs/sec", res.DurationMicroseconds, res.ThroughputIPsPerSec)
	})

	// Test 6: API Geofence Evaluation
	t.Run("API Geofence Evaluation", func(t *testing.T) {
		geoReq := api.GeofenceEvaluationRequest{
			Policy: api.GeofencePolicy{
				Name:             "Sanction Check",
				BlockedCountries: []string{"RU", "CN"},
			},
			IPs: []string{"8.8.8.8", "95.173.136.1", "202.96.128.1"},
		}
		payload, _ := json.Marshal(geoReq)
		resp, err := client.Post(gatewayServer.URL+"/v1/geofence/evaluate", "application/json", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("Geofence request failed: %v", err)
		}
		defer resp.Body.Close()

		var res api.GeofenceEvaluationResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}
		if res.Allowed != 1 || res.Blocked != 2 {
			t.Errorf("Expected 1 allowed, 2 blocked, got %d and %d", res.Allowed, res.Blocked)
		}
	})

	// Test 7: Prometheus Metrics endpoint
	t.Run("Prometheus Metrics /metrics", func(t *testing.T) {
		resp, err := client.Get(gatewayServer.URL + "/metrics")
		if err != nil {
			t.Fatalf("Metrics request failed: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if !bytes.Contains(body, []byte("geogate_lookups_total")) {
			t.Errorf("Metrics response missing geogate_lookups_total")
		}
		if !bytes.Contains(body, []byte("geogate_uptime_seconds")) {
			t.Errorf("Metrics response missing geogate_uptime_seconds")
		}
	})

	// Test 8: Embedded Web UI Static Asset Serving
	t.Run("Embedded Web UI Serving", func(t *testing.T) {
		resp, err := client.Get(gatewayServer.URL + "/geogate/")
		if err != nil {
			t.Fatalf("Web UI request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK for Web UI, got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if !bytes.Contains(body, []byte("<title>web</title>")) && !bytes.Contains(body, []byte("<!doctype html>")) {
			t.Errorf("Expected HTML response from embedded Web UI, got %s", string(body[:min(100, len(body))]))
		}
	})

	// Test 9: Purchasing Power Parity API
	t.Run("API PPP Lookup /v1/ppp/RO", func(t *testing.T) {
		resp, err := client.Get(gatewayServer.URL + "/v1/ppp/RO")
		if err != nil {
			t.Fatalf("PPP request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK for PPP endpoint, got %d", resp.StatusCode)
		}

		var pppData map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&pppData); err != nil {
			t.Fatalf("Failed to parse PPP JSON: %v", err)
		}

		if pppData["country_code"] != "RO" {
			t.Errorf("PPP country_code = %v, want RO", pppData["country_code"])
		}
		if pppData["recommended_discount"] != float64(52) {
			t.Errorf("PPP discount = %v, want 52", pppData["recommended_discount"])
		}
	})

	// Test 10: Proxy Request Replay Engine
	t.Run("API Proxy Replay Engine /v1/proxy/replay", func(t *testing.T) {
		replayReq := api.ReplayRequest{
			Method:          "GET",
			Path:            "/api/pricing",
			SimulateCountry: "DE",
		}
		payload, _ := json.Marshal(replayReq)
		resp, err := client.Post(gatewayServer.URL+"/v1/proxy/replay", "application/json", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("Replay request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK for replay, got %d", resp.StatusCode)
		}

		var replayResp api.ReplayResponse
		if err := json.NewDecoder(resp.Body).Decode(&replayResp); err != nil {
			t.Fatalf("Failed to decode replay response: %v", err)
		}

		if !replayResp.Success {
			t.Errorf("Expected replay to succeed, got error: %s", replayResp.Error)
		}
		if replayResp.SimulatedCountry != "DE" {
			t.Errorf("Replay simulated country = %s, want DE", replayResp.SimulatedCountry)
		}
		if replayResp.InjectedHeaders["CF-IPCountry"] != "DE" {
			t.Errorf("Injected header CF-IPCountry = %s, want DE", replayResp.InjectedHeaders["CF-IPCountry"])
		}
	})
}
