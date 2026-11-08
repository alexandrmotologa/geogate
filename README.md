# GeoGate

Self-hosted local IP geolocation, ASN intelligence, and edge geofencing gateway written in Go and React 19. Runs with less than 35MB RAM.

GeoGate replaces cloud GeoIP SaaS APIs (such as IPinfo and MaxMind Cloud) during local development and testing. It loads MaxMind MMDB datasets into memory, resolves lookups in under 1 microsecond, and functions as a reverse proxy that automatically injects standard CDN edge headers into incoming HTTP requests.

---

## Features

- In-Memory Radix Engine: Lookups resolve in 0.2 to 0.6 microseconds using a bitwise prefix trie.
- Edge Header Injection: Injects standard headers (`CF-IPCountry`, `X-Geo-City`, `X-Geo-Latitude`, `X-Geo-Longitude`, `X-Geo-EU`, `X-ASN`, `X-Geo-Currency`) into proxied requests.
- Purchasing Power Parity (PPP) Engine: Injects `X-Geo-PPP-Factor` and `X-Geo-PPP-Discount` headers for dynamic localized pricing. Dedicated endpoint at `GET /v1/ppp/:country`.
- Security & Anti-Fraud Classification: Identifies datacenters (AWS, GCP, Azure, Hetzner, DigitalOcean), VPNs, and Tor exit nodes (`X-Geo-Connection-Type`, `X-Geo-Is-Datacenter`, `X-Geo-Is-Tor`).
- Chaos Latency Simulation: Inject realistic transatlantic or transpacific round-trip delays (`?simulate_latency=auto` or `?simulate_latency=150ms`).
- Request Replay & Header Diff: Replay any captured request as a different country or security persona via `POST /v1/proxy/replay`, and inspect incoming vs injected header diffs.
- Traffic Simulation: Test geographic behavior by clicking on an interactive 3D globe or passing query parameters (`?simulate_country=JP`), custom headers (`X-Simulate-Country`), or cookies (`geogate_simulate_country`).
- High-Throughput Batch API: Goroutine worker pool on `POST /v1/batch/lookup` processes over 500,000 IP lookups per second.
- Geofencing Matrix: Evaluate access rules including country allow/block lists, ASN restrictions, and European Union (EU) boundaries.
- Live Traffic Stream: Server-Sent Events (SSE) stream on `GET /v1/events` displays real-time requests passing through the gateway.
- Embedded Studio UI: Built with React 19, TypeScript, and Tailwind CSS, compiled directly into the single Go executable.
- Open DB Downloader: `geogate db download` fetches and extracts the latest DB-IP open database.
- Declarative Configuration: Define custom CIDR overrides and routing rules in `geogate.yaml`.
- Prometheus Metrics: Native exporter on `/metrics` tracking throughput, latency, and memory consumption.
- Zero External Dependencies: Built-in dataset covers all 249 ISO 3166-1 territories, major world cities, and network subnets out of the box.

---

## Architecture Overview

```
[ Developer Browser / Test Runner ] (e.g. GET /pricing?simulate_country=JP&simulate_conn=tor)
                  |
                  v
+----------------------------------------------------+
| GeoGate Gateway Daemon (:8080)                     |
|                                                    |
| 1. Simulation Override & Security Persona Resolver |
|    - Evaluates ?simulate_country=...               |
|    - Evaluates ?simulate_conn=tor|datacenter       |
|    - Evaluates ?simulate_latency=auto|200ms        |
|    - Evaluates X-Simulate-Country header           |
|    - Evaluates geogate_simulate_country cookie     |
|                                                    |
| 2. In-Memory MMDB Trie (< 1 microsecond lookup)    |
|    - Resolves Country, City, Coordinates, EU, ASN  |
|    - Computes PPP discount and factor              |
|                                                    |
| 3. Edge Header Synthesizer                         |
|    - CF-IPCountry: JP                              |
|    - X-Geo-City: Tokyo                             |
|    - X-Geo-EU: 0                                   |
|    - X-ASN: AS9370                                 |
|    - X-Geo-PPP-Factor: 0.75                        |
|    - X-Geo-PPP-Discount: 25                        |
|    - X-Geo-Connection-Type: tor                    |
|    - X-Geo-Is-Tor: 1                               |
|                                                    |
| 4. Reverse Proxy Streamer & Chaos Delay Injector   |
+-----------------+----------------------------------+
                  | Proxied Request with Edge Headers
                  v
[ Local App (Next.js, Express, FastAPI, Django on :3000) ]
```

