package phonex

import "testing"

func TestMatch(t *testing.T) {
	tests := []struct {
		name   string
		a, b   string
		region string
		want   MatchType
	}{
		{"identical", "+998901234567", "+998901234567", "", ExactMatch},
		{"different spelling", "+1 202 555 0123", "+1(202)555-0123", "", ExactMatch},
		{"national against international", "901234567", "+998901234567", "UZ", ExactMatch},
		{"different numbers", "+998901234567", "+998901234568", "", NoMatch},
		{"different countries", "+12025550123", "+442070313000", "", NoMatch},
		{"same digits different extension", "+12025550123 ext 1", "+12025550123 ext 2", "", NoMatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []ParseOption
			if tt.region != "" {
				opts = append(opts, WithDefaultCountry(tt.region))
			}
			if got := MatchNumbers(tt.a, tt.b, opts...); got != tt.want {
				t.Errorf("MatchNumbers(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestMatchWithoutCallingCode covers the case the plain string comparison a
// previous version used got wrong: a number typed without its calling code
// against the same number in international form.
func TestMatchShortNSN(t *testing.T) {
	a, err := Parse("+1 650 253 0000")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	b, err := Parse("6502530000", WithDefaultCountry("US"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := a.Match(b); got != ExactMatch {
		t.Errorf("Match() = %v, want EXACT_MATCH", got)
	}
}

func TestEqual(t *testing.T) {
	if !Equal("+998 90 123 45 67", "+998901234567") {
		t.Error("Equal() = false for the same number written differently")
	}
	if Equal("+998901234567", "+998901234568") {
		t.Error("Equal() = true for different numbers")
	}
	if Equal("not a number", "+998901234567") {
		t.Error("Equal() = true when one side fails to parse")
	}
}

func TestEqualExact(t *testing.T) {
	a, err := Parse("+998901234567")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	b, err := Parse("+998 90 123 45 67")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !a.Equal(b) {
		t.Error("Equal() = false")
	}
	if a.EqualExact(b) {
		t.Error("EqualExact() = true for different spellings")
	}
	if !a.EqualExact(a.Clone()) {
		t.Error("EqualExact() = false for a clone")
	}
}

func TestMatchNilSafe(t *testing.T) {
	var p *Phone
	if p.Match(nil) != NoMatch {
		t.Error("Match(nil) should be NO_MATCH")
	}
	q, err := Parse("+998901234567")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if q.Match(nil) != NoMatch || p.Match(q) != NoMatch {
		t.Error("matching against nil should be NO_MATCH")
	}
}
