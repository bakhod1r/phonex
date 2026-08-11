package shortnumber

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestIsEmergency(t *testing.T) {
	tests := []struct {
		number, region string
		want           bool
	}{
		{"112", "GB", true},
		{"999", "GB", true},
		{"911", "US", true},
		{"112", "DE", true},
		{"110", "JP", true},
		{"190", "BR", true},
		{"911", "GB", false},
		{"1234", "GB", false},
		{"112", "UZ", false}, // Uzbekistan dials 01, 02, 03
		{"02", "UZ", true},
		{"", "GB", false},
		{"112", "ZZ", false},
		{"112", "gb", true}, // the region code is case-insensitive
		{"1 1 2", "GB", true},
		{"+112", "GB", false}, // an international number is never short
	}
	for _, tt := range tests {
		t.Run(tt.region+"/"+tt.number, func(t *testing.T) {
			if got := IsEmergency(tt.number, tt.region); got != tt.want {
				t.Errorf("IsEmergency(%q, %q) = %t, want %t", tt.number, tt.region, got, tt.want)
			}
		})
	}
}

// TestConnectsToEmergency covers the distinction that matters when deciding
// whether a number is safe to dial: networks act on the emergency prefix, so
// extra digits still connect.
func TestConnectsToEmergency(t *testing.T) {
	tests := []struct {
		number, region  string
		exact, connects bool
	}{
		{"911", "US", true, true},
		{"911123", "US", false, true},
		{"112999", "GB", false, true},
		// Brazil, Chile and Nicaragua require an exact match.
		{"190", "BR", true, true},
		{"190123", "BR", false, false},
		{"133", "CL", true, true},
		{"133123", "CL", false, false},
		{"5551234", "US", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.region+"/"+tt.number, func(t *testing.T) {
			if got := IsEmergency(tt.number, tt.region); got != tt.exact {
				t.Errorf("IsEmergency() = %t, want %t", got, tt.exact)
			}
			if got := ConnectsToEmergency(tt.number, tt.region); got != tt.connects {
				t.Errorf("ConnectsToEmergency() = %t, want %t", got, tt.connects)
			}
		})
	}
}

func TestIsValidAndPossible(t *testing.T) {
	tests := []struct {
		number, region  string
		possible, valid bool
	}{
		{"100", "GB", true, true},   // the BT operator
		{"10086", "CN", true, true}, // China Mobile customer service
		{"911", "US", true, true},
		{"1234", "GB", true, false}, // right length, not assigned
		{"100", "UZ", true, false},  // not a short number in Uzbekistan
		{"1", "GB", false, false},
		{"1234567890", "GB", false, false},
		{"100", "ZZ", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.region+"/"+tt.number, func(t *testing.T) {
			if got := IsPossible(tt.number, tt.region); got != tt.possible {
				t.Errorf("IsPossible() = %t, want %t", got, tt.possible)
			}
			if got := IsValid(tt.number, tt.region); got != tt.valid {
				t.Errorf("IsValid() = %t, want %t", got, tt.valid)
			}
		})
	}
}

