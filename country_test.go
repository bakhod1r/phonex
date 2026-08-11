package phonex

import "testing"

func TestCountry(t *testing.T) {
	m, ok := Country("UZ")
	if !ok {
		t.Fatal("Country(UZ) not found")
	}
	if m.DialCode != "998" || m.ISO3 != "UZB" || m.Name != "Uzbekistan" {
		t.Errorf("unexpected metadata: %+v", struct{ Dial, ISO3, Name string }{m.DialCode, m.ISO3, m.Name})
	}
	if _, ok := Country("uz"); !ok {
		t.Error("Country should accept a lower-case region code")
	}
	if _, ok := Country("ZZ"); ok {
		t.Error("Country(ZZ) should not be found")
	}
}

func TestCountryByDialCode(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"998", "UZ"},
		{"+998", "UZ"},
		{"1", "US"},
		{"44", "GB"},
		{"800", "001"},
	}
	for _, tt := range tests {
		m, ok := CountryByDialCode(tt.code)
		if !ok {
			t.Errorf("CountryByDialCode(%q) not found", tt.code)
			continue
		}
		if m.ISO2 != tt.want {
			t.Errorf("CountryByDialCode(%q) = %s, want %s", tt.code, m.ISO2, tt.want)
		}
	}
	if _, ok := CountryByDialCode("999"); ok {
		t.Error("CountryByDialCode(999) should not be found")
	}
}

func TestRegionsForDialCode(t *testing.T) {
	regions := RegionsForDialCode("1")
	if len(regions) < 2 {
		t.Fatalf("+1 should be shared by several regions, got %d", len(regions))
	}
	if regions[0].ISO2 != "US" {
		t.Errorf("first region for +1 = %s, want US", regions[0].ISO2)
	}
}

func TestCountryByPhone(t *testing.T) {
	m, ok := CountryByPhone("+44 20 7031 3000")
	if !ok {
		t.Fatal("CountryByPhone failed")
	}
	if m.ISO2 != "GB" {
		t.Errorf("ISO2 = %s, want GB", m.ISO2)
	}
	if _, ok := CountryByPhone("nonsense"); ok {
		t.Error("CountryByPhone should fail on unparsable input")
	}
}

func TestSearchCountries(t *testing.T) {
	if got := SearchCountries("uz"); len(got) != 1 || got[0].ISO2 != "UZ" {
		t.Errorf("SearchCountries(uz) = %v", isoCodes(got))
	}
	if got := SearchCountries("UZB"); len(got) != 1 || got[0].ISO2 != "UZ" {
		t.Errorf("SearchCountries(UZB) = %v", isoCodes(got))
	}
	if got := SearchCountries("united"); len(got) < 2 {
		t.Errorf("SearchCountries(united) = %v, want several", isoCodes(got))
	}
	if got := SearchCountries(""); got != nil {
		t.Errorf("SearchCountries(\"\") = %v, want nil", isoCodes(got))
	}
}

func TestCountriesAndSupportedRegions(t *testing.T) {
	all := Countries()
	regions := SupportedRegions()
	if len(all) != len(regions) {
		t.Fatalf("Countries() has %d entries, SupportedRegions() has %d", len(all), len(regions))
	}
	if len(all) < 200 {
		t.Fatalf("only %d regions in the metadata", len(all))
	}
	for i := 1; i < len(regions); i++ {
		if regions[i-1] >= regions[i] {
			t.Fatalf("SupportedRegions() is not sorted at %d: %s, %s", i, regions[i-1], regions[i])
		}
	}
}

func TestExampleNumberForType(t *testing.T) {
	p, ok := ExampleNumberForType("GB", Mobile)
	if !ok {
		t.Fatal("no GB mobile example")
	}
	if !p.IsValid() || p.Type() != Mobile {
		t.Errorf("GB mobile example %s is %v, valid=%t", p.E164(), p.Type(), p.IsValid())
	}
	if _, ok := ExampleNumberForType("ZZ", Mobile); ok {
		t.Error("ExampleNumberForType(ZZ) should fail")
	}
}

func TestTimezonesAndPortability(t *testing.T) {
	p, err := Parse("+44 7400 123456")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !p.MobileNumberPortable() {
		t.Error("GB should be reported as a number-portable region")
	}
	// Time zones are absent from the bundled metadata; the accessor must
	// still be safe to call.
	if tz := p.Timezones(); len(tz) != 0 {
		t.Errorf("Timezones() = %v, want empty with the bundled metadata", tz)
	}
}

func isoCodes(ms []*Metadata) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.ISO2
	}
	return out
}
