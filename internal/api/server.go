package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/geogate/internal/config"
	"github.com/alexandrmotologa/geogate/internal/geoip"
	"github.com/alexandrmotologa/geogate/internal/proxy"
	"github.com/alexandrmotologa/geogate/internal/telemetry"
)

// Server coordinates the GeoGate API endpoints, metrics, and web dashboard.
type Server struct {
	engine       *geoip.Engine
	hub          *proxy.EventHub
	cfg          *config.Config
	startTime    time.Time
	maxBatchSize int
	webFS        fs.FS
	mux          *http.ServeMux
}

// NewServer initializes the internal HTTP router and API handlers.
func NewServer(engine *geoip.Engine, hub *proxy.EventHub, cfg *config.Config, webFS fs.FS) *Server {
	s := &Server{
		engine:       engine,
		hub:          hub,
		cfg:          cfg,
		startTime:    time.Now(),
		maxBatchSize: cfg.MaxBatchSize,
		webFS:        webFS,
		mux:          http.NewServeMux(),
	}

	s.registerRoutes()
	return s
}

// registerRoutes sets up routing table.
func (s *Server) registerRoutes() {
	// Health & Metrics
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		s.respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	metricsHandler := telemetry.NewMetricsHandler(s.engine, s.startTime)
	s.mux.Handle("/metrics", metricsHandler)

	// API Endpoints
	s.mux.HandleFunc("/v1/lookup/", s.HandleSingleLookup)
	s.mux.HandleFunc("/v1/batch/lookup", s.HandleBatchLookup)
	s.mux.HandleFunc("/v1/geofence/evaluate", s.HandleGeofenceEvaluate)
	s.mux.HandleFunc("/v1/countries", s.HandleGetCountries)
	s.mux.HandleFunc("/v1/cities", s.HandleGetCities)
	s.mux.HandleFunc("/v1/status", s.HandleGetStatus)
	s.mux.HandleFunc("/v1/headers/simulate", s.HandleSimulateHeaders)

	// Server-Sent Events stream for live proxy transactions
	s.mux.HandleFunc("/v1/events", s.hub.ServeSSE)

	// Web Dashboard Static Assets
	if s.webFS != nil {
		fileServer := http.FileServer(http.FS(s.webFS))
		s.mux.HandleFunc("/geogate/", func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/geogate")
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
		})
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/v1/") || r.URL.Path == "/metrics" || r.URL.Path == "/health" {
				return
			}
			fileServer.ServeHTTP(w, r)
		})
	}
}

// ServeHTTP delegates to internal mux, wrapping CORS headers when configured.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.cfg.EnableCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Simulate-Country, X-Simulate-IP, X-Simulate-City")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	s.mux.ServeHTTP(w, r)
}

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		http.Error(w, `{"error":"failed to marshal response"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{
		"error": message,
	})
}
