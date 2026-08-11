// Package difftest compares phonex against Google's libphonenumber, through
// the nyaruka/phonenumbers port, on a large generated corpus.
//
// The corpus is built from the metadata's own example numbers plus
// deterministic pseudo-random numbers in each region's shape, so it covers
// both numbers that should be valid and numbers that should not.
package difftest

import (
	"flag"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/bakhod1r/phonex"
	"github.com/bakhod1r/phonex/countries"
	lpn "github.com/nyaruka/phonenumbers"
)

var (
	randomPerRegion = flag.Int("random-per-region", 40, "pseudo-random numbers to generate per region")
	maxReported     = flag.Int("max-reported", 15, "mismatches to print per check before summarising")
	corpusSeed      = flag.Int64("seed", 1, "seed for the pseudo-random corpus")
	strict          = flag.Bool("strict", false, "fail on any disagreement, ignoring the metadata skew budget")
)

// skewBudget is how many disagreements a check may report before the run
// fails.
//
// It is not slack for bugs. phonex is generated from a tagged libphonenumber
// release, while nyaruka/phonenumbers bundles metadata regenerated from a
// snapshot taken at a different moment, so a handful of number ranges and
// formats genuinely differ between the two data sets. Every disagreement is
// printed, so a real regression shows up as new cases rather than as a
// silently absorbed count. Run with -strict to see them all as failures.
const skewBudget = 10

// testCase is one input, in the form a caller would supply it.
type testCase struct {
	input  string
	region string // default region, "" for international input
}

func (c testCase) String() string {
	if c.region == "" {
		return fmt.Sprintf("%q", c.input)
	}
	return fmt.Sprintf("%q (region %s)", c.input, c.region)
}

// corpus builds the shared input set. It is deterministic, so a failure is
// reproducible from the reported case alone.
func corpus(t testing.TB) []testCase {
	t.Helper()

	regions := phonex.SupportedRegions()
	cases := make([]testCase, 0, len(regions)*(*randomPerRegion+12))
	rnd := rand.New(rand.NewSource(*corpusSeed))

	for _, region := range regions {
		m, ok := phonex.Country(region)
		if !ok {
			continue
		}

		// Every example the metadata carries, in international form and,
		// where the region has one, in national form too.
		for typ := phonex.PhoneType(0); typ < countries.NumDescs; typ++ {
			example := m.Descs[typ].Example
			if example == "" {
				continue
			}
			cases = append(cases,
				testCase{input: "+" + m.DialCode + example},
				testCase{input: example, region: region},
			)
			if m.NationalPrefix != "" {
				cases = append(cases, testCase{input: m.NationalPrefix + example, region: region})
			}
		}

		// Numbers of each possible length, with pseudo-random digits. Most
		// are invalid, which is exactly the interesting case: the two
		// implementations must reject the same things.
		lengths := m.General.Lengths
		if len(lengths) == 0 {
			continue
		}
		for i := 0; i < *randomPerRegion; i++ {
			n := int(lengths[rnd.Intn(len(lengths))])
			var b strings.Builder
			b.WriteByte('+')
			b.WriteString(m.DialCode)
			for j := 0; j < n; j++ {
				b.WriteByte(byte('0' + rnd.Intn(10)))
			}
			cases = append(cases, testCase{input: b.String()})
		}
	}
	return cases
}

// parsePair parses one case with both implementations.
type parsePair struct {
	ours   *phonex.Phone
	ourErr error
	theirs *lpn.PhoneNumber
	theirE error
}

func parseBoth(c testCase) parsePair {
	var p parsePair
	if c.region == "" {
		p.ours, p.ourErr = phonex.Parse(c.input)
	} else {
		p.ours, p.ourErr = phonex.Parse(c.input, phonex.WithDefaultCountry(c.region))
	}
	p.theirs, p.theirE = lpn.Parse(c.input, c.region)
	return p
}

// mismatches accumulates disagreements and prints a bounded sample.
type mismatches struct {
	t    *testing.T
	name string
	// budget overrides skewBudget for this check. Zero means use the default.
	budget int
	total  int
	shown  int
	cases  []testCase
}

func (m *mismatches) add(c testCase, format string, args ...any) {
	if len(m.cases) < 200 {
		m.cases = append(m.cases, c)
	}
	m.note(c, format, args...)
}

