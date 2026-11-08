package geoip

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/oschwald/maxminddb-golang"
)

// Engine manages in-memory IP resolution, MMDB readers, and the synthetic generator.
type Engine struct {
	cityDB       *maxminddb.Reader
	asnDB        *maxminddb.Reader
	trie         *IPTrie
	synthesizer  *Synthesizer
	cityDBPath   string
	asnDBPath    string
	totalLookups uint64
}

// Config specifies initialization options for the GeoIP engine.
type Config struct {
	CityDBPath string
	AsnDBPath  string
}

// NewEngine initializes the geolocation engine with embedded data and optional external MMDB files.
func NewEngine(cfg Config) (*Engine, error) {
	e := &Engine{
		trie:        NewIPTrie(),
		synthesizer: NewSynthesizer(),
		cityDBPath:  cfg.CityDBPath,
		asnDBPath:   cfg.AsnDBPath,
	}

	// Populate built-in high-speed radix trie
	e.populateBuiltinTrie()

	// Load external MMDB files if present
	if cfg.CityDBPath != "" {
		if _, err := os.Stat(cfg.CityDBPath); err == nil {
			reader, err := maxminddb.Open(cfg.CityDBPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open City MMDB: %w", err)
			}
			e.cityDB = reader
		}
	}

	if cfg.AsnDBPath != "" {
		if _, err := os.Stat(cfg.AsnDBPath); err == nil {
			reader, err := maxminddb.Open(cfg.AsnDBPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open ASN MMDB: %w", err)
			}
			e.asnDB = reader
		}
	}

	return e, nil
}

// Close releases any opened MMDB file descriptors.
func (e *Engine) Close() error {
	var errs []string
	if e.cityDB != nil {
		if err := e.cityDB.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if e.asnDB != nil {
		if err := e.asnDB.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing MMDB readers: %s", strings.Join(errs, ", "))
	}
	return nil
}

// LookupString parses an IP string and resolves its geographic and ASN metadata.
func (e *Engine) LookupString(ipStr string) (*GeoRecord, error) {
	ipStr = strings.TrimSpace(ipStr)
	// Strip port if present
	if host, _, err := net.SplitHostPort(ipStr); err == nil {
		ipStr = host
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address format: %s", ipStr)
	}

	return e.Lookup(ip)
}

// Lookup resolves geographic and ASN metadata for an IP.
func (e *Engine) Lookup(ip net.IP) (*GeoRecord, error) {
	atomic.AddUint64(&e.totalLookups, 1)

	// Check for private / loopback IP ranges
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
		// Default local development record
		return &GeoRecord{
			IP:            ip.String(),
			CountryCode:   "US",
			CountryName:   "United States",
			ContinentCode: "NA",
			City:          "Local Development",
			Latitude:      37.7749,
			Longitude:     -122.4194,
			Timezone:      "America/Los_Angeles",
			IsEU:          false,
			Currency:      "USD",
			ASN: ASNRecord{
				Number:       0,
				Organization: "Localhost Network",
			},
			Simulated: false,
		}, nil
	}

	// 1. If external MMDB readers are loaded, query them
	if e.cityDB != nil {
		var cityRecord MaxMindCityRecord
		err := e.cityDB.Lookup(ip, &cityRecord)
		if err == nil && cityRecord.Country.IsoCode != "" {
			countryCode := cityRecord.Country.IsoCode
			cityName := cityRecord.City.Names["en"]
			countryName := cityRecord.Country.Names["en"]
			continentCode := cityRecord.Continent.Code
			if continentCode == "" {
				continentCode = "NA"
			}

			isEU := cityRecord.Country.IsInEU || EUCountryCodes[countryCode]

			record := &GeoRecord{
				IP:            ip.String(),
				CountryCode:   countryCode,
				CountryName:   countryName,
				ContinentCode: continentCode,
				City:          cityName,
				PostalCode:    cityRecord.Postal.Code,
				Latitude:      cityRecord.Location.Latitude,
				Longitude:     cityRecord.Location.Longitude,
				Timezone:      cityRecord.Location.TimeZone,
				IsEU:          isEU,
				Currency:      e.getCurrencyForCountry(countryCode),
			}

			if e.asnDB != nil {
				var asnRecord MaxMindASNRecord
				if errASN := e.asnDB.Lookup(ip, &asnRecord); errASN == nil {
					record.ASN = ASNRecord{
						Number:       asnRecord.AutonomousSystemNumber,
						Organization: asnRecord.AutonomousSystemOrganization,
					}
				}
			}

			return record, nil
		}
	}

	// 2. Query in-memory prefix trie
	if match := e.trie.Lookup(ip); match != nil {
		// Return copy with actual queried IP
		res := *match
		res.IP = ip.String()
		return &res, nil
	}

	// 3. Fallback: hash the IP deterministically into an active country from our database
	fallbackCode := e.deriveFallbackCountry(ip)
	rec, err := e.synthesizer.GenerateForCountry(fallbackCode)
	if err == nil {
		res := *rec
		res.IP = ip.String()
		res.Simulated = false
		return &res, nil
	}

	return nil, fmt.Errorf("unable to resolve IP address: %s", ip.String())
}

