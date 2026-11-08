package geoip

import (
	"net"
	"testing"
	"time"
)

func TestEngineLookups(t *testing.T) {
	engine, err := NewEngine(Config{})
	if err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}
	defer engine.Close()

	tests := []struct {
		name        string
		ip          string
		wantCountry string
		wantCity    string
		wantIsEU    bool
		wantASN     uint
	}{
		{
			name:        "Google DNS",
			ip:          "8.8.8.8",
			wantCountry: "US",
			wantCity:    "Mountain View",
			wantIsEU:    false,
			wantASN:     15169,
		},
		{
			name:        "Hetzner DE",
			ip:          "88.198.24.1",
			wantCountry: "DE",
			wantCity:    "Berlin",
			wantIsEU:    true,
			wantASN:     24940,
		},
		{
			name:        "RCS & RDS RO",
			ip:          "86.120.10.5",
			wantCountry: "RO",
			wantCity:    "Bucharest",
			wantIsEU:    true,
			wantASN:     8708,
		},
		{
			name:        "Sakura Japan",
			ip:          "133.242.1.1",
			wantCountry: "JP",
			wantCity:    "Tokyo",
			wantIsEU:    false,
			wantASN:     9370,
		},
		{
			name:        "Localhost Loopback",
			ip:          "127.0.0.1",
			wantCountry: "US",
			wantCity:    "Local Development",
			wantIsEU:    false,
			wantASN:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, err := engine.LookupString(tt.ip)
			if err != nil {
				t.Fatalf("LookupString(%q) unexpected error: %v", tt.ip, err)
			}
			if rec.CountryCode != tt.wantCountry {
				t.Errorf("CountryCode = %q, want %q", rec.CountryCode, tt.wantCountry)
			}
			if rec.City != tt.wantCity {
				t.Errorf("City = %q, want %q", rec.City, tt.wantCity)
			}
			if rec.IsEU != tt.wantIsEU {
				t.Errorf("IsEU = %v, want %v", rec.IsEU, tt.wantIsEU)
			}
			if rec.ASN.Number != tt.wantASN {
				t.Errorf("ASN = %d, want %d", rec.ASN.Number, tt.wantASN)
			}
		})
	}
}

func TestEngineSimulations(t *testing.T) {
	engine, err := NewEngine(Config{})
	if err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}
	defer engine.Close()

	t.Run("LookupCountry JP", func(t *testing.T) {
		rec, err := engine.LookupCountry("JP")
		if err != nil {
			t.Fatalf("LookupCountry error: %v", err)
		}
		if rec.CountryCode != "JP" || rec.CountryName != "Japan" {
			t.Errorf("expected Japan, got %+v", rec)
		}
		if rec.Currency != "JPY" {
			t.Errorf("expected JPY, got %s", rec.Currency)
		}
	})

	t.Run("LookupCity Bucharest", func(t *testing.T) {
		rec, err := engine.LookupCity("Bucharest")
		if err != nil {
			t.Fatalf("LookupCity error: %v", err)
		}
		if rec.CountryCode != "RO" || rec.City != "Bucharest" {
			t.Errorf("expected Bucharest RO, got %+v", rec)
		}
		if !rec.IsEU {
			t.Errorf("expected RO to be EU")
		}
	})
}

func BenchmarkEngineLookup(b *testing.B) {
	engine, err := NewEngine(Config{})
	if err != nil {
		b.Fatalf("Failed to initialize engine: %v", err)
	}
	defer engine.Close()

	testIP := net.ParseIP("8.8.8.8")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rec, err := engine.Lookup(testIP)
		if err != nil || rec == nil {
			b.Fatalf("Lookup error: %v", err)
		}
	}
}

func TestEngineLatencyUnder8Microseconds(t *testing.T) {
	engine, err := NewEngine(Config{})
	if err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}
	defer engine.Close()

	avgDuration, err := engine.BenchmarkSingleLookup(10000)
	if err != nil {
		t.Fatalf("Benchmark error: %v", err)
	}

	t.Logf("Average lookup duration: %v (%d ns)", avgDuration, avgDuration.Nanoseconds())

	// Specification requires < 8 microseconds (8000 nanoseconds)
	if avgDuration > 8*time.Microsecond {
		t.Errorf("Engine lookup took %v, which exceeds the required 8 microseconds", avgDuration)
	}
}
