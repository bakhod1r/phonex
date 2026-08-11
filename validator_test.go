package phonex

import "testing"

func TestTypeResolution(t *testing.T) {
	tests := []struct {
		number string
		want   PhoneType
	}{
		{"+998 90 123 45 67", Mobile},
		{"+44 20 7031 3000", FixedLine},
		{"+44 7400 123456", Mobile},
		{"+1 202 555 0123", FixedLineOrMobile},
		{"+1 800 555 0199", TollFree},
		{"+44 900 123 4567", PremiumRate},
		{"+800 1234 5678", TollFree},
	}
	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			p, err := Parse(tt.number)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := p.Type(); got != tt.want {
				t.Errorf("Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTypePredicates(t *testing.T) {
	mobile, err := Parse("+44 7400 123456")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !mobile.IsMobile() || mobile.IsLandline() || mobile.IsTollFree() {
		t.Errorf("predicates disagree with Type() = %v", mobile.Type())
	}

	landline, err := Parse("+44 20 7031 3000")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !landline.IsLandline() || landline.IsMobile() {
		t.Errorf("predicates disagree with Type() = %v", landline.Type())
	}
}

// TestValidVersusPossible pins down the distinction the two checks make: a
// number can have a plausible shape without belonging to an assigned range.
func TestValidVersusPossible(t *testing.T) {
	tests := []struct {
		number   string
		region   string
		possible bool
		valid    bool
	}{
		{"+998 90 123 45 67", "", true, true},
		{"+1 202 555 0123", "", true, true},
		{"+1 000 000 0000", "", true, false},
		{"+44 2070 31300", "", true, false},
		{"+998 90 123", "", false, false},
		{"+44 20 7031 3000 999", "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			var opts []ParseOption
			if tt.region != "" {
				opts = append(opts, WithDefaultCountry(tt.region))
			}
			p, err := Parse(tt.number, opts...)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := p.IsPossible(); got != tt.possible {
				t.Errorf("IsPossible() = %t, want %t", got, tt.possible)
			}
			if got := p.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %t, want %t", got, tt.valid)
			}
		})
	}
}

func TestPossibility(t *testing.T) {
	tests := []struct {
		number string
		want   Possibility
	}{
		{"+44 2070 313000", IsPossibleNumber},
		// GB keeps short local-only numbers, which are possible to dial
		// from within the same area but are not valid on their own.
		{"+44 20 703", IsPossibleLocalOnly},
		{"+44 20", TooShort},
		{"+44 20 7031 3000 999", TooLong},
		// Andorra allows 6, 8 or 9 digits, so 7 falls between two of them.
		{"+376 2222222", InvalidLength},
	}
	for _, tt := range tests {
		p := new(Phone)
		if err := p.Parse(tt.number); err != nil {
			t.Errorf("Parse(%q) = %v, want it to parse", tt.number, err)
			continue
		}
		if got := p.Possibility(); got != tt.want {
			t.Errorf("Possibility(%q) = %v, want %v", tt.number, got, tt.want)
		}
	}
}

func TestIsValidForRegion(t *testing.T) {
	// +1 242 numbers are Bahamian, not American, even though both regions
	// share the calling code.
	p, err := Parse("+1 242 302 1234")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !p.IsValidForRegion("BS") {
		t.Error("IsValidForRegion(BS) = false")
	}
	if p.IsValidForRegion("US") {
		t.Error("IsValidForRegion(US) = true for a Bahamian number")
	}
	if p.IsValidForRegion("GB") {
		t.Error("IsValidForRegion(GB) = true for a number outside +44")
	}
}

// TestRegionResolvedFromNationalNumber covers a number typed nationally in a
// region that shares its calling code with another: the national number, not
// the default region, decides which one it belongs to.
func TestRegionResolvedFromNationalNumber(t *testing.T) {
	p, err := Parse("242 302 1234", WithDefaultCountry("US"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := p.Country(); got != "BS" {
		t.Errorf("Country() = %q, want BS", got)
	}
	if !p.IsValid() {
		t.Error("IsValid() = false")
	}

	q, err := Parse("415 555 2671", WithDefaultCountry("US"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := q.Country(); got != "US" {
		t.Errorf("Country() = %q, want US", got)
	}
}

func TestCanBeInternationallyDialled(t *testing.T) {
	p, err := Parse("+1 202 555 0123")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !p.CanBeInternationallyDialled() {
		t.Error("CanBeInternationallyDialled() = false for a normal US number")
	}
}

func TestPackageLevelChecks(t *testing.T) {
	if !IsValid("+998901234567") {
		t.Error("IsValid(+998901234567) = false")
	}
	if IsValid("+998901") {
		t.Error("IsValid(+998901) = true")
	}
	if !IsValid("901234567", WithDefaultCountry("UZ")) {
		t.Error("IsValid with default region = false")
	}
	if !IsPossible("+12025550123") {
		t.Error("IsPossible(+12025550123) = false")
	}
}