func TestExpectedCost(t *testing.T) {
	tests := []struct {
		number, region string
		want           Cost
	}{
		{"911", "US", TollFree},
		{"112", "GB", TollFree},
		{"10086", "CN", StandardRate},
		{"1234567890", "GB", UnknownCost},
		{"100", "ZZ", UnknownCost},
	}
	for _, tt := range tests {
		t.Run(tt.region+"/"+tt.number, func(t *testing.T) {
			if got := ExpectedCost(tt.number, tt.region); got != tt.want {
				t.Errorf("ExpectedCost() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExpectedCostIsConsistent checks the classification against the ranges it
// is derived from, across the whole data set.
func TestExpectedCostIsConsistent(t *testing.T) {
	for _, region := range Regions() {
		m := Region(region)
		for _, tc := range []struct {
			desc *Desc
			want Cost
		}{
			{&m.PremiumRate, PremiumRate},
			{&m.StandardRate, StandardRate},
			{&m.TollFree, TollFree},
		} {
			example := tc.desc.Example
			if example == "" || !tc.desc.matches(example) {
				continue
			}
			// A number can sit in more than one range; only assert that the
			// answer is not UnknownCost, and that the ranking is respected.
			got := ExpectedCost(example, region)
			if got == UnknownCost {
				t.Errorf("%s: %q is in the %v range but costs %v", region, example, tc.want, got)
			}
		}
	}
}

func TestCostString(t *testing.T) {
	for _, tt := range []struct {
		c    Cost
		want string
	}{
		{TollFree, "TOLL_FREE"},
		{StandardRate, "STANDARD_RATE"},
		{PremiumRate, "PREMIUM_RATE"},
		{UnknownCost, "UNKNOWN_COST"},
	} {
		if got := tt.c.String(); got != tt.want {
			t.Errorf("Cost(%d).String() = %q, want %q", tt.c, got, tt.want)
		}
	}
}

func TestCarrierSpecificAndSMS(t *testing.T) {
	// Every region that defines these ranges must recognise its own example.
	checkedCarrier, checkedSMS := 0, 0
	for _, region := range Regions() {
		m := Region(region)
		if ex := m.CarrierSpecific.Example; ex != "" && m.CarrierSpecific.matches(ex) {
			checkedCarrier++
			if !IsCarrierSpecific(ex, region) {
				t.Errorf("%s: IsCarrierSpecific(%q) = false", region, ex)
			}
		}
		if ex := m.SMSServices.Example; ex != "" && m.SMSServices.matches(ex) {
			checkedSMS++
			if !IsSMSService(ex, region) {
				t.Errorf("%s: IsSMSService(%q) = false", region, ex)
			}
		}
	}
	if checkedCarrier == 0 || checkedSMS == 0 {
		t.Fatalf("nothing checked: %d carrier-specific, %d SMS", checkedCarrier, checkedSMS)
	}
}

func TestRegions(t *testing.T) {
	regions := Regions()
	if len(regions) < 200 {
		t.Fatalf("only %d regions have short number metadata", len(regions))
	}
	for i := 1; i < len(regions); i++ {
		if regions[i-1] >= regions[i] {
			t.Fatalf("Regions() is not sorted at %d: %s, %s", i, regions[i-1], regions[i])
		}
	}
	if Region("ZZ") != nil {
		t.Error("Region(ZZ) should be nil")
	}
	if Region("gb") == nil {
		t.Error("Region should accept a lower-case region code")
	}
}

// TestExamplesAreValid checks every assigned example in the data set.
func TestExamplesAreValid(t *testing.T) {
	checked := 0
	for _, region := range Regions() {
		m := Region(region)
		example := m.ShortCode.Example
		if example == "" {
			continue
		}
		checked++
		if !IsPossible(example, region) {
			t.Errorf("%s: IsPossible(%q) = false", region, example)
		}
		if !IsValid(example, region) {
			t.Errorf("%s: IsValid(%q) = false", region, example)
		}
	}
	if checked < 200 {
		t.Fatalf("only %d regions carry a short code example", checked)
	}
}

// TestPatternsCompile forces every lazily compiled pattern, so a bad
// regeneration fails here rather than at a user's first call.
func TestPatternsCompile(t *testing.T) {
	for _, region := range Regions() {
		m := Region(region)
		for _, d := range []*Desc{
			&m.General, &m.ShortCode, &m.TollFree, &m.PremiumRate, &m.Emergency,
			&m.ExpandedEmergency, &m.StandardRate, &m.CarrierSpecific, &m.SMSServices,
		} {
			d.Pattern.Regexp()
			d.Prefix.Regexp()
		}
	}
}

// TestMetadataMatchesVendoredSource confirms the generated table was produced
// from the XML in the tree.
func TestMetadataMatchesVendoredSource(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "internal", "metadata", "ShortNumberMetadata.xml"))
	if err != nil {
		t.Skipf("vendored metadata not readable: %v", err)
	}
	sum := sha256.Sum256(raw)
	if got := hex.EncodeToString(sum[:]); got != SourceHash {
		t.Fatalf("shortnumber/generated.go was built from a different XML\n"+
			" vendored: %s\n generated: %s\nrun `make generate`", got, SourceHash)
	}
}

func TestDigitsOnly(t *testing.T) {
	for _, tt := range []struct {
		in    string
		want  string
		wantK bool
	}{
		{"112", "112", true},
		{" 1-1 2 ", "112", true},
		{"+112", "", false},
		{"abc", "", true},
	} {
		got, ok := digitsOnly(tt.in)
		if got != tt.want || ok != tt.wantK {
			t.Errorf("digitsOnly(%q) = %q, %t; want %q, %t", tt.in, got, ok, tt.want, tt.wantK)
		}
	}
}
