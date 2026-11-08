export interface ASNRecord {
  number: number;
  organization: string;
  network?: string;
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
