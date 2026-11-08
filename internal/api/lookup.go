package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/geogate/internal/geoip"
	"github.com/alexandrmotologa/geogate/internal/proxy"
)

// LookupResponse wraps the resolved GeoRecord with timing diagnostics.
type LookupResponse struct {
	Record               *geoip.GeoRecord `json:"record"`
	DurationMicroseconds int64            `json:"duration_microseconds"`
}

// HandleSingleLookup handles GET /v1/lookup/:ip
func (s *Server) HandleSingleLookup(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Extract IP from URL path: /v1/lookup/<ip>
	path := strings.TrimPrefix(r.URL.Path, "/v1/lookup/")
	path = strings.TrimSpace(path)

	if path == "" || path == "me" {
		path = proxy.ExtractClientIP(r)
	}

	rec, err := s.engine.LookupString(path)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	durationUs := time.Since(start).Microseconds()

	resp := LookupResponse{
		Record:               rec,
		DurationMicroseconds: durationUs,
	}

	s.respondJSON(w, http.StatusOK, resp)
}
