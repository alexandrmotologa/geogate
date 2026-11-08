package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/alexandrmotologa/geogate/internal/geoip"
	"github.com/alexandrmotologa/geogate/internal/proxy"
)

// EngineStatus describes the runtime telemetry and memory consumption of the GeoGate daemon.
type EngineStatus struct {
	Status         string  `json:"status"`
	Version        string  `json:"version"`
	UptimeSeconds  int64   `json:"uptime_seconds"`
	MemoryAllocMB  float64 `json:"memory_alloc_mb"`
	MemorySysMB    float64 `json:"memory_sys_mb"`
	NumGoroutine   int     `json:"num_goroutine"`
	TotalLookups   uint64  `json:"total_lookups"`
	HasExternalDB  bool    `json:"has_external_db"`
	DefaultCountry string  `json:"default_country"`
	UpstreamURL    string  `json:"upstream_url"`
}

// HandleGetCountries returns the catalog of supported countries.
func (s *Server) HandleGetCountries(w http.ResponseWriter, r *http.Request) {
	countries := s.engine.Synthesizer().ListCountries()
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"total":     len(countries),
		"countries": countries,
	})
}

// HandleGetCities returns the catalog of major global cities.
func (s *Server) HandleGetCities(w http.ResponseWriter, r *http.Request) {
	cities := s.engine.Synthesizer().ListCities()
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"total":  len(cities),
		"cities": cities,
	})
}

// HandleGetStatus returns memory metrics, uptime, and database status.
func (s *Server) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(s.startTime).Seconds()

	status := EngineStatus{
		Status:         "healthy",
		Version:        "1.0.0",
		UptimeSeconds:  int64(uptime),
		MemoryAllocMB:  float64(m.Alloc) / 1024.0 / 1024.0,
		MemorySysMB:    float64(m.Sys) / 1024.0 / 1024.0,
		NumGoroutine:   runtime.NumGoroutine(),
		TotalLookups:   s.engine.TotalLookups(),
		HasExternalDB:  s.engine.HasExternalDB(),
		DefaultCountry: s.cfg.DefaultCountry,
		UpstreamURL:    s.cfg.UpstreamURL,
	}

	s.respondJSON(w, http.StatusOK, status)
}

// HandleSimulateHeaders computes edge headers for a target country or city.
func (s *Server) HandleSimulateHeaders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	country := q.Get("country")
	city := q.Get("city")
	ip := q.Get("ip")

	var rec *geoip.GeoRecord
	var err error

	if country != "" {
		rec, err = s.engine.LookupCountry(country)
	} else if city != "" {
		rec, err = s.engine.LookupCity(city)
	} else if ip != "" {
		rec, err = s.engine.LookupString(ip)
		if err == nil {
			rec.Simulated = true
		}
	} else {
		rec, err = s.engine.LookupCountry(s.cfg.DefaultCountry)
	}

	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	headers := proxy.SynthesizeHeaders(rec)

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"record":  rec,
		"headers": headers,
	})
}
