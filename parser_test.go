package phonex

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		region  string
		e164    string
		country string
		source  CountryCodeSource
	}{
		{"international", "+998901234567", "", "+998901234567", "UZ", FromNumberWithPlusSign},
		{"international with spacing", "+998 90 123 45 67", "", "+998901234567", "UZ", FromNumberWithPlusSign},
		{"international without plus", "998901234567", "", "+998901234567", "UZ", FromNumberWithoutPlusSign},
		{"national with default region", "901234567", "UZ", "+998901234567", "UZ", FromDefaultCountry},
		{"calling code repeated nationally", "998901234567", "UZ", "+998901234567", "UZ", FromNumberWithoutPlusSign},
		{"lowercase region", "901234567", "uz", "+998901234567", "UZ", FromDefaultCountry},

		{"us punctuation", "(202) 555-0123", "US", "+12025550123", "US", FromDefaultCountry},
		{"us trunk prefix", "1 202 555 0123", "US", "+12025550123", "US", FromNumberWithoutPlusSign},
		{"gb trunk prefix stripped", "020 7031 3000", "GB", "+442070313000", "GB", FromDefaultCountry},
		{"au trunk prefix stripped", "0412 345 678", "AU", "+61412345678", "AU", FromDefaultCountry},
		{"ru trunk prefix stripped", "8 495 123 45 67", "RU", "+74951234567", "RU", FromDefaultCountry},

		{"idd from us", "011 44 20 7031 3000", "US", "+442070313000", "GB", FromNumberWithIDD},
		{"idd from gb", "00 1 202 555 0123", "GB", "+12025550123", "US", FromNumberWithIDD},

		{"rfc3966", "tel:+1-202-555-0123", "", "+12025550123", "US", FromNumberWithPlusSign},
		{"rfc3966 phone context", "tel:2025550123;phone-context=+1", "", "+12025550123", "US", FromNumberWithPlusSign},

		{"shared calling code picks region", "+1 242 302 1234", "", "+12423021234", "BS", FromNumberWithPlusSign},
		{"non geographical", "+800 1234 5678", "", "+80012345678", "001", FromNumberWithPlusSign},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []ParseOption
			if tt.region != "" {
				opts = append(opts, WithDefaultCountry(tt.region))
			}
			p, err := Parse(tt.input, opts...)
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tt.input, err)
			}
			if got := p.E164(); got != tt.e164 {
				t.Errorf("E164() = %q, want %q", got, tt.e164)
			}
			if got := p.Country(); got != tt.country {
				t.Errorf("Country() = %q, want %q", got, tt.country)
			}
			if got := p.Source(); got != tt.source {
				t.Errorf("Source() = %v, want %v", got, tt.source)
			}
			if !p.IsValid() {
				t.Errorf("IsValid() = false for %q", tt.input)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		region string
		want   error
	}{
		{"empty", "", "", ErrTooShort},
		{"punctuation only", "()- ", "", ErrTooShort},
		{"plus only", "+", "", ErrTooShort},
		{"unknown calling code", "+9991234567890", "", ErrInvalidCountryCode},
		{"leading zero after plus", "+0123456789", "", ErrInvalidCountryCode},
		{"no region to infer", "123", "", ErrMissingCountry},
		{"unknown default region", "901234567", "ZZ", ErrInvalidCountry},
		{"below the E.164 minimum", "+9989", "", ErrTooShort},
		{"above the E.164 maximum", "+998" + "901234567890123456", "", ErrTooLong},
		{"letters without alpha option", "+1 800 CALL NOW", "", ErrInvalidCharacters},
		{"input beyond limit", string(make([]byte, maxRawLen+1)), "", ErrTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []ParseOption
			if tt.region != "" {
				opts = append(opts, WithDefaultCountry(tt.region))
			}
			_, err := Parse(tt.input, opts...)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Parse(%q) error = %v, want %v", tt.input, err, tt.want)
			}
		})
	}
}

