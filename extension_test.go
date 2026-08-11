package phonex

import "testing"

func TestParseExtension(t *testing.T) {
	tests := []struct {
		input string
		e164  string
		ext   string
	}{
		{"+1 202 555 0123 ext. 4321", "+12025550123", "4321"},
		{"+1 202 555 0123 x4321", "+12025550123", "4321"},
		{"+1 202 555 0123 ext 4321", "+12025550123", "4321"},
		{"+1 202 555 0123 extension 4321", "+12025550123", "4321"},
		{"+1 202 555 0123#4321", "+12025550123", "4321"},
		{"tel:+1-202-555-0123;ext=4321", "+12025550123", "4321"},
		{"+1 202 555 0123", "+12025550123", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := p.E164(); got != tt.e164 {
				t.Errorf("E164() = %q, want %q", got, tt.e164)
			}
			if got := p.Extension(); got != tt.ext {
				t.Errorf("Extension() = %q, want %q", got, tt.ext)
			}
			if got := p.HasExtension(); got != (tt.ext != "") {
				t.Errorf("HasExtension() = %t", got)
			}
		})
	}
}

// TestExtensionMarkerNeedsWordBoundary guards the case that makes a naive
// scanner strip real digits: a vanity number whose letters happen to contain
// an extension marker.
func TestExtensionMarkerNeedsWordBoundary(t *testing.T) {
	p, err := Parse("1-800-PAINT-11", WithDefaultCountry("US"), WithAlphaCharacters())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.HasExtension() {
		t.Errorf("Extension() = %q, want none", p.Extension())
	}
	if got, want := p.E164(), "+18007246811"; got != want {
		t.Errorf("E164() = %q, want %q", got, want)
	}
}

func TestPhoneContext(t *testing.T) {
	p, err := Parse("tel:2025550123;phone-context=+1")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := p.E164(), "+12025550123"; got != want {
		t.Errorf("E164() = %q, want %q", got, want)
	}
}

func TestStringIncludesExtension(t *testing.T) {
	p, err := Parse("+1 202 555 0123 ext 7")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := p.String(), "+12025550123;ext=7"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