---

## Quick Start

### 1. Build and Run from Source

Requires Go 1.23+ and Node 20+:

```bash
git clone https://github.com/alexandrmotologa/geogate.git
cd geogate

# Build web studio and binary
make all

# Start GeoGate proxying to your local app on :3000
./bin/geogate serve -upstream http://localhost:3000 -port 8080
```

Open `http://localhost:8080/` in your browser to launch the GeoGate World Map Studio.

### 2. Run with Docker

```bash
docker build -t geogate:latest .
docker run -p 8080:8080 geogate:latest serve -upstream http://host.docker.internal:3000
```

---

## CLI Usage

GeoGate includes utility commands for command-line lookups, database downloads, and performance benchmarking:

```bash
# Resolve a single IP
geogate lookup 8.8.8.8

# Run an in-memory throughput benchmark with 100,000 lookups
geogate benchmark 100000

# Download the latest open DB-IP MMDB database file to ./data
geogate db download --dir ./data

# Start gateway daemon with custom options and declarative rules
geogate serve -upstream http://localhost:4000 -port 8080 -config geogate.yaml
```

### CLI Flags

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-port` | `8080` | Port for the HTTP server to listen on |
| `-upstream` | `http://localhost:3000` | Target URL to forward incoming requests |
| `-default-country` | `US` | Fallback ISO country code for local loopback traffic |
| `-config` | `""` | Path to declarative geogate.yaml configuration file |
| `-city-db` | `""` | Path to optional external MaxMind GeoLite2/GeoIP2 City MMDB file |
| `-asn-db` | `""` | Path to optional external MaxMind GeoLite2/GeoIP2 ASN MMDB file |
| `-simulate-latency` | `""` | Global chaos latency injection (e.g. `auto`, `150ms`) |
| `-cors` | `true` | Enable CORS headers on all API endpoints |
| `-max-batch` | `100000` | Maximum number of IPs allowed in a single batch request |

---

## Declarative Configuration (`geogate.yaml`)

You can define custom IP-to-Country rules and default behavior in a YAML file:

```yaml
port: 8080
upstream_url: "http://localhost:3000"
default_country: "US"
simulate_latency: "auto"

rules:
  - cidr: "192.168.1.0/24"
    country: "RO"
    city: "Cluj-Napoca"
    asn: 8708
    asn_org: "Custom Internal Network"
  - cidr: "10.0.0.0/8"
    country: "DE"
    city: "Frankfurt"
    is_datacenter: true
```

Launch GeoGate with your config:
```bash
geogate serve -config geogate.yaml
```

---

## API Endpoints

### 1. Single IP Lookup
```http
GET /v1/lookup/:ip
GET /v1/lookup/me
```

Example Request:
```bash
curl http://localhost:8080/v1/lookup/8.8.8.8
```

Example Response:
```json
{
  "record": {
    "ip": "8.8.8.8",
    "country_code": "US",
    "country_name": "United States",
    "continent_code": "NA",
    "city": "Mountain View",
    "latitude": 37.0902,
    "longitude": -95.7129,
    "timezone": "America/New_York",
    "is_eu": false,
    "currency": "USD",
    "ppp_factor": 1.0,
    "ppp_discount": 0,
    "connection_type": "datacenter",
    "is_datacenter": true,
    "is_vpn": false,
    "is_tor": false,
    "asn": {
      "number": 15169,
      "organization": "Google LLC",
      "network": "8.8.8.0/24"
    }
  },
  "duration_microseconds": 1
}
```

### 2. Purchasing Power Parity (PPP)
```http
GET /v1/ppp/:country
```

Example Request:
```bash
curl http://localhost:8080/v1/ppp/RO
```

Example Response:
```json
{
  "country_code": "RO",
  "country_name": "Romania",
  "currency": "RON",
  "ppp_factor": 0.48,
  "recommended_discount": 52,
  "is_eu": true
}
```

### 3. Request Replay Engine
```http
POST /v1/proxy/replay
Content-Type: application/json
```

Payload:
```json
{
  "method": "GET",
  "path": "/api/pricing",
  "simulate_country": "JP",
  "simulate_conn": "residential",
  "simulate_latency": "100ms"
}
```

