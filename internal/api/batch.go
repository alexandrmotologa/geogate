package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/alexandrmotologa/geogate/internal/geoip"
)

// BatchLookupRequest specifies the list of target IPs to resolve.
type BatchLookupRequest struct {
	IPs []string `json:"ips"`
}

// BatchLookupResponse encapsulates batch results and performance telemetry.
type BatchLookupResponse struct {
	TotalProcessed       int                `json:"total_processed"`
	Results              []*geoip.GeoRecord `json:"results"`
	DurationMicroseconds int64              `json:"duration_microseconds"`
	ThroughputIPsPerSec  float64            `json:"throughput_ips_per_sec"`
}

// HandleBatchLookup processes high-throughput parallel IP resolutions.
func (s *Server) HandleBatchLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed. Use POST.")
		return
	}

	var req BatchLookupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	if len(req.IPs) == 0 {
		s.respondError(w, http.StatusBadRequest, "IPs array must not be empty")
		return
	}

	if len(req.IPs) > s.maxBatchSize {
		s.respondError(w, http.StatusBadRequest, "Batch size exceeds maximum limit")
		return
	}

	start := time.Now()
	results := make([]*geoip.GeoRecord, len(req.IPs))

	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}
	if len(req.IPs) < numWorkers {
		numWorkers = len(req.IPs)
	}

	chunkSize := (len(req.IPs) + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

	for wIdx := 0; wIdx < numWorkers; wIdx++ {
		startIdx := wIdx * chunkSize
		endIdx := startIdx + chunkSize
		if startIdx >= len(req.IPs) {
			break
		}
		if endIdx > len(req.IPs) {
			endIdx = len(req.IPs)
		}

		wg.Add(1)
		go func(from, to int) {
			defer wg.Done()
			for i := from; i < to; i++ {
				rec, err := s.engine.LookupString(req.IPs[i])
				if err != nil {
					results[i] = &geoip.GeoRecord{
						IP:          req.IPs[i],
						CountryCode: "XX",
						CountryName: "Unresolvable",
					}
				} else {
					results[i] = rec
				}
			}
		}(startIdx, endIdx)
	}

	wg.Wait()
	duration := time.Since(start)
	durationUs := duration.Microseconds()
	if durationUs <= 0 {
		durationUs = 1
	}

	sec := duration.Seconds()
	var throughput float64
	if sec > 0 {
		throughput = float64(len(req.IPs)) / sec
	} else {
		throughput = float64(len(req.IPs)) * 1000000.0
	}

	resp := BatchLookupResponse{
		TotalProcessed:       len(results),
		Results:              results,
		DurationMicroseconds: durationUs,
		ThroughputIPsPerSec:  throughput,
	}

	s.respondJSON(w, http.StatusOK, resp)
}
