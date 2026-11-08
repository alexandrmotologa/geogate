package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/alexandrmotologa/geogate/internal/geoip"
)

// GeofencePolicy defines access control rules based on geography and network origins.
type GeofencePolicy struct {
	Name              string   `json:"name"`
	AllowedCountries  []string `json:"allowed_countries,omitempty"`
	BlockedCountries  []string `json:"blocked_countries,omitempty"`
	RequireEU         bool     `json:"require_eu,omitempty"`
	AllowedContinents []string `json:"allowed_continents,omitempty"`
	AllowedASNs       []uint   `json:"allowed_asns,omitempty"`
	BlockedASNs       []uint   `json:"blocked_asns,omitempty"`
}

// GeofenceEvaluationRequest contains the policy and the candidate IPs to evaluate.
type GeofenceEvaluationRequest struct {
	Policy GeofencePolicy `json:"policy"`
	IPs    []string       `json:"ips"`
}

// GeofenceEvaluationItem represents the evaluation verdict for an individual IP.
type GeofenceEvaluationItem struct {
	IP          string           `json:"ip"`
	Allowed     bool             `json:"allowed"`
	Reason      string           `json:"reason"`
	MatchedRule string           `json:"matched_rule"`
	Record      *geoip.GeoRecord `json:"record"`
}

// GeofenceEvaluationResponse encapsulates all verdicts.
type GeofenceEvaluationResponse struct {
	PolicyName string                   `json:"policy_name"`
	Total      int                      `json:"total"`
	Allowed    int                      `json:"allowed_count"`
	Blocked    int                      `json:"blocked_count"`
	Results    []GeofenceEvaluationItem `json:"results"`
}

// HandleGeofenceEvaluate evaluates a set of IPs against an access policy matrix.
func (s *Server) HandleGeofenceEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed. Use POST.")
		return
	}

	var req GeofenceEvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	if len(req.IPs) == 0 {
		s.respondError(w, http.StatusBadRequest, "IPs array must not be empty")
		return
	}

	// Prepare quick-lookup sets
	blockedCountries := make(map[string]bool)
	for _, c := range req.Policy.BlockedCountries {
		blockedCountries[strings.ToUpper(strings.TrimSpace(c))] = true
	}

	allowedCountries := make(map[string]bool)
	for _, c := range req.Policy.AllowedCountries {
		allowedCountries[strings.ToUpper(strings.TrimSpace(c))] = true
	}

	allowedContinents := make(map[string]bool)
	for _, cont := range req.Policy.AllowedContinents {
		allowedContinents[strings.ToUpper(strings.TrimSpace(cont))] = true
	}

	blockedASNs := make(map[uint]bool)
	for _, a := range req.Policy.BlockedASNs {
		blockedASNs[a] = true
	}

	allowedASNs := make(map[uint]bool)
	for _, a := range req.Policy.AllowedASNs {
		allowedASNs[a] = true
	}

	results := make([]GeofenceEvaluationItem, 0, len(req.IPs))
	allowedCount := 0
	blockedCount := 0

	for _, ipStr := range req.IPs {
		rec, err := s.engine.LookupString(ipStr)
		if err != nil {
			results = append(results, GeofenceEvaluationItem{
				IP:          ipStr,
				Allowed:     false,
				Reason:      "IP resolution failed: " + err.Error(),
				MatchedRule: "ResolutionFailure",
				Record:      nil,
			})
			blockedCount++
			continue
		}

		cCode := strings.ToUpper(rec.CountryCode)
		contCode := strings.ToUpper(rec.ContinentCode)

		// 1. Check Explicit Blocked ASN
		if blockedASNs[rec.ASN.Number] {
			results = append(results, GeofenceEvaluationItem{
				IP:          ipStr,
				Allowed:     false,
				Reason:      fmt.Sprintf("ASN AS%d (%s) is in the blocked ASN list", rec.ASN.Number, rec.ASN.Organization),
				MatchedRule: "BlockedASN",
				Record:      rec,
			})
			blockedCount++
			continue
		}

		// 2. Check Explicit Blocked Country
		if blockedCountries[cCode] {
			results = append(results, GeofenceEvaluationItem{
				IP:          ipStr,
				Allowed:     false,
				Reason:      fmt.Sprintf("Country %s (%s) is in the blocked country list", cCode, rec.CountryName),
				MatchedRule: "BlockedCountry",
				Record:      rec,
			})
			blockedCount++
			continue
		}

		// 3. Check EU Requirement
		if req.Policy.RequireEU && !rec.IsEU {
			results = append(results, GeofenceEvaluationItem{
				IP:          ipStr,
				Allowed:     false,
				Reason:      fmt.Sprintf("Country %s is not an EU member state", cCode),
				MatchedRule: "RequireEU",
				Record:      rec,
			})
			blockedCount++
			continue
		}

		// 4. Check Allowed Continents Filter
		if len(allowedContinents) > 0 && !allowedContinents[contCode] {
			results = append(results, GeofenceEvaluationItem{
				IP:          ipStr,
				Allowed:     false,
				Reason:      fmt.Sprintf("Continent %s is not in allowed continents list", contCode),
				MatchedRule: "AllowedContinents",
				Record:      rec,
			})
			blockedCount++
			continue
		}

		// 5. Check Allowed Countries Filter
		if len(allowedCountries) > 0 && !allowedCountries[cCode] {
			results = append(results, GeofenceEvaluationItem{
				IP:          ipStr,
				Allowed:     false,
				Reason:      fmt.Sprintf("Country %s is not in the allowed country list", cCode),
				MatchedRule: "AllowedCountries",
				Record:      rec,
			})
			blockedCount++
			continue
		}

		// 6. Check Allowed ASNs Filter
		if len(allowedASNs) > 0 && !allowedASNs[rec.ASN.Number] {
			results = append(results, GeofenceEvaluationItem{
				IP:          ipStr,
				Allowed:     false,
				Reason:      fmt.Sprintf("ASN AS%d is not in the allowed ASN list", rec.ASN.Number),
				MatchedRule: "AllowedASNs",
				Record:      rec,
			})
			blockedCount++
			continue
		}

		// Passed all checks
		results = append(results, GeofenceEvaluationItem{
			IP:          ipStr,
			Allowed:     true,
			Reason:      "Passed all geofence policy rules",
			MatchedRule: "Pass",
			Record:      rec,
		})
		allowedCount++
	}

	policyName := req.Policy.Name
	if policyName == "" {
		policyName = "Custom Geofence Policy"
	}

	resp := GeofenceEvaluationResponse{
		PolicyName: policyName,
		Total:      len(results),
		Allowed:    allowedCount,
		Blocked:    blockedCount,
		Results:    results,
	}

	s.respondJSON(w, http.StatusOK, resp)
}
