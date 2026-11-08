package config

import (
	"flag"
	"os"
	"strconv"
	"strings"
)

// Config represents runtime settings for GeoGate.
type Config struct {
	Port           int    `json:"port"`
	UpstreamURL    string `json:"upstream_url"`
	CityDBPath     string `json:"city_db_path"`
	AsnDBPath      string `json:"asn_db_path"`
	DefaultCountry string `json:"default_country"`
	EnableCORS     bool   `json:"enable_cors"`
	MaxBatchSize   int    `json:"max_batch_size"`
}

// LoadConfig parses command line flags and environment variables.
func LoadConfig() *Config {
	cfg := &Config{
		Port:           8080,
		UpstreamURL:    "http://localhost:3000",
		CityDBPath:     "",
		AsnDBPath:      "",
		DefaultCountry: "US",
		EnableCORS:     true,
		MaxBatchSize:   100000,
	}

	// Environment variable overrides
	if envPort := os.Getenv("GEOGATE_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			cfg.Port = p
		}
	}
	if envUpstream := os.Getenv("GEOGATE_UPSTREAM"); envUpstream != "" {
		cfg.UpstreamURL = envUpstream
	}
	if envCityDB := os.Getenv("GEOGATE_CITY_DB"); envCityDB != "" {
		cfg.CityDBPath = envCityDB
	}
	if envAsnDB := os.Getenv("GEOGATE_ASN_DB"); envAsnDB != "" {
		cfg.AsnDBPath = envAsnDB
	}
	if envCountry := os.Getenv("GEOGATE_DEFAULT_COUNTRY"); envCountry != "" {
		cfg.DefaultCountry = envCountry
	}

	// CLI flags override environment variables
	flag.IntVar(&cfg.Port, "port", cfg.Port, "HTTP server listening port (default: 8080)")
	flag.StringVar(&cfg.UpstreamURL, "upstream", cfg.UpstreamURL, "Upstream application URL for reverse proxying")
	flag.StringVar(&cfg.CityDBPath, "city-db", cfg.CityDBPath, "Path to MaxMind GeoLite2/GeoIP2 City MMDB file")
	flag.StringVar(&cfg.AsnDBPath, "asn-db", cfg.AsnDBPath, "Path to MaxMind GeoLite2/GeoIP2 ASN MMDB file")
	flag.StringVar(&cfg.DefaultCountry, "default-country", cfg.DefaultCountry, "Default ISO country code for simulation fallback")
	flag.BoolVar(&cfg.EnableCORS, "cors", cfg.EnableCORS, "Enable CORS headers for API endpoints")
	flag.IntVar(&cfg.MaxBatchSize, "max-batch", cfg.MaxBatchSize, "Maximum IPs allowed in a single batch lookup request")

	if !flag.Parsed() {
		flag.Parse()
	}

	// Ensure port doesn't have colon prefix if provided via env
	cfg.DefaultCountry = strings.ToUpper(strings.TrimSpace(cfg.DefaultCountry))

	return cfg
}
