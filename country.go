package phonex

import (
	"sort"
	"strings"

	"github.com/bakhod1r/phonex/countries"
)

// normalizeRegion accepts a region code in any case and returns the canonical
// upper-case form.
func normalizeRegion(region string) string {
	return strings.ToUpper(strings.TrimSpace(region))
}

// Country returns the metadata for an ISO-3166 alpha-2 region code.
func Country(iso2 string) (*Metadata, bool) {
	m, ok := countries.Data[normalizeRegion(iso2)]
	return m, ok
}

// CountryByDialCode returns the main region for a calling code. The code may
// be given with or without a leading '+'. Regions sharing a code (such as the
// many behind "+1") resolve to the main one; use RegionsForDialCode for all
// of them.
func CountryByDialCode(code string) (*Metadata, bool) {
	code = strings.TrimPrefix(strings.TrimSpace(code), "+")
	if regions := countries.RegionsForCode(code); len(regions) > 0 {
		return regions[0], true
	}
	if m, ok := countries.NonGeo[code]; ok {
		return m, true
	}
	return nil, false
}

// RegionsForDialCode returns every region sharing a calling code, main region
// first. The returned slice must not be modified.
func RegionsForDialCode(code string) []*Metadata {
	code = strings.TrimPrefix(strings.TrimSpace(code), "+")
	return countries.RegionsForCode(code)
}

// CountryByPhone parses a number and returns the metadata of its region.
func CountryByPhone(number string, opts ...ParseOption) (*Metadata, bool) {
	var p Phone
	if err := p.Parse(number, opts...); err != nil {
		return nil, false
	}
	return p.meta, true
}

// Countries returns the metadata of every region, ordered by ISO-3166
// alpha-2 code. Non-geographical ranges are not included; see NonGeoEntities.
func Countries() []*Metadata {
	out := make([]*Metadata, 0, len(countries.Data))
	for _, m := range countries.Data {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ISO2 < out[j].ISO2 })
	return out
}

// NonGeoEntities returns the metadata of the non-geographical ranges such as
// +800 (universal freephone), ordered by calling code.
func NonGeoEntities() []*Metadata {
	out := make([]*Metadata, 0, len(countries.NonGeo))
	for _, m := range countries.NonGeo {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DialCode < out[j].DialCode })
	return out
}

// SupportedRegions returns every ISO-3166 alpha-2 code with metadata, sorted.
func SupportedRegions() []string {
	out := make([]string, 0, len(countries.Data))
	for iso := range countries.Data {
		out = append(out, iso)
	}
	sort.Strings(out)
	return out
}

// SearchCountries returns the regions whose name or ISO code matches query,
// ordered by ISO-3166 alpha-2 code. Matching is case-insensitive: an exact
// alpha-2 or alpha-3 code matches that region alone, otherwise the query is
// matched as a substring of the country name.
func SearchCountries(query string) []*Metadata {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	if m, ok := countries.Data[strings.ToUpper(q)]; ok {
		return []*Metadata{m}
	}
	var out []*Metadata
	for _, m := range countries.Data {
		if strings.ToLower(m.ISO3) == q || strings.Contains(strings.ToLower(m.Name), q) {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ISO2 < out[j].ISO2 })
	return out
}

// ExampleNumber returns a valid example number for a region, or nil if the
// metadata carries none. Examples come from libphonenumber and are safe to
// use in documentation and tests.
func ExampleNumber(region string) (*Phone, bool) {
	m, ok := countries.Data[normalizeRegion(region)]
	if !ok {
		return nil, false
	}
	for t := PhoneType(0); t < countries.NumDescs; t++ {
		if ex := m.Descs[t].Example; ex != "" {
			if p, err := Parse("+" + m.DialCode + ex); err == nil {
				return p, true
			}
		}
	}
	return nil, false
}

// ExampleNumberForType returns a valid example number of a given range, or
// nil when the region does not define that range.
func ExampleNumberForType(region string, t PhoneType) (*Phone, bool) {
	m, ok := countries.Data[normalizeRegion(region)]
	if !ok || t >= countries.NumDescs {
		return nil, false
	}
	ex := m.Descs[t].Example
	if ex == "" {
		return nil, false
	}
	p, err := Parse("+" + m.DialCode + ex)
	if err != nil {
		return nil, false
	}
	return p, true
}
