package phonex

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhod1r/phonex/countries"
)

// TestExampleNumbersAreValid checks the whole library against libphonenumber's
// own example numbers: every example must parse, must be reported valid, and
// must resolve to the range it was listed under.
func TestExampleNumbersAreValid(t *testing.T) {
	for _, region := range SupportedRegions() {
		m, _ := Country(region)
		for typ := PhoneType(0); typ < countries.NumDescs; typ++ {
			example := m.Descs[typ].Example
			if example == "" {
				continue
			}
			t.Run(region+"/"+typ.String(), func(t *testing.T) {
				p, err := Parse("+" + m.DialCode + example)
				if err != nil {
					t.Fatalf("Parse(+%s%s) failed: %v", m.DialCode, example, err)
				}
				if !p.IsValid() {
					t.Fatalf("+%s%s reported invalid", m.DialCode, example)
				}
				if got := p.Type(); got != typ && got != FixedLineOrMobile {
					t.Errorf("Type() = %v, want %v", got, typ)
				}
			})
		}
	}
}

// TestNationalFormatRoundTrip checks that a number formatted for its own
// country parses back to the same number. This exercises the formatting rules
// and the trunk-prefix stripping against each other.
func TestNationalFormatRoundTrip(t *testing.T) {
	for _, region := range SupportedRegions() {
		m, _ := Country(region)
		p, ok := ExampleNumber(region)
		if !ok {
			continue
		}
		t.Run(region, func(t *testing.T) {
			national := p.National()
			back, err := Parse(national, WithDefaultCountry(region))
			if err != nil {
				t.Fatalf("Parse(%q, %s) failed: %v", national, region, err)
			}
			if back.E164() != p.E164() {
				t.Errorf("round trip through %q gave %s, want %s", national, back.E164(), p.E164())
			}
			_ = m
		})
	}
}

// TestInternationalFormatRoundTrip checks the international rendering parses
// back unchanged, which is the property callers rely on when they display a
// number and read it back.
func TestInternationalFormatRoundTrip(t *testing.T) {
	for _, region := range SupportedRegions() {
		p, ok := ExampleNumber(region)
		if !ok {
			continue
		}
		t.Run(region, func(t *testing.T) {
			for _, s := range []string{p.International(), p.E164(), p.RFC3966()} {
				back, err := Parse(s)
				if err != nil {
					t.Fatalf("Parse(%q) failed: %v", s, err)
				}
				if back.E164() != p.E164() {
					t.Errorf("Parse(%q) = %s, want %s", s, back.E164(), p.E164())
				}
			}
		})
	}
}

// TestValidNumbersRoundTripStably checks the identity callers depend on when
// they store a number: E.164 in, the same E.164 out. It runs over pseudo-random
// numbers in every region's shape rather than only the metadata examples, so it
// exercises ranges the examples never reach.
//
// The guarantee holds for valid numbers only. An invalid one cannot say where
// its digits end and a trunk prefix begins, so it may normalise differently.
func TestValidNumbersRoundTripStably(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	checked := 0
	for _, region := range SupportedRegions() {
		m, _ := Country(region)
		lengths := m.General.Lengths
		if len(lengths) == 0 {
			continue
		}
		for i := 0; i < 400; i++ {
			var b strings.Builder
			b.WriteByte('+')
			b.WriteString(m.DialCode)
			for j, n := 0, int(lengths[rnd.Intn(len(lengths))]); j < n; j++ {
				b.WriteByte(byte('0' + rnd.Intn(10)))
			}
			p, err := Parse(b.String())
			if err != nil || !p.IsValid() {
				continue
			}
			checked++
			back, err := Parse(p.E164())
			if err != nil {
				t.Fatalf("%s: re-parsing %q failed: %v", region, p.E164(), err)
			}
			if back.E164() != p.E164() {
				t.Errorf("%s: %q re-parsed as %q", region, p.E164(), back.E164())
			}
		}
	}
	if checked < 5000 {
		t.Fatalf("only %d valid numbers generated", checked)
	}
	t.Logf("%d valid numbers round-tripped", checked)
}