Response:
```json
{
  "success": true,
  "status_code": 200,
  "duration_ms": 102.4,
  "simulated_country": "JP",
  "injected_headers": {
    "CF-IPCountry": "JP",
    "X-Geo-City": "Tokyo",
    "X-Geo-PPP-Factor": "0.75",
    "X-Geo-PPP-Discount": "25"
  },
  "upstream_response": "{\"status\":\"ok\"}"
}
```

### 4. High-Throughput Batch Lookup
```http
POST /v1/batch/lookup
Content-Type: application/json
```

Payload:
```json
{
  "ips": ["8.8.8.8", "88.198.24.1", "86.120.10.5", "133.242.1.1"]
}
```

Response:
```json
{
  "total_processed": 4,
  "results": [ ... ],
  "duration_microseconds": 240,
  "throughput_ips_per_sec": 540000.0
}
```

### 5. Geofence Rule Evaluator
```http
POST /v1/geofence/evaluate
Content-Type: application/json
```

Payload:
```json
{
  "policy": {
    "name": "EU GDPR Boundary",
    "require_eu": true,
    "blocked_countries": ["RU", "CN"]
  },
  "ips": ["88.198.24.1", "8.8.8.8"]
}
```

Response:
```json
{
  "policy_name": "EU GDPR Boundary",
  "total": 2,
  "allowed_count": 1,
  "blocked_count": 1,
  "results": [
    {
      "ip": "88.198.24.1",
      "allowed": true,
      "reason": "Passed all geofence policy rules",
      "matched_rule": "Pass",
      "record": { "country_code": "DE", "city": "Berlin" }
    },
    {
      "ip": "8.8.8.8",
      "allowed": false,
      "reason": "Country US is not an EU member state",
      "matched_rule": "RequireEU",
      "record": { "country_code": "US", "city": "Mountain View" }
    }
  ]
}
```

---

## Edge Header Reference

When a request passes through the reverse proxy, GeoGate adds the following headers before forwarding:

| Header | Example | Notes |
| :--- | :--- | :--- |
| `CF-IPCountry` | `JP` | ISO 3166-1 alpha-2 country code |
| `X-Country-Code` | `JP` | ISO 3166-1 alpha-2 country code |
| `X-Country-Name` | `Japan` | Full English country name |
| `X-Geo-City` | `Tokyo` | City name |
| `X-Geo-Latitude` | `35.6762` | Geographic latitude |
| `X-Geo-Longitude` | `139.6503` | Geographic longitude |
| `X-Geo-Timezone` | `Asia/Tokyo` | IANA timezone identifier |
| `X-Geo-EU` | `0` or `1` | 1 if country is in the European Union, 0 otherwise |
| `X-ASN` | `AS9370` | Autonomous System Number |
| `X-ASN-Org` | `Sakura Internet Inc.` | Autonomous System Organization name |
| `X-Geo-Currency` | `JPY` | ISO 4217 currency code |
| `X-Geo-PPP-Factor` | `0.75` | Purchasing Power Parity factor relative to USD |
| `X-Geo-PPP-Discount` | `25` | Recommended localized pricing discount percentage |
| `X-Geo-Connection-Type` | `residential` | Connection type: residential, datacenter, vpn, or tor |
| `X-Geo-Is-Datacenter` | `0` or `1` | 1 if IP belongs to a cloud or hosting provider |
| `X-Geo-Is-VPN` | `0` or `1` | 1 if IP is associated with known commercial VPNs |
| `X-Geo-Is-Tor` | `0` or `1` | 1 if traffic simulates or originates from a Tor exit |
| `X-GeoGate-Injected-Latency` | `125ms` | Added when chaos latency injection is active |
| `X-GeoGate-Simulated` | `true` | Added when an override rule is active |

---

## Performance Benchmarks

Run on an 11th Gen Intel Core i5-1135G7:

| Operation | Metric | Value | Target |
| :--- | :--- | :--- | :--- |
| Single IP Lookup | Latency | **0.222 microseconds** | &lt; 8 microseconds |
| In-Memory Engine | Throughput | **4,310,000 lookups/sec** | &gt; 1,000,000/sec |
| Batch API (HTTP) | Throughput | **1,900,000+ IPs/sec** | &gt; 150,000/sec |
| Daemon Memory | Heap Alloc | **12.4 MB** | &lt; 35 MB |

---

## Documentation

Detailed technical documentation is available in the `docs/` directory:
- [Architecture & Internals](docs/ARCHITECTURE.md)
- [Application Integration Guide](docs/INTEGRATION_GUIDE.md)

---

## License

MIT License. See [LICENSE](LICENSE) for details.
