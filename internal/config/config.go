package config

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// CustomRule allows mapping arbitrary local subnets or IP ranges to specific geographic locations.
type CustomRule struct {
	CIDR         string `yaml:"cidr" json:"cidr"`
	Country      string `yaml:"country" json:"country"`
	City         string `yaml:"city" json:"city"`
	ASN          uint   `yaml:"asn" json:"asn"`
	ASNOrg       string `yaml:"asn_org" json:"asn_org"`
	IsDatacenter bool   `yaml:"is_datacenter" json:"is_datacenter"`
}

// Config represents runtime settings for GeoGate.
type Config struct {
	Port            int          `yaml:"port" json:"port"`
	UpstreamURL     string       `yaml:"upstream" json:"upstream_url"`
	CityDBPath      string       `yaml:"city_db" json:"city_db_path"`
	AsnDBPath       string       `yaml:"asn_db" json:"asn_db_path"`
	DefaultCountry  string       `yaml:"default_country" json:"default_country"`
	EnableCORS      bool         `yaml:"cors" json:"enable_cors"`
	MaxBatchSize    int          `yaml:"max_batch" json:"max_batch_size"`
	SimulateLatency string       `yaml:"latency" json:"simulate_latency"`
	ConfigFile      string       `yaml:"-" json:"config_file,omitempty"`
	Rules           []CustomRule `yaml:"rules" json:"rules"`
}

// LoadConfig parses configuration files, environment variables, and CLI flags.
func LoadConfig() *Config {
	cfg := &Config{
		Port:            8080,
		UpstreamURL:     "http://localhost:3000",
		CityDBPath:      "",
		AsnDBPath:       "",
		DefaultCountry:  "US",
		EnableCORS:      true,
		MaxBatchSize:    100000,
		SimulateLatency: "",
		ConfigFile:      "",
		Rules:           []CustomRule{},
	}

	// 1. Detect optional -config CLI flag first
	configPath := ""
	for i, arg := range os.Args {
		if (arg == "-config" || arg == "--config") && i+1 < len(os.Args) {
			configPath = os.Args[i+1]
			break
		}
	}

	if configPath == "" {
		if _, err := os.Stat("geogate.yaml"); err == nil {
			configPath = "geogate.yaml"
		} else if _, err := os.Stat("geogate.yml"); err == nil {
			configPath = "geogate.yml"
		} else if _, err := os.Stat("geogate.json"); err == nil {
			configPath = "geogate.json"
		}
	}

	if configPath != "" {
		cfg.ConfigFile = configPath
		loadConfigFile(configPath, cfg)
	}

	// 2. Environment variable overrides
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
	if envLatency := os.Getenv("GEOGATE_SIMULATE_LATENCY"); envLatency != "" {
		cfg.SimulateLatency = envLatency
	}

	// 3. CLI flags override environment and config files
	flag.StringVar(&cfg.ConfigFile, "config", cfg.ConfigFile, "Path to geogate.yaml or geogate.json config file")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "HTTP server listening port (default: 8080)")
	flag.StringVar(&cfg.UpstreamURL, "upstream", cfg.UpstreamURL, "Upstream application URL for reverse proxying")
	flag.StringVar(&cfg.CityDBPath, "city-db", cfg.CityDBPath, "Path to MaxMind GeoLite2/GeoIP2 City MMDB file")
	flag.StringVar(&cfg.AsnDBPath, "asn-db", cfg.AsnDBPath, "Path to MaxMind GeoLite2/GeoIP2 ASN MMDB file")
	flag.StringVar(&cfg.DefaultCountry, "default-country", cfg.DefaultCountry, "Default ISO country code for simulation fallback")
	flag.BoolVar(&cfg.EnableCORS, "cors", cfg.EnableCORS, "Enable CORS headers for API endpoints")
	flag.IntVar(&cfg.MaxBatchSize, "max-batch", cfg.MaxBatchSize, "Maximum IPs allowed in a single batch lookup request")
	flag.StringVar(&cfg.SimulateLatency, "simulate-latency", cfg.SimulateLatency, "Default artificial network latency (e.g. 'auto', '150ms')")

	if !flag.Parsed() {
		flag.Parse()
	}

	cfg.DefaultCountry = strings.ToUpper(strings.TrimSpace(cfg.DefaultCountry))

	return cfg
}

func loadConfigFile(path string, cfg *Config) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	if strings.HasSuffix(path, ".json") {
		_ = json.Unmarshal(data, cfg)
	} else {
		_ = yaml.Unmarshal(data, cfg)
	}
}