// TestMetadataInvariants guards the generated table against the mistakes a
// bad regeneration would introduce.
func TestMetadataInvariants(t *testing.T) {
	for _, region := range SupportedRegions() {
		m, _ := Country(region)
		switch {
		case m.ISO2 != region:
			t.Errorf("%s: ISO2 = %q", region, m.ISO2)
		case m.DialCode == "":
			t.Errorf("%s: empty dial code", region)
		case m.MinLength <= 0 || m.MaxLength < m.MinLength:
			t.Errorf("%s: length range [%d,%d]", region, m.MinLength, m.MaxLength)
		case m.General.Pattern.Empty():
			t.Errorf("%s: no general pattern", region)
		}

		regions := RegionsForDialCode(m.DialCode)
		if len(regions) == 0 {
			t.Errorf("%s: dial code %q not indexed", region, m.DialCode)
			continue
		}
		if !regions[0].IsMainCountry {
			t.Errorf("dial code %q: first region %s is not the main one", m.DialCode, regions[0].ISO2)
		}
		found := false
		for _, r := range regions {
			if r == m {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: missing from the index for dial code %q", region, m.DialCode)
		}
	}
}

// TestPatternsCompile forces every lazily compiled pattern in the metadata,
// so that a bad regeneration fails here rather than at a user's first parse.
func TestPatternsCompile(t *testing.T) {
	all := append(Countries(), NonGeoEntities()...)
	for _, m := range all {
		m.General.Pattern.Regexp()
		m.NoIntlDialling.Pattern.Regexp()
		m.LeadingDigits.Regexp()
		m.InternationalPrefix.Regexp()
		m.NationalPrefixForParsing.Regexp()
		for i := range m.Descs {
			m.Descs[i].Pattern.Regexp()
		}
		for i := range m.Formats {
			m.Formats[i].Pattern.Regexp()
			m.Formats[i].LeadingDigits.Regexp()
		}
	}
}

// TestRegionDetailsArePresent guards the details the JSON side of the
// metadata supplies, which the XML does not carry.
func TestRegionDetailsArePresent(t *testing.T) {
	for _, m := range Countries() {
		if m.ISO3 == "" || m.ISO3 == m.ISO2 {
			t.Errorf("%s: missing ISO-3166 alpha-3 code", m.ISO2)
		}
		if m.Name == "" || m.Name == m.ISO2 {
			t.Errorf("%s: missing country name", m.ISO2)
		}
	}
}

// TestTimezonesAreEmpty records that the bundled metadata carries no time
// zones. It is here so that populating them is a deliberate change with a
// failing test to update, rather than a silent one.
func TestTimezonesAreEmpty(t *testing.T) {
	for _, m := range Countries() {
		if len(m.Timezones) != 0 {
			t.Fatalf("%s now has time zones; update the documentation on "+
				"(*Phone).Timezones and the readme, then drop this test", m.ISO2)
		}
	}
}

func TestNonGeoEntities(t *testing.T) {
	entities := NonGeoEntities()
	if len(entities) == 0 {
		t.Fatal("no non-geographical entities in the metadata")
	}
	p, err := Parse("+800 1234 5678")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !p.IsNonGeographical() {
		t.Error("IsNonGeographical() = false for a +800 number")
	}
	if p.Country() != "001" {
		t.Errorf("Country() = %q, want 001", p.Country())
	}
}

// TestMetadataMatchesVendoredSource confirms the generated table was produced
// from the XML that sits in the tree. A mismatch means someone edited one
// without regenerating the other.
func TestMetadataMatchesVendoredSource(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("internal", "metadata", "PhoneNumberMetadata.xml"))
	if err != nil {
		t.Skipf("vendored metadata not readable: %v", err)
	}
	sum := sha256.Sum256(raw)
	if got := hex.EncodeToString(sum[:]); got != countries.SourceHash {
		t.Fatalf("countries/generated.go was built from a different XML\n"+
			" vendored: %s\n generated: %s\nrun `make generate`", got, countries.SourceHash)
	}
}