func TestParseAlphaCharacters(t *testing.T) {
	p, err := Parse("1-800-FLOWERS", WithDefaultCountry("US"), WithAlphaCharacters())
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if got, want := p.E164(), "+18003569377"; got != want {
		t.Errorf("E164() = %q, want %q", got, want)
	}
	if !p.IsTollFree() {
		t.Errorf("Type() = %v, want TOLL_FREE", p.Type())
	}
}

func TestParseReusesReceiver(t *testing.T) {
	var p Phone
	if err := p.Parse("+998901234567"); err != nil {
		t.Fatalf("first Parse: %v", err)
	}
	if err := p.Parse("+12025550123"); err != nil {
		t.Fatalf("second Parse: %v", err)
	}
	if got, want := p.E164(), "+12025550123"; got != want {
		t.Errorf("E164() = %q, want %q", got, want)
	}
	if p.HasExtension() {
		t.Error("extension carried over from the previous number")
	}
	if p.Country() != "US" {
		t.Errorf("Country() = %q, want US", p.Country())
	}
}

func TestParseKeepsRawInput(t *testing.T) {
	const raw = "+998 (90) 123-45-67"
	p, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.RawInput() != raw {
		t.Errorf("RawInput() = %q, want %q", p.RawInput(), raw)
	}

	p, err = Parse(raw, WithoutRawInput())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.RawInput() != "" {
		t.Errorf("RawInput() = %q, want empty", p.RawInput())
	}
}

func TestParseBytes(t *testing.T) {
	in := []byte("+998901234567")
	p, err := ParseBytes(in)
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	// Mutating the caller's slice must not disturb the parsed number.
	for i := range in {
		in[i] = 'x'
	}
	if got, want := p.E164(), "+998901234567"; got != want {
		t.Errorf("E164() = %q, want %q", got, want)
	}
}

func TestParseDoesNotAllocate(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector allocates on its own, so the count is meaningless")
	}
	var p Phone
	got := testing.AllocsPerRun(200, func() {
		if err := p.Parse("+14155552671", WithoutRawInput()); err != nil {
			t.Fatal(err)
		}
	})
	if got != 0 {
		t.Errorf("Parse allocated %v times per run, want 0", got)
	}
}

// TestParseAcceptsWrongLengthForRegion pins the division of labour between
// Parse and the validity checks: a number whose length is wrong for its own
// country still parses, so callers can report why rather than just "invalid".
func TestParseAcceptsWrongLengthForRegion(t *testing.T) {
	p, err := Parse("+9989012")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.IsValid() {
		t.Error("IsValid() = true for a number too short for UZ")
	}
	if got := p.Possibility(); got != TooShort {
		t.Errorf("Possibility() = %v, want TOO_SHORT", got)
	}
}

func TestNSNAccessors(t *testing.T) {
	p, err := Parse("+44 20 7031 3000")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := p.NSN(), "2070313000"; got != want {
		t.Errorf("NSN() = %q, want %q", got, want)
	}
	if got, want := p.Digits(), "442070313000"; got != want {
		t.Errorf("Digits() = %q, want %q", got, want)
	}
	if got, want := p.DialCode(), "44"; got != want {
		t.Errorf("DialCode() = %q, want %q", got, want)
	}
	if got, want := p.CountryCode(), "+44"; got != want {
		t.Errorf("CountryCode() = %q, want %q", got, want)
	}
	if got, want := p.ISO3(), "GBR"; got != want {
		t.Errorf("ISO3() = %q, want %q", got, want)
	}
}

func TestNilPhoneAccessorsAreSafe(t *testing.T) {
	var p *Phone
	if p.E164() != "" || p.National() != "" || p.International() != "" || p.RFC3966() != "" {
		t.Error("formatting a nil Phone should yield empty strings")
	}
	if p.IsValid() || p.IsPossible() || p.HasExtension() {
		t.Error("predicates on a nil Phone should be false")
	}
	if p.Type() != Unknown {
		t.Errorf("Type() = %v, want UNKNOWN", p.Type())
	}
}
