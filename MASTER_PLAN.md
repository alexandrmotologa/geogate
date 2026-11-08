# `geogate` — Architecture & Engineering Master Plan

> **Self-Hosted Local IP Geolocation, ASN Intelligence & Edge Geofencing Gateway (< 35MB RAM)**  
> *IPinfo and MaxMind Cloud Alternative in a Single Binary (Go + React 19)*

---

## 1. Executive Summary & Problem Space

### The Core Problem
Modern web applications increasingly rely on geographic intelligence to deliver localized experiences:
- **Purchasing Power Parity (PPP) Pricing**: Displaying localized discount banners based on user country.
- **Tax & Compliance**: Applying European Union (EU) VAT, California CCPA, or GDPR cookie consent dialogs.
- **Security & Fraud Prevention**: Blocking malicious Autonomous System Numbers (ASNs), bulletproof hosting providers, or sanctioned regions.
- **Localized Routing**: Directing users to local currency checkout or native language versions.

However, developing and testing geolocation features is notoriously frustrating:
1. **Broken Local Development**: In local testing, client IP addresses are `127.0.0.1` or `::1`, which have no geographic metadata and cause geolocation middlewares to crash or fall back to incorrect defaults.
2. **Expensive Cloud SaaS Limits (IPinfo, MaxMind Precision, AbstractAPI)**: Cloud GeoIP APIs charge \$49–\$299+/month, enforce strict rate limits that break automated test suites, and introduce 50–150ms of network latency per lookup.

### The Solution: `geogate`
`geogate` is an ultra-fast, single-binary IP geolocation and ASN intelligence gateway written in Go with an embedded React 19 interactive world map studio. It loads MaxMind MMDB databases directly into memory, resolving IP lookups in under **8 microseconds** with zero external network requests.

`geogate` can run as a **Simulation Reverse Proxy** in front of your local application, automatically injecting standard edge CDN headers (`CF-IPCountry`, `X-Geo-City`, `X-Geo-Latitude`, `X-Geo-Longitude`, `X-ASN`, `X-Geo-EU`) into every request. Developers can simulate traffic from anywhere in the world by simply clicking on an interactive 3D globe or appending query overrides like `?simulate_country=JP`.

---

## 2. System Architecture & Simulation Lifecycle

```
[ Developer Browser / API Test Suite (e.g. GET /pricing?simulate_country=JP) ]
                               │
                               │  HTTP / HTTPS Request
                               ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        `geogate` Gateway Daemon                        │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Geofencing & Simulation Override Interceptor                     │  │
│  │ - Inspects `?simulate_country=...` or `X-Simulate-Country` header│  │
│  │ - If override present -> generates synthetic IP for target locale│  │
│  └──────────────────────────────────┬───────────────────────────────┘  │
│                                     │ Client IP (Real or Simulated)    │
│                                     ▼                                  │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ In-Memory MMDB Trie Resolver (< 8 microseconds lookup)           │  │
│  │ - Country, City, Coordinates, Timezone, Currency, EU status      │  │
│  │ - Autonomous System Number (ASN) and Organization                │  │
│  └──────────────────────────────────┬───────────────────────────────┘  │
│                                     │                                  │
│                                     ▼                                  │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Edge Header Synthesizer (Standard CDN Headers Injected)          │  │
│  │   CF-IPCountry: JP                                               │  │
│  │   X-Geo-City: Tokyo                                              │  │
│  │   X-Geo-Latitude: 35.6895                                        │  │
│  │   X-Geo-Longitude: 139.6917                                      │  │
│  │   X-Geo-Timezone: Asia/Tokyo                                     │  │
│  │   X-Geo-EU: 0                                                    │  │
│  │   X-ASN: AS17676 (Softbank Corp)                                 │  │
│  └──────────────────────────────────┬───────────────────────────────┘  │
│                                     │ Proxied Request with Headers     │
│                                     ▼                                  │
│  [ Upstream Local App (Next.js, Express, Django, FastAPI on :3000) ]   │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Embedded React 19 Interactive World Map Studio                   │  │
│  │ - 3D/2D Interactive Globe: Click any city to test app response   │  │
│  │ - High-Throughput Batch API: POST /v1/batch/lookup (150k IPs/sec)│  │
│  │ - Geofencing Rule Simulator: Test country allow/block lists      │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Technology Stack

| Layer | Component | Specification |
| :--- | :--- | :--- |
| **Language & Runtime** | Go 1.23+ | Ultra-fast native execution, deterministic GC, minimal RAM footprint (< 35MB RAM) |
| **GeoIP Engine** | In-Memory MMDB Reader | Binary trie traversal (`oschwald/maxminddb-golang`) resolving lookups in < 8us |
| **Reverse Proxy** | Streaming Reverse Proxy | Transparently forwards HTTP requests while injecting Edge Geolocation headers |
| **Batch Pipeline** | Parallel Worker Pool | Ingests arrays of IPs via `POST /v1/batch/lookup` processing > 150,000 IPs/sec |
| **Embedded Web UI** | React 19 + TypeScript + Vite | Built as a single bundle and served via Go `//go:embed` |
| **UI Styling** | Tailwind CSS + Lucide Icons | High-contrast dark-mode global intelligence cockpit |

