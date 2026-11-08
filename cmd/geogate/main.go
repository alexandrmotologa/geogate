package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/alexandrmotologa/geogate/internal/api"
	"github.com/alexandrmotologa/geogate/internal/config"
	"github.com/alexandrmotologa/geogate/internal/geoip"
	"github.com/alexandrmotologa/geogate/internal/proxy"
	"github.com/alexandrmotologa/geogate/web"
)

const (
	Version = "1.0.0"
	Banner  = "   ______            ______      __     \n" +
		"  / ____/___  ____  / ____/___ _/ /____ \n" +
		" / / __/ __ \\/ __ \\/ / __/ __ `/ __/ _ \\\n" +
		"/ /_/ / /_/ / /_/ / /_/ / /_/ / /_/  __/\n" +
		"\\____/\\____/\\____/\\____/\\__,_/\\__/\\___/ \n" +
		"  Local IP Geolocation & Edge Gateway\n\n"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			printVersion()
			return
		case "lookup":
			runLookupCommand(os.Args[2:])
			return
		case "benchmark":
			runBenchmarkCommand(os.Args[2:])
			return
		case "help", "-h", "--help":
			printUsage()
			return
		}
	}

	runServer()
}

func printVersion() {
	fmt.Printf("GeoGate %s (%s/%s, %s)\n", Version, runtime.GOOS, runtime.GOARCH, runtime.Version())
}

func printUsage() {
	fmt.Printf(`GeoGate %s: Self-Hosted Local IP Geolocation, ASN Intelligence and Edge Geofencing Gateway

Usage:
  geogate [command] [flags]

Commands:
  serve              Start reverse proxy, Edge header injection, and Web Map Studio (default)
  lookup <ip>        Resolve geographic and network metadata for a single IP
  benchmark [count]  Execute in-memory lookup benchmark asserting sub-8us latency
  version            Print GeoGate version and build details

Flags (for serve):
  -port <port>             HTTP listening port (default: 8080, env: GEOGATE_PORT)
  -upstream <url>          Upstream application URL (default: http://localhost:3000, env: GEOGATE_UPSTREAM)
  -default-country <code>  Fallback ISO country code for simulation (default: US, env: GEOGATE_DEFAULT_COUNTRY)
  -city-db <path>          Path to external MaxMind GeoLite2/GeoIP2 City MMDB file
  -asn-db <path>           Path to external MaxMind GeoLite2/GeoIP2 ASN MMDB file
  -cors                    Enable CORS headers for all API requests (default: true)
  -max-batch <n>           Maximum batch size for /v1/batch/lookup (default: 100000)

Examples:
  geogate serve -upstream http://localhost:3000 -port 8080
  geogate lookup 8.8.8.8
  geogate benchmark 100000
`, Version)
}

func runLookupCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: Missing target IP address. Example: geogate lookup 8.8.8.8")
		os.Exit(1)
	}

	ip := strings.TrimSpace(args[0])
	engine, err := geoip.NewEngine(geoip.Config{})
	if err != nil {
		fmt.Printf("Failed to initialize engine: %v\n", err)
		os.Exit(1)
	}
	defer engine.Close()

	rec, err := engine.LookupString(ip)
	if err != nil {
		fmt.Printf("Lookup failed: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(rec, "", "  ")
	fmt.Println(string(out))
}

func runBenchmarkCommand(args []string) {
	count := 50000
	if len(args) > 0 {
		var parsed int
		if _, err := fmt.Sscanf(args[0], "%d", &parsed); err == nil && parsed > 0 {
			count = parsed
		}
	}

	fmt.Printf("Initializing in-memory trie engine and running %d lookups...\n", count)
	engine, err := geoip.NewEngine(geoip.Config{})
	if err != nil {
		fmt.Printf("Failed to initialize engine: %v\n", err)
		os.Exit(1)
	}
	defer engine.Close()

	avgDuration, err := engine.BenchmarkSingleLookup(count)
	if err != nil {
		fmt.Printf("Benchmark error: %v\n", err)
		os.Exit(1)
	}

	ns := avgDuration.Nanoseconds()
	us := float64(ns) / 1000.0
	ipsPerSec := float64(time.Second) / float64(avgDuration)

	fmt.Println("-----------------------------------------------------------------")
	fmt.Printf("  Total Lookups:          %d\n", count)
	fmt.Printf("  Average Latency:        %.3f microseconds (%d ns)\n", us, ns)
	fmt.Printf("  Throughput (Single Core): %.0f lookups/second\n", ipsPerSec)
	fmt.Printf("  Target Requirement:     < 8.000 microseconds\n")
	if avgDuration <= 8*time.Microsecond {
		fmt.Printf("  Status:                 PASSED (%.1fx faster than spec)\n", 8000.0/float64(ns))
	} else {
		fmt.Println("  Status:                 FAILED")
	}
	fmt.Println("-----------------------------------------------------------------")
}

func runServer() {
	// If first argument is 'serve', strip it before parsing flags
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	cfg := config.LoadConfig()

	fmt.Print(Banner)
	fmt.Printf("  Version:        %s\n", Version)
	fmt.Printf("  Port:           %d\n", cfg.Port)
	fmt.Printf("  Upstream App:   %s\n", cfg.UpstreamURL)
	fmt.Printf("  Default Locale: %s\n", cfg.DefaultCountry)

	// Initialize in-memory engine
	engine, err := geoip.NewEngine(geoip.Config{
		CityDBPath: cfg.CityDBPath,
		AsnDBPath:  cfg.AsnDBPath,
	})
	if err != nil {
		fmt.Printf("Fatal: Engine initialization failed: %v\n", err)
		os.Exit(1)
	}
	defer engine.Close()

	if engine.HasExternalDB() {
		fmt.Printf("  External DB:    Loaded (%s)\n", cfg.CityDBPath)
	} else {
		fmt.Printf("  Database:       Embedded High-Speed Radix Trie (Zero External Dependencies)\n")
	}

	// Warm-up latency test
	latency, _ := engine.BenchmarkSingleLookup(1000)
	fmt.Printf("  Engine Latency: %.3f microseconds (In-Memory Trie)\n", float64(latency.Nanoseconds())/1000.0)

	// Memory check
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("  RAM Footprint:  %.2f MB (Spec: < 35MB RAM)\n", float64(m.Alloc)/1024.0/1024.0)

	// Embedded web assets
	webFS, err := web.GetFS()
	if err != nil {
		fmt.Printf("Notice: Embedded web assets not loaded: %v\n", err)
	}

	// Hub & Components
	hub := proxy.NewEventHub(150)
	resolver := proxy.NewResolver(engine, cfg.DefaultCountry)
	apiServer := api.NewServer(engine, hub, cfg, webFS)

	proxyHandler, err := proxy.NewProxyHandler(cfg.UpstreamURL, resolver, hub, apiServer)
	if err != nil {
		fmt.Printf("Fatal: Failed to initialize reverse proxy: %v\n", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      proxyHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Println("-----------------------------------------------------------------")
	fmt.Printf("  GeoGate Studio UI:     http://localhost:%d/\n", cfg.Port)
	fmt.Printf("  Edge Proxy Forwarder:  http://localhost:%d/ (forwarding to %s)\n", cfg.Port, cfg.UpstreamURL)
	fmt.Printf("  High-Throughput API:   http://localhost:%d/v1/batch/lookup\n", cfg.Port)
	fmt.Printf("  Prometheus Metrics:    http://localhost:%d/metrics\n", cfg.Port)
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("Ready for incoming traffic. Press Ctrl+C to terminate.")

	// Graceful shutdown handling
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server failure: %v\n", err)
			os.Exit(1)
		}
	}()

	<-stopChan
	fmt.Println("\nShutting down GeoGate gateway daemon gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = server.Shutdown(ctx)
	fmt.Println("GeoGate shutdown complete.")
}