func (m *mismatches) report(checked int) {
	if m.total == 0 {
		m.t.Logf("%s: %d cases, no disagreement", m.name, checked)
		return
	}
	budget := m.budget
	if budget == 0 {
		budget = skewBudget
	}
	msg := fmt.Sprintf("%s: %d of %d cases disagree (%.3f%%)",
		m.name, m.total, checked, 100*float64(m.total)/float64(checked))
	if !*strict && m.total <= budget {
		m.t.Logf("%s — within the %d-case data skew budget", msg, budget)
		return
	}
	m.t.Errorf("%s", msg)
}

// add records a disagreement. Reporting is deferred to report so that a run
// inside the skew budget does not print failures.
func (m *mismatches) note(c testCase, format string, args ...any) {
	m.total++
	if m.shown < *maxReported {
		m.shown++
		m.t.Logf("%s: %s: %s", m.name, c, fmt.Sprintf(format, args...))
	}
}

// TestParseAgreement checks that both implementations accept and reject the
// same inputs, and agree on the resulting E.164 number and region.
func TestParseAgreement(t *testing.T) {
	cases := corpus(t)
	accept := &mismatches{t: t, name: "accept/reject"}
	e164 := &mismatches{t: t, name: "E164"}
	region := &mismatches{t: t, name: "region"}

	for _, c := range cases {
		p := parseBoth(c)

		if (p.ourErr == nil) != (p.theirE == nil) {
			accept.add(c, "phonex err=%v, libphonenumber err=%v", p.ourErr, p.theirE)
			continue
		}
		if p.ourErr != nil {
			continue
		}

		if got, want := p.ours.E164(), lpn.Format(p.theirs, lpn.E164); got != want {
			e164.add(c, "phonex %q, libphonenumber %q", got, want)
			continue
		}
		if got, want := p.ours.Country(), lpn.GetRegionCodeForNumber(p.theirs); got != want && want != "" {
			region.add(c, "phonex %q, libphonenumber %q (number %s)", got, want, p.ours.E164())
		}
	}

	accept.report(len(cases))
	e164.report(len(cases))
	region.report(len(cases))
}

// TestValidityAgreement checks the validity and possibility verdicts, which
// are what callers gate on.
func TestValidityAgreement(t *testing.T) {
	cases := corpus(t)
	valid := &mismatches{t: t, name: "IsValid"}
	possible := &mismatches{t: t, name: "IsPossible"}
	checked := 0

	for _, c := range cases {
		p := parseBoth(c)
		if p.ourErr != nil || p.theirE != nil {
			continue
		}
		checked++

		if got, want := p.ours.IsValid(), lpn.IsValidNumber(p.theirs); got != want {
			valid.add(c, "phonex %t, libphonenumber %t (number %s, type %v)",
				got, want, p.ours.E164(), p.ours.Type())
			continue
		}
		if got, want := p.ours.IsPossible(), lpn.IsPossibleNumber(p.theirs); got != want {
			possible.add(c, "phonex %t, libphonenumber %t (number %s)", got, want, p.ours.E164())
		}
	}

	valid.report(checked)
	possible.report(checked)
}

// TestTypeAgreement checks the resolved number type.
func TestTypeAgreement(t *testing.T) {
	cases := corpus(t)
	typ := &mismatches{t: t, name: "Type"}
	checked := 0

	for _, c := range cases {
		p := parseBoth(c)
		if p.ourErr != nil || p.theirE != nil {
			continue
		}
		// A type is only meaningful for a number both sides call valid.
		if !p.ours.IsValid() || !lpn.IsValidNumber(p.theirs) {
			continue
		}
		checked++

		if got, want := p.ours.Type(), theirType(lpn.GetNumberType(p.theirs)); got != want {
			typ.add(c, "phonex %v, libphonenumber %v (number %s)", got, want, p.ours.E164())
		}
	}
	typ.report(checked)
}