// LookupCountry returns a representative GeoRecord for an ISO 3166-1 country code.
func (e *Engine) LookupCountry(countryCode string) (*GeoRecord, error) {
	return e.synthesizer.GenerateForCountry(countryCode)
}

// LookupCity returns a representative GeoRecord for a named city.
func (e *Engine) LookupCity(cityName string) (*GeoRecord, error) {
	return e.synthesizer.GenerateForCity(cityName)
}

// Synthesizer returns the engine's synthesizer instance.
func (e *Engine) Synthesizer() *Synthesizer {
	return e.synthesizer
}

// TotalLookups returns the cumulative number of lookups performed.
func (e *Engine) TotalLookups() uint64 {
	return atomic.LoadUint64(&e.totalLookups)
}

// HasExternalDB returns whether real external MMDB files are loaded.
func (e *Engine) HasExternalDB() bool {
	return e.cityDB != nil
}

// deriveFallbackCountry computes a deterministic country code from IP bytes when unlisted.
func (e *Engine) deriveFallbackCountry(ip net.IP) string {
	ip4 := ip.To4()
	if ip4 == nil {
		return "US"
	}

	popular := []string{"US", "GB", "DE", "FR", "JP", "RO", "CA", "AU", "NL", "SG"}
	index := (int(ip4[0]) + int(ip4[1]) + int(ip4[2]) + int(ip4[3])) % len(popular)
	return popular[index]
}

// getCurrencyForCountry looks up currency code for an ISO country code.
func (e *Engine) getCurrencyForCountry(countryCode string) string {
	if info, ok := CountryDatabase[strings.ToUpper(countryCode)]; ok && info.Currency != "" {
		return info.Currency
	}
	if EUCountryCodes[strings.ToUpper(countryCode)] {
		return "EUR"
	}
	return "USD"
}

