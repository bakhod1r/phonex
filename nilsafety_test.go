package phonex

import (
	"errors"
	"testing"
)

// TestNilPhoneIsSafe pins the contract every accessor makes: a nil *Phone is
// a legal receiver and yields a zero answer. Callers get one from a failed
// Parse, and the alternative — panicking deep inside a formatter — would turn
// a bad phone number into an outage.
func TestNilPhoneIsSafe(t *testing.T) {
	var p *Phone

	strings := map[string]func() string{
		"NSN":            p.NSN,
		"NationalDigits": p.NationalDigits,
		"Digits":         p.Digits,
		"Extension":      p.Extension,
		"CarrierCode":    p.CarrierCode,
		"RawInput":       p.RawInput,
		"Country":        p.Country,
		"ISO2":           p.ISO2,
		"ISO3":           p.ISO3,
		"CountryName":    p.CountryName,
		"DialCode":       p.DialCode,
		"CountryCode":    p.CountryCode,
		"String":         p.String,
		"E164":           p.E164,
		"International":  p.International,
		"National":       p.National,
		"RFC3966":        p.RFC3966,
	}
	for name, fn := range strings {
		if got := fn(); got != "" {
			t.Errorf("%s() = %q on a nil Phone, want empty", name, got)
		}
	}

	bools := map[string]func() bool{
		"HasExtension":                p.HasExtension,
		"IsNonGeographical":           p.IsNonGeographical,
		"MobileNumberPortable":        p.MobileNumberPortable,
		"IsValid":                     p.IsValid,
		"IsPossible":                  p.IsPossible,
		"CanBeInternationallyDialled": p.CanBeInternationallyDialled,
	}
	for name, fn := range bools {
		if fn() {
			t.Errorf("%s() = true on a nil Phone", name)
		}
	}

	if got := p.Timezones(); got != nil {
		t.Errorf("Timezones() = %v on a nil Phone, want nil", got)
	}
	if got := p.Metadata(); got != nil {
		t.Errorf("Metadata() = %v on a nil Phone, want nil", got)
	}
	if got := p.Clone(); got != nil {
		t.Errorf("Clone() = %v on a nil Phone, want nil", got)
	}
	if got := p.Source(); got != FromDefaultCountry {
		t.Errorf("Source() = %v on a nil Phone", got)
	}
	if got := p.Type(); got != Unknown {
		t.Errorf("Type() = %v on a nil Phone", got)
	}
	if got := p.Possibility(); got != InvalidCountryCode {
		t.Errorf("Possibility() = %v on a nil Phone", got)
	}
	if got := p.PossibilityForType(Mobile); got != InvalidCountryCode {
		t.Errorf("PossibilityForType() = %v on a nil Phone", got)
	}
	if p.IsValidForRegion("GB") {
		t.Error("IsValidForRegion() = true on a nil Phone")
	}
	if got := p.OutOfCountry("US"); got != "" {
		t.Errorf("OutOfCountry() = %q on a nil Phone", got)
	}
	if got := p.Format(FormatNational); got != "" {
		t.Errorf("Format() = %q on a nil Phone", got)
	}
	if got := p.AppendE164(nil); got != nil {
		t.Errorf("AppendE164() = %v on a nil Phone", got)
	}
	if p.EqualExact(nil) || p.Equal(nil) {
		t.Error("comparing two nil Phones should not report equality")
	}
}

// TestFormatCoversEveryFormat walks the FormatType switch, which is the one
// place a new format could be added and silently fall through to E.164.
func TestFormatCoversEveryFormat(t *testing.T) {
	p, err := Parse("+44 20 7031 3000")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	cases := map[FormatType]string{
		FormatE164:          p.E164(),
		FormatInternational: p.International(),
		FormatNational:      p.National(),
		FormatRFC3966:       p.RFC3966(),
	}
	seen := map[string]bool{}
	for f, want := range cases {
		got := p.Format(f)
		if got != want {
			t.Errorf("Format(%d) = %q, want %q", f, got, want)
		}
		if seen[got] {
			t.Errorf("Format(%d) = %q, which another format also produced", f, got)
		}
		seen[got] = true
	}
}

// TestMatchDifferentAreaCodes checks that numbers sharing their subscriber
// digits but not their area code are not treated as the same number.
func TestMatchDifferentAreaCodes(t *testing.T) {
	full, err := Parse("+1 650 253 0000")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// The same subscriber digits with a different area code.
	other, err := Parse("+1 415 253 0000")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := full.Match(other); got != NoMatch {
		t.Errorf("Match() = %v for different area codes, want NO_MATCH", got)
	}

	// Numbers too short to share a meaningful suffix never match.
	a, err := Parse("+998901234567")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := a.Match(nil); got != NoMatch {
		t.Errorf("Match(nil) = %v", got)
	}
}

// TestErrorIsRejectsOtherErrors checks that a ValidationError does not match
// an unrelated error type, which errors.Is relies on.
func TestErrorIsRejectsOtherErrors(t *testing.T) {
	if errors.Is(ErrTooShort, errors.New("too short")) {
		t.Error("a ValidationError should not match a plain error")
	}
	if !errors.Is(ErrTooShort, ErrTooShort) {
		t.Error("a ValidationError should match itself")
	}
}

// TestParseBytesEmpty covers the guard that keeps an empty slice from
// indexing past the end.
func TestParseBytesEmpty(t *testing.T) {
	var p Phone
	if err := p.ParseBytes(nil); err == nil {
		t.Error("ParseBytes(nil) should fail")
	}
	if err := p.ParseBytes([]byte{}); err == nil {
		t.Error("ParseBytes(empty) should fail")
	}
}
