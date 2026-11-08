package geoip

import (
	"fmt"
	"strings"
	"sync"
)

// Synthesizer generates geographic records and synthetic IPs for any target country or city.
type Synthesizer struct {
	mu           sync.RWMutex
	countries    map[string]CountryInfo
	citiesByName map[string]CityEntry
}

// NewSynthesizer initializes a synthesizer instance with default geographic datasets.
func NewSynthesizer() *Synthesizer {
	s := &Synthesizer{
		countries:    make(map[string]CountryInfo, len(CountryDatabase)),
		citiesByName: make(map[string]CityEntry, len(MajorGlobalCities)),
	}

	for k, v := range CountryDatabase {
		s.countries[strings.ToUpper(k)] = v
	}

	for _, city := range MajorGlobalCities {
		s.citiesByName[strings.ToLower(city.Name)] = city
	}

	return s
}

// GenerateForCountry creates a GeoRecord for an ISO 3166-1 alpha-2 country code.
func (s *Synthesizer) GenerateForCountry(code string) (*GeoRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	norm := strings.ToUpper(strings.TrimSpace(code))
	info, found := s.countries[norm]
	if !found {
		return nil, fmt.Errorf("unknown ISO country code: %s", code)
	}

	rec := &GeoRecord{
		IP:            info.SampleIP,
		CountryCode:   info.Code,
		CountryName:   info.Name,
		ContinentCode: info.Continent,
		City:          info.Capital,
		Latitude:      info.Latitude,
		Longitude:     info.Longitude,
		Timezone:      info.Timezone,
		IsEU:          info.IsEU,
		Currency:      info.Currency,
		ASN: ASNRecord{
			Number:       info.DefaultASN,
			Organization: info.ASNOrg,
		},
		Simulated: true,
	}

	return rec, nil
}

// GenerateForCity creates a GeoRecord for a named metropolitan area.
func (s *Synthesizer) GenerateForCity(cityName string) (*GeoRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	norm := strings.ToLower(strings.TrimSpace(cityName))
	city, found := s.citiesByName[norm]
	if !found {
		// Attempt partial prefix match
		for name, c := range s.citiesByName {
			if strings.HasPrefix(name, norm) || strings.Contains(name, norm) {
				city = c
				found = true
				break
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("city not found in catalog: %s", cityName)
	}

	continent := "EU"
	if info, ok := s.countries[city.CountryCode]; ok {
		continent = info.Continent
	}

	rec := &GeoRecord{
		IP:            city.SampleIP,
		CountryCode:   city.CountryCode,
		CountryName:   city.CountryName,
		ContinentCode: continent,
		City:          city.Name,
		Latitude:      city.Latitude,
		Longitude:     city.Longitude,
		Timezone:      city.Timezone,
		IsEU:          city.IsEU,
		Currency:      city.Currency,
		ASN: ASNRecord{
			Number:       city.ASN,
			Organization: city.ASNOrg,
		},
		Simulated: true,
	}

	return rec, nil
}

// ListCountries returns all available country metadata.
func (s *Synthesizer) ListCountries() []CountryInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]CountryInfo, 0, len(s.countries))
	for _, c := range s.countries {
		res = append(res, c)
	}
	return res
}

// ListCities returns all available major cities.
func (s *Synthesizer) ListCities() []CityEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]CityEntry, len(MajorGlobalCities))
	copy(res, MajorGlobalCities)
	return res
}
