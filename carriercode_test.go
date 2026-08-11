package phonex

import "testing"

// TestCarrierSelectionCode covers the numbers people dial with a carrier
// selection code in front, which several countries use to choose the network
// that carries a long-distance call. The code is not part of the number and
// must be separated from it.
func TestCarrierSelectionCode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		region  string
		e164    string
		carrier string
	}{
		{
			// Brazil dials 0 + carrier + area + number.
			name:    "brazil with carrier",
			input:   "031 11 91234 5678",
			region:  "BR",
			e164:    "+5511912345678",
			carrier: "31",
		},
		{
			name:    "brazil without carrier",
			input:   "011 91234 5678",
			region:  "BR",
			e164:    "+5511912345678",
			carrier: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Parse(tt.input, WithDefaultCountry(tt.region))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := p.E164(); got != tt.e164 {
				t.Errorf("E164() = %q, want %q", got, tt.e164)
			}
			if got := p.CarrierCode(); got != tt.carrier {
				t.Errorf("CarrierCode() = %q, want %q", got, tt.carrier)
			}
			if !p.IsValid() {
				t.Error("IsValid() = false")
			}
		})
	}
}

// TestNationalWithCarrier checks the formatting counterpart: putting a
// carrier code back in front of a number, the way the region writes it.
func TestNationalWithCarrier(t *testing.T) {
	p, err := Parse("+55 11 91234 5678")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	plain := p.National()
	withCode := p.NationalWithCarrier("31")
	if withCode == plain {
		t.Errorf("NationalWithCarrier() = %q, unchanged from National()", withCode)
	}
	if got, want := withCode, "0 31 (11) 91234-5678"; got != want {
		t.Errorf("NationalWithCarrier() = %q, want %q", got, want)
	}
	// An empty carrier code falls back to the plain national format.
	if got := p.NationalWithCarrier(""); got != plain {
		t.Errorf("NationalWithCarrier(\"\") = %q, want %q", got, plain)
	}
	if got := (*Phone)(nil).NationalWithCarrier("31"); got != "" {
		t.Errorf("NationalWithCarrier on nil = %q", got)
	}
}

// TestNationalPrefixTransformRule covers regions whose trunk prefix rule
// rewrites the number rather than only stripping digits. The Bahamas expands
// a local seven-digit number to its full form.
func TestNationalPrefixTransformRule(t *testing.T) {
	p, err := Parse("302 1234", WithDefaultCountry("BS"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := p.E164(), "+12423021234"; got != want {
		t.Errorf("E164() = %q, want %q", got, want)
	}
	if got, want := p.Country(), "BS"; got != want {
		t.Errorf("Country() = %q, want %q", got, want)
	}
	if !p.IsValid() {
		t.Error("IsValid() = false")
	}
}

// TestParseWith covers the allocation-free entry point.
func TestParseWith(t *testing.T) {
	opts := DefaultParseOptions()
	opts.DefaultCountry = "GB"
	opts.KeepRawInput = false

	p, err := ParseWith("020 7031 3000", opts)
	if err != nil {
		t.Fatalf("ParseWith: %v", err)
	}
	if got, want := p.E164(), "+442070313000"; got != want {
		t.Errorf("E164() = %q, want %q", got, want)
	}
	if p.RawInput() != "" {
		t.Errorf("RawInput() = %q, want empty", p.RawInput())
	}
	if _, err := ParseWith("nonsense", opts); err == nil {
		t.Error("ParseWith should reject an unparsable number")
	}
}
