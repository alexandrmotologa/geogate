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
	var rec *geoip.GeoRecord
	var err error

	// 1. Check Query Parameter Overrides
	query := req.URL.Query()
	if c := query.Get("simulate_country"); c != "" {
		rec, err = r.engine.LookupCountry(c)
	} else if c := query.Get("country"); c != "" && len(c) == 2 {
		rec, err = r.engine.LookupCountry(c)
	} else if city := query.Get("simulate_city"); city != "" {
		rec, err = r.engine.LookupCity(city)
	} else if ip := query.Get("simulate_ip"); ip != "" {
		rec, err = r.engine.LookupString(ip)
		if err == nil {
			rec.Simulated = true
		}
	} else if c := req.Header.Get("X-Simulate-Country"); c != "" {
		// 2. Check Header Overrides
		rec, err = r.engine.LookupCountry(c)
	} else if city := req.Header.Get("X-Simulate-City"); city != "" {
		rec, err = r.engine.LookupCity(city)
	} else if ip := req.Header.Get("X-Simulate-IP"); ip != "" {
		rec, err = r.engine.LookupString(ip)
		if err == nil {
			rec.Simulated = true
		}
	} else if cookie, cErr := req.Cookie("geogate_simulate_country"); cErr == nil && cookie.Value != "" {
		// 3. Check Cookie Overrides
		rec, err = r.engine.LookupCountry(cookie.Value)
	} else if cookie, cErr := req.Cookie("geogate_country"); cErr == nil && cookie.Value != "" {
		rec, err = r.engine.LookupCountry(cookie.Value)
	} else if cookie, cErr := req.Cookie("geogate_simulate_city"); cErr == nil && cookie.Value != "" {
		rec, err = r.engine.LookupCity(cookie.Value)
	} else if cookie, cErr := req.Cookie("geogate_simulate_ip"); cErr == nil && cookie.Value != "" {
		rec, err = r.engine.LookupString(cookie.Value)
		if err == nil {
			rec.Simulated = true
		}
	}

	if err != nil {
		return nil, err
	}

	if rec != nil {
		r.applySecurityOverrides(req, rec)
		return rec, nil
	}

	// 4. Resolve Actual Client IP
	clientIP := ExtractClientIP(req)
	parsedIP := net.ParseIP(clientIP)

	if parsedIP == nil || parsedIP.IsLoopback() || parsedIP.IsPrivate() || parsedIP.IsUnspecified() {
		// Use default country simulation for local development loopback
		rrec, err := r.engine.LookupCountry(r.defaultCountry)
		if err == nil {
			rrec.IP = clientIP
			rrec.Simulated = true
			rec = rrec
		}
	}

	if rec == nil {
		resolved, err := r.engine.LookupString(clientIP)
		if err != nil {
			return nil, err
		}
		rec = resolved
	}

	// Apply connection type and security overrides if specified
	r.applySecurityOverrides(req, rec)

	return rec, nil
}

// applySecurityOverrides processes simulate_conn or simulate_tor flags.
func (r *Resolver) applySecurityOverrides(req *http.Request, rec *geoip.GeoRecord) {
	connType := req.URL.Query().Get("simulate_conn")
	if connType == "" {
		connType = req.Header.Get("X-Simulate-Connection-Type")
	}
	if connType == "" {
		connType = req.Header.Get("X-Simulate-Connection")
	}

	if connType != "" {
		connType = strings.ToLower(strings.TrimSpace(connType))
		rec.ConnectionType = connType
		rec.Simulated = true
		switch connType {
		case "datacenter", "hosting":
			rec.IsDatacenter = true
			rec.IsVPN = false
			rec.IsTor = false
		case "vpn":
			rec.IsVPN = true
			rec.IsDatacenter = false
			rec.IsTor = false
		case "tor":
			rec.IsTor = true
			rec.IsVPN = false
			rec.IsDatacenter = false
		case "residential":
			rec.IsDatacenter = false
			rec.IsVPN = false
			rec.IsTor = false
		}
	}
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
