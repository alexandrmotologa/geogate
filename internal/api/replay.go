package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/geogate/internal/geoip"
	"github.com/alexandrmotologa/geogate/internal/proxy"
)

// ReplayRequest encapsulates instructions to resend a captured request under a new simulated persona.
type ReplayRequest struct {
	EventID         uint64 `json:"event_id,omitempty"`
	Method          string `json:"method"`
	Path            string `json:"path"`
	SimulateCountry string `json:"simulate_country"`
	SimulateCity    string `json:"simulate_city,omitempty"`
	SimulateConn    string `json:"simulate_conn,omitempty"`
	SimulateLatency string `json:"simulate_latency,omitempty"`
}

// ReplayResponse contains upstream execution diagnostics.
type ReplayResponse struct {
	Success          bool              `json:"success"`
	StatusCode       int               `json:"status_code"`
	DurationMs       float64           `json:"duration_ms"`
	SimulatedCountry string            `json:"simulated_country"`
	InjectedHeaders  map[string]string `json:"injected_headers"`
	UpstreamResponse string            `json:"upstream_response,omitempty"`
	Error            string            `json:"error,omitempty"`
}

// HandleReplayRequest re-executes a proxied transaction towards upstream with new geolocation parameters.
func (s *Server) HandleReplayRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed. Use POST.")
		return
	}

	var req ReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		path = "/"
	}

	// If EventID provided, pull historical metadata if missing
	if req.EventID > 0 && s.hub != nil {
		if evt := s.hub.GetEventByID(req.EventID); evt != nil {
			if req.Method == "" {
				method = evt.Method
			}
			if req.Path == "" {
				path = evt.Path
			}
		}
	}

	targetCountry := req.SimulateCountry
	if targetCountry == "" {
		targetCountry = s.cfg.DefaultCountry
	}

	var geoRec *geoip.GeoRecord
	var err error

	if req.SimulateCity != "" {
		geoRec, err = s.engine.LookupCity(req.SimulateCity)
	} else {
		geoRec, err = s.engine.LookupCountry(targetCountry)
	}

	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Failed to resolve simulation target: "+err.Error())
		return
	}

	if req.SimulateConn != "" {
		geoRec.ConnectionType = req.SimulateConn
		if req.SimulateConn == "datacenter" {
			geoRec.IsDatacenter = true
		} else if req.SimulateConn == "vpn" {
			geoRec.IsVPN = true
		} else if req.SimulateConn == "tor" {
			geoRec.IsTor = true
		}
	}

	targetURL := strings.TrimRight(s.cfg.UpstreamURL, "/") + path

	httpReq, err := http.NewRequest(method, targetURL, nil)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Failed to build upstream request: "+err.Error())
		return
	}

	injected := proxy.SynthesizeHeaders(geoRec)
	for k, v := range injected {
		httpReq.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	start := time.Now()
	resp, err := client.Do(httpReq)
	duration := float64(time.Since(start).Microseconds()) / 1000.0

	if err != nil {
		s.respondJSON(w, http.StatusOK, ReplayResponse{
			Success:          false,
			StatusCode:       http.StatusBadGateway,
			DurationMs:       duration,
			SimulatedCountry: geoRec.CountryCode,
			InjectedHeaders:  injected,
			Error:            err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	s.respondJSON(w, http.StatusOK, ReplayResponse{
		Success:          true,
		StatusCode:       resp.StatusCode,
		DurationMs:       duration,
		SimulatedCountry: geoRec.CountryCode,
		InjectedHeaders:  injected,
		UpstreamResponse: string(bodyBytes),
	})
}

// HandleGetPPP returns Purchasing Power Parity factor and discount for an ISO country.
func (s *Server) HandleGetPPP(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/v1/ppp/")
	code = strings.ToUpper(strings.TrimSpace(code))

	info, ok := geoip.CountryDatabase[code]
	if !ok {
		s.respondError(w, http.StatusNotFound, "Country code not found: "+code)
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"country_code":         info.Code,
		"country_name":         info.Name,
		"currency":             info.Currency,
		"ppp_factor":           info.PPPFactor,
		"recommended_discount": info.PPPDiscount,
		"is_eu":                info.IsEU,
	})
}
