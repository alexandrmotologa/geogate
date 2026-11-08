package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/alexandrmotologa/geogate/internal/geoip"
)

// ProxyHandler coordinates reverse proxying, edge header injection, and live telemetry.
type ProxyHandler struct {
	upstreamURL    *url.URL
	reverseProxy   *httputil.ReverseProxy
	resolver       *Resolver
	hub            *EventHub
	internalRouter http.Handler
}

// statusRecorder captures the response status code for logging.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// NewProxyHandler creates a configured reverse proxy handler.
func NewProxyHandler(upstream string, resolver *Resolver, hub *EventHub, internalRouter http.Handler) (*ProxyHandler, error) {
	var targetURL *url.URL
	var revProxy *httputil.ReverseProxy

	if upstream != "" {
		parsed, err := url.Parse(upstream)
		if err != nil {
			return nil, fmt.Errorf("invalid upstream URL: %w", err)
		}
		targetURL = parsed
		revProxy = httputil.NewSingleHostReverseProxy(parsed)

		// Custom error handler for offline upstreams
		revProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, `{"error":"Upstream application unreachable","upstream":"%s","message":"%s","geogate_status":"active"}`, parsed.String(), err.Error())
		}
	}

	return &ProxyHandler{
		upstreamURL:    targetURL,
		reverseProxy:   revProxy,
		resolver:       resolver,
		hub:            hub,
		internalRouter: internalRouter,
	}, nil
}

// ServeHTTP handles requests, routing internal API / UI requests or forwarding to upstream.
func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Route internal GeoGate endpoints
	if p.isInternalRoute(req.URL.Path) {
		if p.internalRouter != nil {
			p.internalRouter.ServeHTTP(w, req)
			return
		}
	}

	// If no upstream configured or path is root and upstream is unset, route to studio UI
	if p.reverseProxy == nil {
		if p.internalRouter != nil {
			p.internalRouter.ServeHTTP(w, req)
			return
		}
		http.Error(w, "GeoGate active - no upstream configured", http.StatusOK)
		return
	}

	start := time.Now()

	// Resolve geographic intelligence for this request
	geoRec, err := p.resolver.Resolve(req)
	if err != nil {
		// Fallback to basic record if resolution fails
		geoRec = &geoip.GeoRecord{
			IP:          ExtractClientIP(req),
			CountryCode: "US",
			CountryName: "United States",
			City:        "Unknown",
		}
	}

	// Inject standard edge CDN headers
	InjectHeaders(req, geoRec)

	// Prepare tracking recorder
	recorder := &statusRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// Forward request to upstream application
	p.reverseProxy.ServeHTTP(recorder, req)

	duration := float64(time.Since(start).Microseconds()) / 1000.0

	// Emit live event to event hub
	if p.hub != nil {
		p.hub.Publish(&RequestEvent{
			Method:      req.Method,
			Path:        req.URL.Path,
			ClientIP:    geoRec.IP,
			Simulated:   geoRec.Simulated,
			CountryCode: geoRec.CountryCode,
			CountryName: geoRec.CountryName,
			City:        geoRec.City,
			StatusCode:  recorder.statusCode,
			DurationMs:  duration,
			Headers:     SynthesizeHeaders(geoRec),
		})
	}
}

// isInternalRoute identifies endpoints reserved for GeoGate API, metrics, and cockpit studio.
func (p *ProxyHandler) isInternalRoute(path string) bool {
	if strings.HasPrefix(path, "/v1/") ||
		strings.HasPrefix(path, "/geogate/") ||
		path == "/metrics" ||
		path == "/health" ||
		path == "/favicon.ico" {
		return true
	}
	return false
}
