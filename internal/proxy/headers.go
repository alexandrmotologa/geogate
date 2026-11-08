package proxy

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/alexandrmotologa/geogate/internal/geoip"
)

// HeaderMap holds the injected key-value pairs for debugging and inspection.
type HeaderMap map[string]string

// SynthesizeHeaders generates a map of standard CDN edge headers for a GeoRecord.
func SynthesizeHeaders(rec *geoip.GeoRecord) HeaderMap {
	headers := HeaderMap{
		"CF-IPCountry":     rec.CountryCode,
		"X-Country-Code":   rec.CountryCode,
		"X-Country-Name":   rec.CountryName,
		"X-Geo-City":       rec.City,
		"X-Geo-Latitude":   fmt.Sprintf("%.4f", rec.Latitude),
		"X-Geo-Longitude":  fmt.Sprintf("%.4f", rec.Longitude),
		"X-Geo-Timezone":   rec.Timezone,
		"X-Geo-EU":         strconv.Itoa(btoi(rec.IsEU)),
		"X-ASN":            fmt.Sprintf("AS%d", rec.ASN.Number),
		"X-ASN-Org":        rec.ASN.Organization,
		"X-Geo-Currency":   rec.Currency,
		"X-Real-IP":        rec.IP,
		"X-Forwarded-For":  rec.IP,
	}

	if rec.Simulated {
		headers["X-GeoGate-Simulated"] = "true"
	} else {
		headers["X-GeoGate-Simulated"] = "false"
	}

	return headers
}

// InjectHeaders writes standard CDN edge headers onto an outgoing HTTP request.
func InjectHeaders(req *http.Request, rec *geoip.GeoRecord) {
	headers := SynthesizeHeaders(rec)
	for key, val := range headers {
		req.Header.Set(key, val)
	}
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