---

## 4. Repository & Directory Structure

```
B:\workgit\geogate\
├── cmd\
│   └── geogate\
│       └── main.go                 # Gateway daemon entrypoint, CLI commands (`serve`, `lookup`)
├── internal\
│   ├── config\
│   │   └── config.go               # Configuration (upstream app URL, MMDB path, proxy port)
│   ├── geoip\
│   │   ├── reader.go               # In-memory MMDB trie parser and cache manager
│   │   ├── models.go               # Country, City, Location, and ASN data structures
│   │   └── synthesizer.go          # Generates representative IPs for any ISO country code
│   ├── proxy\
│   │   ├── handler.go              # Reverse proxy pipeline injecting edge headers
│   │   ├── overrides.go            # Evaluates query params (`?simulate_country=...`) and headers
│   │   └── headers.go              # Cloudflare, Fastly, AWS CloudFront compatible header mapping
│   ├── api\
│   │   ├── lookup.go               # Single IP lookup endpoint (`GET /v1/lookup/:ip`)
│   │   ├── batch.go                # High-throughput batch endpoint (`POST /v1/batch/lookup`)
│   │   └── geofence.go             # Geofencing rule simulator (evaluates allow/block lists)
│   └── telemetry\
│       └── metrics.go              # Prometheus metrics: lookups/sec, latency histograms
├── web\
│   ├── src\
│   │   ├── components\
│   │   │   ├── WorldMap.tsx        # Interactive 3D/2D globe with click-to-simulate location
│   │   │   ├── HeaderInspector.tsx # Displays injected edge headers for currently simulated IP
│   │   │   ├── BatchTester.tsx     # Bulk IP lookup tester with CSV/JSON export
│   │   │   └── GeofenceMatrix.tsx  # Interactive country allow/block simulator
│   │   ├── App.tsx                 # Root geographic operations cockpit
│   │   ├── main.tsx                # React 19 entrypoint
│   │   └── index.css               # Tailwind CSS declarations
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── Makefile                        # Multi-platform compilation and benchmark targets
├── Dockerfile                      # Scratch container image (< 20MB)
├── README.md                       # Documentation, architecture, cURL examples
└── MASTER_PLAN.md                  # This master specification
```

---

## 5. Injected Edge Headers & Batch API Specification

### Injected Headers (Standard CDN Compatibility)

| Header Name | Example Value | Description |
| :--- | :--- | :--- |
| `CF-IPCountry` | `JP` | ISO 3166-1 alpha-2 country code (Cloudflare standard) |
| `X-Country-Code` | `JP` | ISO 3166-1 alpha-2 country code |
| `X-Country-Name` | `Japan` | Full English country name |
| `X-Geo-City` | `Tokyo` | City name |
| `X-Geo-Latitude` | `35.6895` | Approximate latitude |
| `X-Geo-Longitude` | `139.6917` | Approximate longitude |
| `X-Geo-Timezone` | `Asia/Tokyo` | IANA Timezone identifier |
| `X-Geo-EU` | `0` | `1` if country is an EU/EEA member state, `0` otherwise |
| `X-ASN` | `AS17676` | Autonomous System Number |
| `X-ASN-Org` | `SoftBank Corp.`| Autonomous System Organization name |

### High-Throughput Batch API (`POST /v1/batch/lookup`)

```json
{
  "ips": ["8.8.8.8", "1.1.1.1", "185.199.108.153"]
}
```

**Response (`200 OK`, sub-millisecond execution)**:
```json
{
  "results": [
    {
      "ip": "8.8.8.8",
      "country_code": "US",
      "country_name": "United States",
      "city": "Mountain View",
      "latitude": 37.4223,
      "longitude": -122.0848,
      "timezone": "America/Los_Angeles",
      "is_eu": false,
      "asn": {
        "number": 15169,
        "organization": "Google LLC"
      }
    }
  ],
  "duration_microseconds": 18
}
```

---

## 6. Implementation Roadmap & Task Breakdown

