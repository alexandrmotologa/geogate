export interface ASNRecord {
  number: number;
  organization: string;
  network?: string;
  is_datacenter?: boolean;
  is_vpn?: boolean;
  is_tor?: boolean;
}

export interface GeoRecord {
  ip: string;
  country_code: string;
  country_name: string;
  continent_code: string;
  city: string;
  postal_code?: string;
  latitude: number;
  longitude: number;
  timezone: string;
  is_eu: boolean;
  currency: string;
  ppp_factor?: number;
  ppp_discount?: number;
  connection_type?: string;
  is_datacenter?: boolean;
  is_vpn?: boolean;
  is_tor?: boolean;
  asn: ASNRecord;
  simulated?: boolean;
}

export interface CityEntry {
  name: string;
  country_code: string;
  country_name: string;
  latitude: number;
  longitude: number;
  timezone: string;
  sample_ip: string;
  is_eu: boolean;
  currency: string;
  ppp_factor?: number;
  ppp_discount?: number;
  asn: number;
  asn_org: string;
}

export interface CountryInfo {
  code: string;
  name: string;
  continent: string;
  capital: string;
  latitude: number;
  longitude: number;
  timezone: string;
  currency: string;
  is_eu: boolean;
  ppp_factor?: number;
  ppp_discount?: number;
  sample_ip: string;
  default_asn: number;
  asn_org: string;
}

export interface EngineStatus {
  status: string;
  version: string;
  uptime_seconds: number;
  memory_alloc_mb: number;
  memory_sys_mb: number;
  num_goroutine: number;
  total_lookups: number;
  has_external_db: boolean;
  default_country: string;
  upstream_url: string;
}

export interface RequestEvent {
  id: number;
  timestamp: string;
  method: string;
  path: string;
  client_ip: string;
  simulated: boolean;
  country_code: string;
  country_name: string;
  city: string;
  status_code: number;
  duration_ms: number;
  headers: Record<string, string>;
  original_headers?: Record<string, string>;
  injected_latency?: string;
  connection_type?: string;
}

export interface ReplayRequest {
  event_id?: number;
  method: string;
  path: string;
  simulate_country: string;
  simulate_city?: string;
  simulate_conn?: string;
  simulate_latency?: string;
}

export interface ReplayResponse {
  success: boolean;
  status_code: number;
  duration_ms: number;
  simulated_country: string;
  injected_headers: Record<string, string>;
  upstream_response?: string;
  error?: string;
}

export interface GeofencePolicy {
  name: string;
  allowed_countries?: string[];
  blocked_countries?: string[];
  require_eu?: boolean;
  allowed_continents?: string[];
  allowed_asns?: number[];
  blocked_asns?: number[];
}

export interface GeofenceEvaluationItem {
  ip: string;
  allowed: boolean;
  reason: string;
  matched_rule: string;
  record: GeoRecord | null;
}

export interface GeofenceEvaluationResponse {
  policy_name: string;
  total: number;
  allowed_count: number;
  blocked_count: number;
  results: GeofenceEvaluationItem[];
}

export interface BatchLookupResponse {
  total_processed: number;
  results: GeoRecord[];
  duration_microseconds: number;
  throughput_ips_per_sec: number;
}
