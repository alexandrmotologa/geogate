package proxy

import (
	"net"
	"net/http"
	"strings"

	"github.com/alexandrmotologa/geogate/internal/geoip"
)

// Resolver resolves geographic records from HTTP requests considering simulation overrides.
type Resolver struct {
	engine         *geoip.Engine
	defaultCountry string
}

// NewResolver initializes a simulation override resolver.
func NewResolver(engine *geoip.Engine, defaultCountry string) *Resolver {
	if defaultCountry == "" {
		defaultCountry = "US"
	}
	return &Resolver{
		engine:         engine,
		defaultCountry: strings.ToUpper(defaultCountry),
	}
}

// Resolve extracts client identity or simulation overrides and returns the active GeoRecord.
func (r *Resolver) Resolve(req *http.Request) (*geoip.GeoRecord, error) {
	// 1. Check Query Parameter Overrides
	query := req.URL.Query()
	if c := query.Get("simulate_country"); c != "" {
		return r.engine.LookupCountry(c)
	}
	if c := query.Get("country"); c != "" && len(c) == 2 {
		return r.engine.LookupCountry(c)
	}
	if city := query.Get("simulate_city"); city != "" {
		return r.engine.LookupCity(city)
	}
	if ip := query.Get("simulate_ip"); ip != "" {
		rec, err := r.engine.LookupString(ip)
		if err == nil {
			rec.Simulated = true
		}
		return rec, err
	}

	// 2. Check Header Overrides
	if c := req.Header.Get("X-Simulate-Country"); c != "" {
		return r.engine.LookupCountry(c)
	}
	if city := req.Header.Get("X-Simulate-City"); city != "" {
		return r.engine.LookupCity(city)
	}
	if ip := req.Header.Get("X-Simulate-IP"); ip != "" {
		rec, err := r.engine.LookupString(ip)
		if err == nil {
			rec.Simulated = true
		}
		return rec, err
	}

	// 3. Check Cookie Overrides
	if cookie, err := req.Cookie("geogate_simulate_country"); err == nil && cookie.Value != "" {
		return r.engine.LookupCountry(cookie.Value)
	}
	if cookie, err := req.Cookie("geogate_country"); err == nil && cookie.Value != "" {
		return r.engine.LookupCountry(cookie.Value)
	}
	if cookie, err := req.Cookie("geogate_simulate_city"); err == nil && cookie.Value != "" {
		return r.engine.LookupCity(cookie.Value)
	}
	if cookie, err := req.Cookie("geogate_simulate_ip"); err == nil && cookie.Value != "" {
		rec, err := r.engine.LookupString(cookie.Value)
		if err == nil {
			rec.Simulated = true
		}
		return rec, err
	}

	// 4. Resolve Actual Client IP
	clientIP := ExtractClientIP(req)
	parsedIP := net.ParseIP(clientIP)

	if parsedIP == nil || parsedIP.IsLoopback() || parsedIP.IsPrivate() || parsedIP.IsUnspecified() {
		// Use default country simulation for local development loopback
		rec, err := r.engine.LookupCountry(r.defaultCountry)
		if err == nil {
			rec.IP = clientIP
			rec.Simulated = true
			return rec, nil
		}
	}

	return r.engine.LookupString(clientIP)
}

// ExtractClientIP retrieves the real remote IP address from HTTP request headers or network connection.
func ExtractClientIP(req *http.Request) string {
	// Check standard proxy headers
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	if xReal := req.Header.Get("X-Real-IP"); xReal != "" {
		ip := strings.TrimSpace(xReal)
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	// Extract from RemoteAddr
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil && net.ParseIP(host) != nil {
		return host
	}

	return req.RemoteAddr
}