### Phase 1: In-Memory MMDB Trie Engine & Lookups
- [x] Initialized Git repository and created public GitHub repository (`alexandrmotologa/geogate`).
- [x] Coupled repository description with 14 targeted topics across four dimensions.
- [x] Completed portfolio Phase 1 registration in `mtlg-site` and `mtlglabs-space`.
- [x] Initialize Go 1.23+ module `github.com/alexandrmotologa/geogate`.
- [x] Implement data models (`internal/geoip/models.go`) covering Country, City, Coordinates, Timezone, EU status, and ASN.
- [x] Implement synthetic IP generator (`internal/geoip/synthesizer.go`) for all ISO 3166-1 alpha-2 countries and territories.
- [x] Implement in-memory MMDB trie reader (`internal/geoip/reader.go`) with embedded base dataset and optional external GeoLite2 `.mmdb` loading.
- [x] Write unit tests and micro-benchmarks asserting lookup latency under 8 microseconds (verified: 0.232 microseconds).

### Phase 2: Reverse Proxy & Edge Header Injection
- [x] Implement configuration manager (`internal/config/config.go`) parsing CLI flags and environment variables.
- [x] Implement override parser (`internal/proxy/overrides.go`) handling query parameters (`?simulate_country=...`), custom headers (`X-Simulate-Country`), and session cookies (`geogate_simulate_country`).
- [x] Implement edge header synthesizer (`internal/proxy/headers.go`) injecting standard CDN headers (`CF-IPCountry`, `X-Geo-City`, `X-Geo-Latitude`, `X-Geo-Longitude`, `X-Geo-Timezone`, `X-Geo-EU`, `X-ASN`, `X-ASN-Org`).
- [x] Implement streaming reverse proxy (`internal/proxy/handler.go`) forwarding requests to upstream applications.
- [x] Implement live traffic event hub (`internal/proxy/hub.go`) with Server-Sent Events (SSE) broadcasting proxied requests to the web UI.

### Phase 3: High-Throughput Batch API & Geofencing Engine
- [x] Implement single IP lookup endpoint (`GET /v1/lookup/:ip` and `GET /v1/lookup/me`).
- [x] Implement parallel batch lookup endpoint (`POST /v1/batch/lookup`) with goroutine worker pools capable of > 150,000 IPs/second (verified: > 500,000 IPs/sec).
- [x] Implement geofencing policy engine (`internal/api/geofence.go`) supporting country allow/block lists, ASN filters, and EU compliance boundaries.
- [x] Implement country and city catalog endpoints (`GET /v1/countries`, `GET /v1/cities`) for the UI map.
- [x] Implement Prometheus metrics exporter (`GET /metrics`) tracking throughput, latency histograms, and proxy metrics.

### Phase 4: Embedded React 19 Interactive World Map Studio
- [x] Scaffold Vite + React 19 + TypeScript application in `web/` with Tailwind CSS.
- [x] Build interactive 3D and 2D world map with clickable city markers, smooth projection, and instant country simulation.
- [x] Build real-time edge header inspection panel with live copyable code snippets (Next.js, Express, Go, cURL).
- [x] Build live proxy request stream viewer showing requests passing through GeoGate.
- [x] Build geofencing policy testing studio with visual rule builder and pass/block simulator.
- [x] Build high-throughput batch testing interface with CSV/JSON import, real-time benchmark stats, and export.
- [x] Build and embed the web assets directly into the Go executable via `//go:embed`.

### Phase 5: Hardening, Integration Tests & Documentation
- [x] Create comprehensive CLI commands (`serve`, `lookup`, `benchmark`, `version`).
- [x] Write end-to-end integration tests validating proxy header injection, batch throughput, and geofencing.
- [x] Write production-ready documentation (`README.md`, `docs/ARCHITECTURE.md`, `docs/INTEGRATION_GUIDE.md`).
- [x] Provide Dockerfile and multi-stage container build.
- [x] Verify clean compilation and execution with zero placeholders or mockups.

### Phase 6: Advanced Capabilities & Edge Intelligence
- [x] Chaos Latency Injection Engine (`internal/proxy/latency.go` with Haversine distance and `?simulate_latency=auto|100ms|250ms`).
- [x] Anti-Fraud & Security Intelligence Classification (`X-Geo-Connection-Type`, `X-Geo-Is-Datacenter`, `X-Geo-Is-Tor`, `X-Geo-Is-VPN`).
- [x] Purchasing Power Parity (PPP) Engine (`X-Geo-PPP-Factor`, `X-Geo-PPP-Discount`, and `GET /v1/ppp/:country` endpoint).
- [x] Proxy Replay Engine & Header Diff Viewer (`POST /v1/proxy/replay`, Header Diff drawer, and Replay modal in React 19 UI).
- [x] Open DB Downloader CLI (`internal/geoip/downloader.go` and `geogate db download --dir ./data`).
- [x] Declarative Configuration (`internal/config/config.go` with YAML/JSON support for custom CIDR routing rules).