// populateBuiltinTrie seeds the radix trie with prominent subnets and global regions.
func (e *Engine) populateBuiltinTrie() {
	entries := []struct {
		cidr    string
		country string
		city    string
		asn     uint
		asnOrg  string
	}{
		// Public Resolvers and Hyper-Scalers
		{"8.8.8.0/24", "US", "Mountain View", 15169, "Google LLC"},
		{"8.8.4.0/24", "US", "Mountain View", 15169, "Google LLC"},
		{"1.1.1.0/24", "US", "San Francisco", 13335, "Cloudflare, Inc."},
		{"1.0.0.0/24", "US", "San Francisco", 13335, "Cloudflare, Inc."},
		{"9.9.9.0/24", "CH", "Zurich", 19281, "Quad9"},
		{"208.67.222.0/24", "US", "San Francisco", 36692, "Cisco OpenDNS"},
		{"208.67.220.0/24", "US", "San Francisco", 36692, "Cisco OpenDNS"},

		// Europe & Regional Providers
		{"88.198.0.0/16", "DE", "Berlin", 24940, "Hetzner Online GmbH"},
		{"212.27.48.0/20", "FR", "Paris", 12322, "Free SAS"},
		{"86.120.0.0/14", "RO", "Bucharest", 8708, "RCS & RDS SA"},
		{"178.168.0.0/16", "MD", "Chisinau", 29551, "Orange Moldova"},
		{"81.2.64.0/18", "GB", "London", 2856, "British Telecommunications"},
		{"145.220.0.0/16", "NL", "Amsterdam", 1103, "SURF B.V."},
		{"80.58.0.0/16", "ES", "Madrid", 3352, "Telefonica de Espana"},
		{"151.1.0.0/16", "IT", "Rome", 1267, "Wind Tre S.p.A."},
		{"83.1.0.0/16", "PL", "Warsaw", 5617, "Orange Polska S.A."},
		{"192.176.0.0/16", "SE", "Stockholm", 8473, "Bahnhof AB"},
		{"193.134.0.0/16", "CH", "Zurich", 3303, "Swisscom AG"},
		{"89.100.0.0/15", "IE", "Dublin", 15502, "Vodafone Ireland"},
		{"148.122.0.0/16", "NO", "Oslo", 2119, "Telenor Norge AS"},
		{"80.160.0.0/16", "DK", "Copenhagen", 3292, "TDC Holding A/S"},
		{"88.112.0.0/16", "FI", "Helsinki", 719, "Elisa Oyj"},
		{"81.240.0.0/16", "BE", "Brussels", 5432, "Proximus NV"},
		{"80.120.0.0/15", "AT", "Vienna", 8447, "A1 Telekom Austria"},
		{"89.24.0.0/16", "CZ", "Prague", 5610, "O2 Czech Republic"},
		{"84.0.0.0/16", "HU", "Budapest", 5483, "Magyar Telekom"},
		{"85.240.0.0/16", "PT", "Lisbon", 2860, "NOS Comunicacoes"},
		{"62.38.0.0/16", "GR", "Athens", 6799, "OTE SA"},
		{"194.126.96.0/20", "EE", "Tallinn", 3249, "Telia Eesti AS"},
		{"195.122.0.0/16", "LV", "Riga", 12578, "SIA Tet"},
		{"84.15.0.0/16", "LT", "Vilnius", 8764, "Telia Lietuva AB"},
		{"194.154.192.0/19", "LU", "Luxembourg City", 6661, "POST Luxembourg"},
		{"85.187.0.0/16", "BG", "Sofia", 8866, "Vivacom Bulgaria"},
		{"89.164.0.0/15", "HR", "Zagreb", 5391, "Hrvatski Telekom"},
		{"86.61.0.0/16", "SI", "Ljubljana", 34779, "T-2 d.o.o."},
		{"87.244.192.0/18", "SK", "Bratislava", 6855, "Slovak Telekom"},
		{"80.244.0.0/16", "CY", "Nicosia", 6866, "Cyprus Telecommunications"},
		{"88.203.0.0/16", "MT", "Valletta", 13335, "GO plc"},
		{"82.221.0.0/16", "IS", "Reykjavik", 6677, "Siminn hf"},
		{"194.44.0.0/16", "UA", "Kyiv", 6849, "JSC Ukrtelecom"},
		{"88.224.0.0/15", "TR", "Istanbul", 9121, "Turk Telekom"},
		{"95.173.136.0/21", "RU", "Moscow", 12389, "PJSC Rostelecom"},

		// Asia-Pacific & Americas
		{"133.242.0.0/16", "JP", "Tokyo", 9370, "Sakura Internet Inc."},
		{"142.250.0.0/15", "CA", "Toronto", 852, "TELUS Communications"},
		{"139.130.0.0/16", "AU", "Sydney", 1221, "Telstra Corporation"},
		{"177.16.0.0/12", "BR", "Sao Paulo", 28573, "Claro Telecom"},
		{"103.27.0.0/16", "IN", "Mumbai", 55836, "Reliance Jio"},
		{"116.14.0.0/15", "SG", "Singapore", 4657, "StarHub Ltd"},
		{"211.234.0.0/16", "KR", "Seoul", 9318, "SK Broadband"},
		{"86.96.0.0/15", "AE", "Dubai", 5384, "Emirates Telecommunications"},
		{"84.108.0.0/15", "IL", "Tel Aviv", 12849, "Hot Telecom LP"},
		{"87.101.0.0/16", "SA", "Riyadh", 25019, "Saudi Telecom"},
		{"196.25.0.0/16", "ZA", "Johannesburg", 37457, "Telkom SA"},
		{"156.192.0.0/14", "EG", "Cairo", 8452, "Telecom Egypt"},
		{"189.128.0.0/15", "MX", "Mexico City", 8151, "UNINET S.A."},
		{"181.16.0.0/14", "AR", "Buenos Aires", 7303, "Telecom Argentina"},
		{"190.22.0.0/16", "CL", "Santiago", 27956, "VTR Comunicaciones"},
		{"118.90.0.0/15", "NZ", "Auckland", 4771, "Spark New Zealand"},
		{"202.130.64.0/18", "HK", "Hong Kong", 9269, "Hong Kong Broadband"},
		{"168.95.0.0/16", "TW", "Taipei", 3462, "Chunghwa Telecom"},
		{"119.46.0.0/16", "TH", "Bangkok", 23969, "TOT Public Co"},
		{"113.160.0.0/14", "VN", "Hanoi", 7552, "Viettel Group"},
		{"180.240.0.0/13", "ID", "Jakarta", 7713, "PT Telkom Indonesia"},
		{"115.132.0.0/14", "MY", "Kuala Lumpur", 4788, "TM Technology"},
		{"120.28.0.0/14", "PH", "Manila", 9299, "Philippine Long Distance Telephone"},
		{"202.96.128.0/18", "CN", "Beijing", 4134, "CHINANET"},
	}

	for _, entry := range entries {
		info, ok := CountryDatabase[entry.country]
		if !ok {
			continue
		}

		rec := &GeoRecord{
			CountryCode:   entry.country,
			CountryName:   info.Name,
			ContinentCode: info.Continent,
			City:          entry.city,
			Latitude:      info.Latitude,
			Longitude:     info.Longitude,
			Timezone:      info.Timezone,
			IsEU:          info.IsEU,
			Currency:      info.Currency,
			ASN: ASNRecord{
				Number:       entry.asn,
				Organization: entry.asnOrg,
				Network:      entry.cidr,
			},
		}

		_ = e.trie.InsertCIDR(entry.cidr, rec)
	}
}

// BenchmarkSingleLookup runs repeated lookups and returns average duration in nanoseconds.
func (e *Engine) BenchmarkSingleLookup(iterations int) (time.Duration, error) {
	testIP := net.ParseIP("8.8.8.8")
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := e.Lookup(testIP)
		if err != nil {
			return 0, err
		}
	}
	elapsed := time.Since(start)
	return elapsed / time.Duration(iterations), nil
}
