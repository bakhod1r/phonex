package difftest

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/bakhod1r/phonex/shortnumber"
	lpn "github.com/nyaruka/phonenumbers"
)

// shortCorpus builds short number inputs per region: the example numbers the
// metadata carries, and pseudo-random digit strings of every length the
// region uses, so both assigned and unassigned codes are covered.
func shortCorpus(t testing.TB) []testCase {
	t.Helper()

	regions := shortnumber.Regions()
	cases := make([]testCase, 0, len(regions)*40)
	rnd := rand.New(rand.NewSource(*corpusSeed))

	for _, region := range regions {
		m := shortnumber.Region(region)
		if m == nil {
			continue
		}
		for _, d := range []*shortnumber.Desc{
			&m.ShortCode, &m.TollFree, &m.PremiumRate, &m.Emergency,
			&m.ExpandedEmergency, &m.StandardRate, &m.CarrierSpecific, &m.SMSServices,
		} {
			if d.Example != "" {
				cases = append(cases, testCase{input: d.Example, region: region})
			}
		}
		for _, n := range m.General.Lengths {
			for i := 0; i < 12; i++ {
				var digits []byte
				for j := int32(0); j < n; j++ {
					digits = append(digits, byte('0'+rnd.Intn(10)))
				}
				cases = append(cases, testCase{input: string(digits), region: region})
			}
		}
		// Emergency numbers with digits appended, which is what the
		// "connects to" question is really about.
		if ex := m.Emergency.Example; ex != "" {
			cases = append(cases,
				testCase{input: ex + "1", region: region},
				testCase{input: ex + "23", region: region},
			)
		}
	}
	return cases
}

// TestShortNumberAgreement compares every short number verdict against
// libphonenumber.
func TestShortNumberAgreement(t *testing.T) {
	cases := shortCorpus(t)
	if len(cases) < 8000 {
		t.Fatalf("short number corpus has only %d cases", len(cases))
	}

	possible := &mismatches{t: t, name: "short IsPossible"}
	valid := &mismatches{t: t, name: "short IsValid"}
	emergency := &mismatches{t: t, name: "short IsEmergency"}
	connects := &mismatches{t: t, name: "short ConnectsToEmergency"}
	// ExpectedCost has no counterpart here: nyaruka/phonenumbers does not
	// port libphonenumber's getExpectedCostForRegion, so there is nothing to
	// compare against. It is covered by unit tests in the package itself.

	checked := 0
	for _, c := range cases {
		// The emergency helpers take the dialled digits directly; the others
		// need libphonenumber's parsed form.
		if got, want := shortnumber.IsEmergency(c.input, c.region), lpn.IsEmergencyNumber(c.input, c.region); got != want {
			emergency.add(c, "phonex %t, libphonenumber %t", got, want)
		}
		if got, want := shortnumber.ConnectsToEmergency(c.input, c.region), lpn.ConnectsToEmergencyNumber(c.input, c.region); got != want {
			connects.add(c, "phonex %t, libphonenumber %t", got, want)
		}

		// libphonenumber's short number API takes a parsed PhoneNumber, so
		// the digits have already been through ordinary national-number
		// parsing by the time it sees them. That can rewrite them: Guernsey
		// turns "848918" into "1481848918" via its national prefix transform
		// rule, and a leading "00" is read as an international prefix. A
		// short number is dialled literally, so phonex does not do that, and
		// comparing the two on such inputs would compare different
		// questions. Restrict the comparison to inputs the parse left alone.
		n, err := lpn.Parse(c.input, c.region)
		if err != nil || lpn.GetNationalSignificantNumber(n) != c.input {
			continue
		}
		checked++

		if got, want := shortnumber.IsPossible(c.input, c.region), lpn.IsPossibleShortNumberForRegion(n, c.region); got != want {
			possible.add(c, "phonex %t, libphonenumber %t", got, want)
		}
		if got, want := shortnumber.IsValid(c.input, c.region), lpn.IsValidShortNumberForRegion(n, c.region); got != want {
			valid.add(c, "phonex %t, libphonenumber %t", got, want)
		}
	}

	emergency.report(len(cases))
	connects.report(len(cases))
	possible.report(checked)
	valid.report(checked)
}

// TestEmergencyNumbersAreRecognised is a sanity check on the data itself: the
// emergency number of every region must be reported as one.
func TestEmergencyNumbersAreRecognised(t *testing.T) {
	missing := 0
	for _, region := range shortnumber.Regions() {
		m := shortnumber.Region(region)
		if m == nil || m.Emergency.Example == "" {
			continue
		}
		if !shortnumber.IsEmergency(m.Emergency.Example, region) {
			t.Errorf("%s: %q is not recognised as an emergency number", region, m.Emergency.Example)
			missing++
		}
	}
	if missing == 0 {
		t.Logf("every region's emergency example is recognised")
	}
}

// TestWellKnownEmergencyNumbers spot-checks numbers whose behaviour a reader
// can verify without consulting the metadata.
func TestWellKnownEmergencyNumbers(t *testing.T) {
	for _, tt := range []struct {
		number, region string
		want           bool
	}{
		{"112", "GB", true},
		{"112", "DE", true},
		{"911", "US", true},
		{"999", "GB", true},
		{"110", "JP", true},
		{"190", "BR", true},
		{"1234", "GB", false},
		{"911", "GB", false},
	} {
		name := fmt.Sprintf("%s/%s", tt.region, tt.number)
		t.Run(name, func(t *testing.T) {
			if got := shortnumber.IsEmergency(tt.number, tt.region); got != tt.want {
				t.Errorf("IsEmergency(%q, %q) = %t, want %t", tt.number, tt.region, got, tt.want)
			}
			// libphonenumber must agree, or one of us has it wrong.
			if got, want := shortnumber.IsEmergency(tt.number, tt.region),
				lpn.IsEmergencyNumber(tt.number, tt.region); got != want {
				t.Errorf("phonex %t, libphonenumber %t", got, want)
			}
		})
	}
}
