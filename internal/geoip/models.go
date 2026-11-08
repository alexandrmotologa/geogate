package geoip

// ASNRecord holds autonomous system metadata and connection classifications.
type ASNRecord struct {
	Number       uint   `json:"number"`
	Organization string `json:"organization"`
	Network      string `json:"network,omitempty"`
	IsDatacenter bool   `json:"is_datacenter"`
	IsVPN        bool   `json:"is_vpn"`
	IsTor        bool   `json:"is_tor"`
}

// GeoRecord holds enriched geographic, network intelligence, security classifications, and PPP metrics.
type GeoRecord struct {
	IP             string    `json:"ip"`
	CountryCode    string    `json:"country_code"`
	CountryName    string    `json:"country_name"`
	ContinentCode  string    `json:"continent_code"`
	City           string    `json:"city"`
	PostalCode     string    `json:"postal_code,omitempty"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	Timezone       string    `json:"timezone"`
	IsEU           bool      `json:"is_eu"`
	Currency       string    `json:"currency"`
	PPPFactor      float64   `json:"ppp_factor"`
	PPPDiscount    int       `json:"ppp_discount"`
	ConnectionType string    `json:"connection_type"`
	IsDatacenter   bool      `json:"is_datacenter"`
	IsVPN          bool      `json:"is_vpn"`
	IsTor          bool      `json:"is_tor"`
	ASN            ASNRecord `json:"asn"`
	Simulated      bool      `json:"simulated,omitempty"`
}

// CityEntry represents a prominent global metropolitan area for map interaction.
type CityEntry struct {
	Name        string  `json:"name"`
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Timezone    string  `json:"timezone"`
	SampleIP    string  `json:"sample_ip"`
	IsEU        bool    `json:"is_eu"`
	Currency    string  `json:"currency"`
	PPPFactor   float64 `json:"ppp_factor"`
	PPPDiscount int     `json:"ppp_discount"`
	ASN         uint    `json:"asn"`
	ASNOrg      string  `json:"asn_org"`
}

// CountryInfo contains ISO country metadata, coordinates, PPP indexes, and default representative IP.
type CountryInfo struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Continent   string  `json:"continent"`
	Capital     string  `json:"capital"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Timezone    string  `json:"timezone"`
	Currency    string  `json:"currency"`
	IsEU        bool    `json:"is_eu"`
	PPPFactor   float64 `json:"ppp_factor"`
	PPPDiscount int     `json:"ppp_discount"`
	SampleIP    string  `json:"sample_ip"`
	DefaultASN  uint    `json:"default_asn"`
	ASNOrg      string  `json:"asn_org"`
}

// MaxMindCityRecord decodes MaxMind City / GeoLite2-City MMDB data.
type MaxMindCityRecord struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Continent struct {
		Code  string            `maxminddb:"code"`
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"continent"`
	Country struct {
		IsoCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
		IsInEU  bool              `maxminddb:"is_in_european_union"`
	} `maxminddb:"country"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
		TimeZone  string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`
	Postal struct {
		Code string `maxminddb:"code"`
	} `maxminddb:"postal"`
}

// MaxMindASNRecord decodes MaxMind ASN / GeoLite2-ASN MMDB data.
type MaxMindASNRecord struct {
	AutonomousSystemNumber       uint   `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
}
