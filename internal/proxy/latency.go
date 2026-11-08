package proxy

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexandrmotologa/geogate/internal/geoip"
)

// ReferenceLocation serves as the origin anchor for computing transatlantic/transpacific latency.
type ReferenceLocation struct {
	Latitude  float64
	Longitude float64
}

// DefaultServerOrigin defaults to standard US East/West cloud gateway (approx Kansas / central US).
var DefaultServerOrigin = ReferenceLocation{
	Latitude:  39.8283,
	Longitude: -98.5795,
}

// ComputeDistanceKm computes the spherical surface distance between two coordinates in kilometers using Haversine.
func ComputeDistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	lat1Rad := lat1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// EstimateRealisticLatency calculates expected round-trip time (RTT) based on geographic fiber distances.
func EstimateRealisticLatency(targetLat, targetLon float64) time.Duration {
	distKm := ComputeDistanceKm(DefaultServerOrigin.Latitude, DefaultServerOrigin.Longitude, targetLat, targetLon)

	// Fiber optic propagation ~ 5 microseconds per km RTT + router processing hops
	// Minimum domestic base 15ms, max global ~280ms
	ms := 15.0 + (distKm / 100.0) * 1.8
	if ms > 350.0 {
		ms = 350.0
	}
	return time.Duration(ms) * time.Millisecond
}

// ResolveInjectedLatency determines if artificial network delay should be introduced.
func ResolveInjectedLatency(req *http.Request, rec *geoip.GeoRecord) time.Duration {
	val := req.URL.Query().Get("simulate_latency")
	if val == "" {
		val = req.Header.Get("X-Simulate-Latency")
	}

	if val == "" {
		return 0
	}

	val = strings.ToLower(strings.TrimSpace(val))

	if val == "auto" || val == "true" || val == "1" {
		if rec != nil && (rec.Latitude != 0 || rec.Longitude != 0) {
			return EstimateRealisticLatency(rec.Latitude, rec.Longitude)
		}
		return 120 * time.Millisecond
	}

	// Try parsing duration string like "150ms" or raw integer "150"
	if d, err := time.ParseDuration(val); err == nil {
		return d
	}
	if ms, err := strconv.Atoi(val); err == nil && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}

	return 0
}
