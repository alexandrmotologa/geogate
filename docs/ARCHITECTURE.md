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

The event hub (`internal/proxy/hub.go`) retains a ring buffer of the 150 most recent transactions and broadcasts them over Server-Sent Events (`GET /v1/events`). This allows the browser studio to visualize live traffic without polling. Each event records both original client headers and GeoGate injected headers, enabling instant header diffing.

## Chaos Latency Simulation

The latency simulation engine (`internal/proxy/latency.go`) calculates realistic network round-trip delays between the client location and the upstream application. When `?simulate_latency=auto` is specified, GeoGate computes the great-circle Haversine distance and adds light-in-fiber transit delay plus typical peering jitter (ranging between 20ms and 240ms). Explicit durations such as `?simulate_latency=150ms` are also accepted. Injected latency is communicated to clients via the `X-GeoGate-Injected-Latency` header.

## Purchasing Power Parity (PPP) Engine

The PPP engine models World Bank and IMF economic indicators for 249 territories. It evaluates local currency purchasing power against standard USD benchmark pricing, computing an index factor (`X-Geo-PPP-Factor`) and a recommended localized pricing discount percentage (`X-Geo-PPP-Discount`). Applications can query country statistics via `GET /v1/ppp/:country`.

## Anti-Fraud and Security Intelligence

GeoGate identifies incoming connection types using Autonomous System classifications:
- Datacenters (`X-Geo-Is-Datacenter: 1`): Cloud providers including AWS, GCP, Azure, Hetzner, DigitalOcean, OVH, and Vultr.
- VPN and Proxies (`X-Geo-Is-VPN: 1`): Known commercial VPN routing ASNs.
- Tor Exit Nodes (`X-Geo-Is-Tor: 1`): Simulated or recognized Tor onion router exits.
- Connection classification: Exposed as `X-Geo-Connection-Type` (`residential`, `datacenter`, `vpn`, or `tor`). Developers can simulate any security persona with `?simulate_conn=tor` or `?simulate_conn=datacenter`.

## Declarative Configuration

Custom subnet classifications, city overrides, and gateway settings can be defined in `geogate.yaml` or `geogate.json`. GeoGate parses these rules at startup, building an augmented trie that overrides public database resolutions for designated enterprise CIDRs.