// TestFormatAgreement checks every output format on numbers both sides call
// valid, which is the population a caller would ever display.
func TestFormatAgreement(t *testing.T) {
	cases := corpus(t)
	checks := []struct {
		name   string
		ours   func(*phonex.Phone) string
		theirs func(*lpn.PhoneNumber) string
		m      *mismatches
	}{
		{
			name:   "International",
			ours:   (*phonex.Phone).International,
			theirs: func(n *lpn.PhoneNumber) string { return lpn.Format(n, lpn.INTERNATIONAL) },
		},
		{
			name:   "National",
			ours:   (*phonex.Phone).National,
			theirs: func(n *lpn.PhoneNumber) string { return lpn.Format(n, lpn.NATIONAL) },
		},
		{
			name:   "RFC3966",
			ours:   (*phonex.Phone).RFC3966,
			theirs: func(n *lpn.PhoneNumber) string { return lpn.Format(n, lpn.RFC3966) },
		},
	}
	for i := range checks {
		checks[i].m = &mismatches{t: t, name: checks[i].name}
	}

	checked := 0
	for _, c := range cases {
		p := parseBoth(c)
		if p.ourErr != nil || p.theirE != nil {
			continue
		}
		if !p.ours.IsValid() || !lpn.IsValidNumber(p.theirs) {
			continue
		}
		checked++
		for _, check := range checks {
			if got, want := check.ours(p.ours), check.theirs(p.theirs); got != want {
				check.m.add(c, "phonex %q, libphonenumber %q", got, want)
			}
		}
	}
	for _, check := range checks {
		check.m.report(checked)
	}
}

// TestOutOfCountryAgreement checks the dialling form from a handful of
// representative origin regions.
func TestOutOfCountryAgreement(t *testing.T) {
	// BR is deliberately absent. Its international prefix is a pattern
	// ("00(?:1[245]|...)") rather than a single literal, and libphonenumber
	// then formats without any prefix at all. nyaruka/phonenumbers tests
	// that pattern with an unanchored match instead of a full one, so it
	// emits the raw regexp as if it were a dialable prefix — a divergence
	// its own test suite notes. Comparing against it would assert the bug.
	origins := []string{"US", "GB", "DE", "JP", "AU", "UZ"}
	m := &mismatches{t: t, name: "OutOfCountry"}
	checked := 0

	for _, region := range phonex.SupportedRegions() {
		p, ok := phonex.ExampleNumber(region)
		if !ok {
			continue
		}
		theirs, err := lpn.Parse(p.E164(), "")
		if err != nil || !lpn.IsValidNumber(theirs) || !p.IsValid() {
			continue
		}
		for _, from := range origins {
			checked++
			got := p.OutOfCountry(from)
			want := lpn.FormatOutOfCountryCallingNumber(theirs, from)
			if got != want {
				m.add(testCase{input: p.E164()}, "from %s: phonex %q, libphonenumber %q", from, got, want)
			}
		}
	}
	m.report(checked)
}

// theirType maps libphonenumber's number type onto ours.
func theirType(t lpn.PhoneNumberType) phonex.PhoneType {
	switch t {
	case lpn.FIXED_LINE:
		return phonex.FixedLine
	case lpn.MOBILE:
		return phonex.Mobile
	case lpn.FIXED_LINE_OR_MOBILE:
		return phonex.FixedLineOrMobile
	case lpn.TOLL_FREE:
		return phonex.TollFree
	case lpn.PREMIUM_RATE:
		return phonex.PremiumRate
	case lpn.SHARED_COST:
		return phonex.SharedCost
	case lpn.VOIP:
		return phonex.VoIP
	case lpn.PERSONAL_NUMBER:
		return phonex.PersonalNumber
	case lpn.PAGER:
		return phonex.Pager
	case lpn.UAN:
		return phonex.UAN
	case lpn.VOICEMAIL:
		return phonex.Voicemail
	default:
		return phonex.Unknown
	}
}

// TestCorpusIsSubstantial guards the comparison itself: a corpus that
// silently shrank would make every other test in this file pass for the
// wrong reason.
func TestCorpusIsSubstantial(t *testing.T) {
	cases := corpus(t)
	if len(cases) < 10000 {
		t.Fatalf("corpus has only %d cases", len(cases))
	}
	regions := map[string]bool{}
	for _, c := range cases {
		if c.region != "" {
			regions[c.region] = true
		}
	}
	if len(regions) < 200 {
		t.Fatalf("corpus covers only %d regions with a default region", len(regions))
	}
	names := make([]string, 0, len(regions))
	for r := range regions {
		names = append(names, r)
	}
	sort.Strings(names)
	t.Logf("corpus: %d cases across %d regions", len(cases), len(names))
}
