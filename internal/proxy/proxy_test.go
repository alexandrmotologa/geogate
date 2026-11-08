package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/geogate/internal/geoip"
)

func TestProxyHeaderInjectionAndOverrides(t *testing.T) {
	eng, err := geoip.NewEngine(geoip.Config{})
	if err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}
	defer eng.Close()

	resolver := NewResolver(eng, "US")

	// Upstream mock server that verifies injected headers
	var receivedHeaders http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-ok"))
	}))
	defer upstream.Close()

	hub := NewEventHub(10)
	handler, err := NewProxyHandler(upstream.URL, resolver, hub, nil)
	if err != nil {
		t.Fatalf("Failed to create proxy handler: %v", err)
	}

	t.Run("Query Param Simulation Override JP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/api/pricing?simulate_country=JP", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Proxy returned code %d", w.Code)
		}
		body, _ := io.ReadAll(w.Body)
		if string(body) != "upstream-ok" {
			t.Errorf("Expected upstream-ok, got %s", string(body))
		}

		if receivedHeaders.Get("CF-IPCountry") != "JP" {
			t.Errorf("CF-IPCountry = %q, want JP", receivedHeaders.Get("CF-IPCountry"))
		}
		if receivedHeaders.Get("X-Country-Name") != "Japan" {
			t.Errorf("X-Country-Name = %q, want Japan", receivedHeaders.Get("X-Country-Name"))
		}
		if receivedHeaders.Get("X-Geo-City") != "Tokyo" {
			t.Errorf("X-Geo-City = %q, want Tokyo", receivedHeaders.Get("X-Geo-City"))
		}
		if receivedHeaders.Get("X-Geo-EU") != "0" {
			t.Errorf("X-Geo-EU = %q, want 0", receivedHeaders.Get("X-Geo-EU"))
		}
		if receivedHeaders.Get("X-GeoGate-Simulated") != "true" {
			t.Errorf("X-GeoGate-Simulated = %q, want true", receivedHeaders.Get("X-GeoGate-Simulated"))
		}
	})

	t.Run("Header Simulation Override RO", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/checkout", nil)
		req.Header.Set("X-Simulate-Country", "RO")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if receivedHeaders.Get("CF-IPCountry") != "RO" {
			t.Errorf("CF-IPCountry = %q, want RO", receivedHeaders.Get("CF-IPCountry"))
		}
		if receivedHeaders.Get("X-Geo-City") != "Bucharest" {
			t.Errorf("X-Geo-City = %q, want Bucharest", receivedHeaders.Get("X-Geo-City"))
		}
		if receivedHeaders.Get("X-Geo-EU") != "1" {
			t.Errorf("X-Geo-EU = %q, want 1", receivedHeaders.Get("X-Geo-EU"))
		}
	})
}
