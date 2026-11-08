# Architecture and Internals

GeoGate operates as an in-memory IP geolocation engine, edge header injection reverse proxy, and test simulator designed for local engineering workflows and high-throughput environments.

## Core Design Principles

1. Minimal Memory Footprint: GeoGate runs with less than 35 megabytes of resident heap memory using an in-memory binary radix trie.
2. Low Latency: Single IP lookups complete in under 1 microsecond in memory without external network calls.
3. Edge Compatibility: Injected headers mirror Cloudflare, Fastly, and AWS CloudFront conventions (`CF-IPCountry`, `X-Geo-City`, `X-Geo-EU`, `X-ASN`).
4. Zero External Dependencies: A built-in dataset covering all ISO 3166-1 territories, major global metro hubs, and network subnets allows immediate startup without downloading external database archives.

## Component Breakdown

```
[ Incoming Request (e.g., GET /checkout?simulate_country=JP) ]
                          |
                          v
+-------------------------------------------------------------+
| Reverse Proxy & Simulation Interceptor                      |
| - Inspects query params (?simulate_country=JP)              |
| - Inspects headers (X-Simulate-Country: JP)                 |
| - Inspects session cookies (geogate_simulate_country=JP)    |
| - Extracts client IP if no simulation override is present   |
+------------------------------+------------------------------+
                               | Target IP or Country Code
                               v
+-------------------------------------------------------------+
| In-Memory Engine (internal/geoip)                           |
| 1. External MaxMind MMDB Reader (if configured)             |
| 2. Bitwise IP Radix Trie (sub-microsecond longest match)    |
| 3. ISO Country & Metropolitan Area Synthesizer              |
+------------------------------+------------------------------+
                               | Resolved GeoRecord
                               v
+-------------------------------------------------------------+
| Edge Header Synthesizer (internal/proxy/headers.go)         |
| Injects:                                                    |
|   CF-IPCountry: JP                                          |
|   X-Country-Code: JP                                        |
|   X-Country-Name: Japan                                     |
|   X-Geo-City: Tokyo                                         |
|   X-Geo-Latitude: 35.6762                                   |
|   X-Geo-Longitude: 139.6503                                 |
|   X-Geo-Timezone: Asia/Tokyo                                |
|   X-Geo-EU: 0                                               |
|   X-ASN: AS9370                                             |
|   X-ASN-Org: Sakura Internet Inc.                           |
|   X-Geo-Currency: JPY                                       |
|   X-GeoGate-Simulated: true                                 |
+------------------------------+------------------------------+
                               |
                               +-----> Broadcast to EventHub (SSE)
                               |
                               v
[ Forwarded Request to Upstream Application (e.g. :3000) ]
```

## Trie Implementation

The IP lookup trie (`internal/geoip/trie.go`) implements a 32-level binary tree for IPv4 CIDR matching. Each bit from MSB to LSB corresponds to one branch step:
- Bit 1 navigates right, bit 0 navigates left.
- Each node stores the most specific known geographic record.
- Lookups require at most 32 bitwise operations, running in tens of nanoseconds.

## Reverse Proxy Behavior

The proxy handler (`internal/proxy/handler.go`) intercepts incoming traffic:
- Reserved paths (`/v1/*`, `/geogate/*`, `/metrics`, `/health`) route to the internal REST API and embedded React 19 studio.
- Application paths are rewritten with Edge headers and streamed to the configured upstream application via `httputil.ReverseProxy`.
- If the upstream service is offline, GeoGate responds with HTTP 502 Bad Gateway with diagnostic JSON rather than dropping the connection.

## Event Hub and Streaming

The event hub (`internal/proxy/hub.go`) retains a ring buffer of the 100 most recent transactions and broadcasts them over Server-Sent Events (`GET /v1/events`). This allows the browser studio to visualize live traffic without polling.
