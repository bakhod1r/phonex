package phonex

import "testing"

// TestFormatterEdgeCases covers the paths an ordinary typing sequence never
// reaches: a lone '+', backing out of it, and a national prefix with nothing
// after it yet.
func TestFormatterEdgeCases(t *testing.T) {
	f := NewFormatter("US")
	if got := f.InputDigit('+'); got != "+" {
		t.Errorf("InputDigit('+') = %q, want %q", got, "+")
	}
	if got := f.RemoveLastDigit(); got != "" {
		t.Errorf("RemoveLastDigit() after '+' = %q, want empty", got)
	}
	if f.Digits() != "" {
		t.Errorf("Digits() = %q, want empty", f.Digits())
	}
	// Removing from an empty Formatter must not panic.
	if got := f.RemoveLastDigit(); got != "" {
		t.Errorf("RemoveLastDigit() on empty = %q, want empty", got)
	}

	// A second '+' is not a digit and must be ignored.
	f.Clear()
	f.InputDigit('+')
	if got := f.InputDigit('+'); got != "+" {
		t.Errorf("a repeated '+' changed the output to %q", got)
	}

	// The trunk prefix alone has no national number to group yet.
	f = NewFormatter("UZ")
	if got := f.InputDigit('8'); got != "8" {
		t.Errorf("InputDigit('8') = %q, want %q", got, "8")
	}

	// A calling code that no region uses is echoed rather than grouped.
	f = NewFormatter("")
	if got := f.Input("+9991234"); got != "+9991234" {
		t.Errorf("Input(+9991234) = %q, want it echoed", got)
	}
}

// TestFormatPartialBelowLeadingDigits pins the rule that a format's leading
// digits only start discriminating once enough digits have been typed.
func TestFormatPartialBelowLeadingDigits(t *testing.T) {
	m, ok := Country("UZ")
	if !ok {
		t.Fatal("UZ metadata missing")
	}
	if _, ok := formatPartial(m, "9", "", false); !ok {
		t.Error("a single digit should still find a candidate format")
	}
	if _, ok := formatPartial(m, "", "", false); ok {
		t.Error("no digits should yield no formatting")
	}
}

func TestRelaxPattern(t *testing.T) {
	tests := []struct{ in, want string }{
		{`(\d{3})(\d{4})`, `(\d{3})(\d{4})`},
		{`(9\d{2})(\d{4})`, `(\d\d{2})(\d{4})`},
		{`([2-9]\d)(\d{3})`, `(\d\d)(\d{3})`},
		{`(\d{2,3})`, `(\d{2,3})`},
		// An unbalanced class cannot be relaxed and is returned unchanged.
		{`([0-9`, `([0-9`},
	}
	for _, tt := range tests {
		if got := relaxPattern(tt.in); got != tt.want {
			t.Errorf("relaxPattern(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestUnanchor(t *testing.T) {
	if got, want := unanchor(`^(?:\d{9})$`), `\d{9}`; got != want {
		t.Errorf("unanchor = %q, want %q", got, want)
	}
	// A pattern that was never anchored is left alone.
	if got, want := unanchor(`\d{9}`), `\d{9}`; got != want {
		t.Errorf("unanchor = %q, want %q", got, want)
	}
}

func TestGoTemplate(t *testing.T) {
	tests := []struct{ in, want string }{
		{"$1 $2", "${1} ${2}"},
		{"$1-$2-$3", "${1}-${2}-${3}"},
		{"no groups", "no groups"},
		// A lone '$' is a literal and must be escaped for the expander.
		{"a$b", "a$$b"},
	}
	for _, tt := range tests {
		if got := goTemplate(tt.in); got != tt.want {
			t.Errorf("goTemplate(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestLiteralPrefix(t *testing.T) {
	if got, want := literalPrefix(`^(?:00)$`), "00"; got != want {
		t.Errorf("literalPrefix = %q, want %q", got, want)
	}
	// A pattern offering a choice has no single dialable prefix.
	if got := literalPrefix(`^(?:00|011)$`); got != "" {
		t.Errorf("literalPrefix = %q, want empty", got)
	}
}

func TestFormatUnknownFormatType(t *testing.T) {
	p, err := Parse("+998901234567")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := p.Format(FormatType(99)), p.E164(); got != want {
		t.Errorf("Format(unknown) = %q, want the E.164 form %q", got, want)
	}
}

func TestOutOfCountryUnknownRegion(t *testing.T) {
	p, err := Parse("+998901234567")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := p.OutOfCountry("ZZ"), p.International(); got != want {
		t.Errorf("OutOfCountry(ZZ) = %q, want the international form %q", got, want)
	}
}

func TestPossibilityForType(t *testing.T) {
	p, err := Parse("+442070313000")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := p.PossibilityForType(FixedLine); got != IsPossibleNumber {
		t.Errorf("PossibilityForType(FIXED_LINE) = %v, want IS_POSSIBLE", got)
	}
	if got := p.PossibilityForType(FixedLineOrMobile); got != IsPossibleNumber {
		t.Errorf("PossibilityForType(FIXED_LINE_OR_MOBILE) = %v, want IS_POSSIBLE", got)
	}
	// GB has no shared-cost range, so no length can be possible for it.
	if got := p.PossibilityForType(SharedCost); got != InvalidLength {
		t.Errorf("PossibilityForType(SHARED_COST) = %v, want INVALID_LENGTH", got)
	}
}

func TestStringers(t *testing.T) {
	if Mobile.String() != "MOBILE" || Unknown.String() != "UNKNOWN" {
		t.Error("PhoneType.String() is wrong")
	}
	if FromNumberWithIDD.String() != "FROM_NUMBER_WITH_IDD" {
		t.Error("CountryCodeSource.String() is wrong")
	}
	if TooShort.String() != "TOO_SHORT" || IsPossibleNumber.String() != "IS_POSSIBLE" {
		t.Error("Possibility.String() is wrong")
	}
	if NSNMatch.String() != "NSN_MATCH" || NoMatch.String() != "NO_MATCH" {
		t.Error("MatchType.String() is wrong")
	}
}
