# Integration Guide

This guide describes how to consume GeoGate Edge headers in your application code.

## Injected Edge Headers

GeoGate adds the following headers to each proxied request:

| Header Name | Type | Example | Description |
| :--- | :--- | :--- | :--- |
| `CF-IPCountry` | string | `JP` | ISO 3166-1 alpha-2 country code |
| `X-Country-Code` | string | `JP` | ISO 3166-1 alpha-2 country code |
| `X-Country-Name` | string | `Japan` | English country name |
| `X-Geo-City` | string | `Tokyo` | City name |
| `X-Geo-Latitude` | string | `35.6762` | Approximate latitude |
| `X-Geo-Longitude` | string | `139.6503` | Approximate longitude |
| `X-Geo-Timezone` | string | `Asia/Tokyo` | IANA timezone identifier |
| `X-Geo-EU` | string | `0` or `1` | 1 if country is an EU/EEA member state, 0 otherwise |
| `X-ASN` | string | `AS9370` | Autonomous System Number |
| `X-ASN-Org` | string | `Sakura Internet Inc.` | Autonomous System Organization name |
| `X-Geo-Currency` | string | `JPY` | ISO 4217 currency code |
| `X-Geo-PPP-Factor` | string | `0.75` | Purchasing Power Parity factor relative to USD |
| `X-Geo-PPP-Discount` | string | `25` | Recommended localized pricing discount percent |
| `X-Geo-Connection-Type` | string | `residential` | Detected or simulated type: residential, datacenter, vpn, tor |
| `X-Geo-Is-Datacenter` | string | `0` or `1` | 1 if IP belongs to a cloud hosting or datacenter provider |
| `X-Geo-Is-VPN` | string | `0` or `1` | 1 if IP is associated with known commercial VPNs |
| `X-Geo-Is-Tor` | string | `0` or `1` | 1 if request simulates or originates from a Tor exit |
| `X-GeoGate-Injected-Latency` | string | `120ms` | Duration slept before forwarding when latency simulation is active |
| `X-Real-IP` | string | `133.242.1.1` | Client or simulated IP |
| `X-GeoGate-Simulated` | string | `true` or `false` | True if request was triggered by a simulation rule |

---

## Next.js (App Router)

Create `middleware.ts` in your Next.js project root:

```typescript
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

export function middleware(request: NextRequest) {
  const country = request.headers.get('cf-ipcountry') || 'US';
  const city = request.headers.get('x-geo-city') || 'Unknown';
  const isEU = request.headers.get('x-geo-eu') === '1';
  const currency = request.headers.get('x-geo-currency') || 'USD';

  // Example: redirect EU visitors to localized terms or display VAT banner
  const response = NextResponse.next();
  response.headers.set('x-applied-currency', currency);
  return response;
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
};
```

---

## Express.js (Node.js)

Add a middleware function at the top of your Express application:

```javascript
const express = require('express');
const app = express();

app.use((req, res, next) => {
  req.geo = {
    country: req.headers['cf-ipcountry'] || 'US',
    countryName: req.headers['x-country-name'],
    city: req.headers['x-geo-city'],
    latitude: parseFloat(req.headers['x-geo-latitude'] || '0'),
    longitude: parseFloat(req.headers['x-geo-longitude'] || '0'),
    isEU: req.headers['x-geo-eu'] === '1',
    asn: req.headers['x-asn'],
    asnOrg: req.headers['x-asn-org'],
    currency: req.headers['x-geo-currency'] || 'USD',
    isSimulated: req.headers['x-geogate-simulated'] === 'true',
  };
  next();
});

app.get('/pricing', (req, res) => {
  res.json({
    user_location: req.geo.country,
    currency: req.geo.currency,
    vat_applicable: req.geo.isEU,
  });
});

app.listen(3000);
```

---

## Python (FastAPI)

```python
from fastapi import FastAPI, Header
from typing import Optional

app = FastAPI()

@app.get("/pricing")
def get_pricing(
    country: str = Header("US", alias="cf-ipcountry"),
    city: Optional[str] = Header(None, alias="x-geo-city"),
    is_eu: str = Header("0", alias="x-geo-eu"),
    currency: str = Header("USD", alias="x-geo-currency"),
):
    return {
        "detected_country": country,
        "city": city,
        "currency": currency,
        "requires_gdpr_consent": is_eu == "1",
    }
```

---

## Go (net/http or Chi)

```go
package main

import (
	"encoding/json"
	"net/http"
)

type UserGeo struct {
	Country  string `json:"country"`
	City     string `json:"city"`
	IsEU     bool   `json:"is_eu"`
	Currency string `json:"currency"`
}

func pricingHandler(w http.ResponseWriter, r *http.Request) {
	geo := UserGeo{
		Country:  r.Header.Get("CF-IPCountry"),
		City:     r.Header.Get("X-Geo-City"),
		IsEU:     r.Header.Get("X-Geo-EU") == "1",
		Currency: r.Header.Get("X-Geo-Currency"),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(geo)
}

func main() {
	http.HandleFunc("/pricing", pricingHandler)
	_ = http.ListenAndServe(":3000", nil)
}
```

---

## Simulation Overrides

You can simulate any geographic location by passing parameters in your requests:

1. Query Parameter:
   ```bash
   curl "http://localhost:8080/pricing?simulate_country=JP"
   curl "http://localhost:8080/pricing?simulate_city=Berlin"
   curl "http://localhost:8080/pricing?simulate_country=RO&simulate_conn=tor"
   curl "http://localhost:8080/pricing?simulate_latency=auto"
   ```

2. Request Header:
   ```bash
   curl -H "X-Simulate-Country: RO" http://localhost:8080/checkout
   curl -H "X-Simulate-City: Tokyo" http://localhost:8080/checkout
   curl -H "X-Simulate-Conn: datacenter" http://localhost:8080/api/orders
   curl -H "X-Simulate-Latency: 150ms" http://localhost:8080/api/orders
   ```

3. Browser Session Cookie:
   In your browser DevTools console:
   ```javascript
   document.cookie = "geogate_simulate_country=JP; path=/";
   ```
   Or click the "Simulate in Browser Session" button inside the GeoGate Studio UI.
