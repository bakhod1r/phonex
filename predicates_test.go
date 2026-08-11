package phonex

import "testing"

// TestTypePredicates checks that each predicate agrees with Type, using a real
// number from the range where the metadata has one. Regions differ in which
// ranges they define, so the example is looked up rather than hard-coded.
func TestTypePredicatesAgreeWithType(t *testing.T) {
	predicates := []struct {
		typ  PhoneType
		name string
		fn   func(*Phone) bool
	}{
		{FixedLine, "IsLandline", (*Phone).IsLandline},
		{Mobile, "IsMobile", (*Phone).IsMobile},
		{TollFree, "IsTollFree", (*Phone).IsTollFree},
		{PremiumRate, "IsPremiumRate", (*Phone).IsPremiumRate},
		{SharedCost, "IsSharedCost", (*Phone).IsSharedCost},
		{VoIP, "IsVoIP", (*Phone).IsVoIP},
		{PersonalNumber, "IsPersonalNumber", (*Phone).IsPersonalNumber},
		{Pager, "IsPager", (*Phone).IsPager},
		{UAN, "IsUAN", (*Phone).IsUAN},
		{Voicemail, "IsVoicemail", (*Phone).IsVoicemail},
	}

	for _, pred := range predicates {
		t.Run(pred.name, func(t *testing.T) {
			p, region := exampleOfType(t, pred.typ)

			if !pred.fn(p) {
				t.Fatalf("%s() = false for %s, a %v number in %s",
					pred.name, p.E164(), p.Type(), region)
			}
			// Exactly one predicate may hold for a given number.
			for _, other := range predicates {
				if other.typ == pred.typ {
					continue
				}
				if other.fn(p) {
					t.Errorf("%s() is also true for %s, which is %v",
						other.name, p.E164(), p.Type())
				}
			}
		})
	}
}

// exampleOfType finds a region whose example number for a range really
// resolves to that range, and returns it.
func exampleOfType(t *testing.T, typ PhoneType) (*Phone, string) {
	t.Helper()
	for _, region := range SupportedRegions() {
		p, ok := ExampleNumberForType(region, typ)
		if !ok || p.Type() != typ {
			continue
		}
		return p, region
	}
	t.Fatalf("no region has an example number that resolves to %v", typ)
	return nil, ""
}

// TestFixedLineOrMobilePredicate covers the range that is deliberately not
// one of the two it spans.
func TestFixedLineOrMobilePredicate(t *testing.T) {
	p, err := Parse("+1 202 555 0123")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.Type() != FixedLineOrMobile {
		t.Skipf("the example number is now %v; pick another", p.Type())
	}
	if !p.IsFixedLineOrMobile() {
		t.Error("IsFixedLineOrMobile() = false")
	}
	// A number in the shared range is neither purely fixed-line nor purely
	// mobile, and the narrow predicates must say so.
	if p.IsLandline() || p.IsMobile() {
		t.Error("a FIXED_LINE_OR_MOBILE number must not report as either alone")
	}
}

// TestPredicatesOnUnparsedPhone checks the zero value, which callers reach by
// declaring a Phone and hitting a parse error.
func TestPredicatesOnUnparsedPhone(t *testing.T) {
	var p Phone
	for name, fn := range map[string]func(*Phone) bool{
		"IsMobile":         (*Phone).IsMobile,
		"IsLandline":       (*Phone).IsLandline,
		"IsTollFree":       (*Phone).IsTollFree,
		"IsPremiumRate":    (*Phone).IsPremiumRate,
		"IsSharedCost":     (*Phone).IsSharedCost,
		"IsVoIP":           (*Phone).IsVoIP,
		"IsPersonalNumber": (*Phone).IsPersonalNumber,
		"IsPager":          (*Phone).IsPager,
		"IsUAN":            (*Phone).IsUAN,
		"IsVoicemail":      (*Phone).IsVoicemail,
		"IsValid":          (*Phone).IsValid,
		"IsPossible":       (*Phone).IsPossible,
	} {
		if fn(&p) {
			t.Errorf("%s() = true for the zero value", name)
		}
	}
}
