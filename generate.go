package phonex

import (
	"math/rand"
	"sort"
	"strings"
	"sync"

	"github.com/bakhod1r/phonex/countries"
)

// generateAttempts bounds the search for a random number that lands inside
// the requested range. The example number's prefix is kept, so a handful of
// attempts is normally enough; the bound only matters for ranges that are
// sparse within their prefix.
const generateAttempts = 50

// AnyType asks for a number of whatever range the region defines, preferring
// mobile. It is the type Generate uses.
//
// It shares a value with Unknown, which Type reports when no range matches:
// both mean "no particular range", read from opposite ends. Passing a Type()
// result straight back into GenerateWith therefore asks for any type rather
// than failing, which is the harmless reading of the two.
const AnyType = Unknown

// Generate returns a random valid number for a region, preferring a mobile
// number. It reports false when the region is unknown or its metadata carries
// no example to build on.
//
// Generated numbers are for tests and demos. They are valid in the sense that
// IsValid accepts them, which means they may well belong to a real subscriber
// — never dial or message them.
func Generate(region string) (*Phone, bool) {
	return GenerateWith(region, AnyType, rand.Intn)
}

// GenerateForType returns a random valid number of a given range. It reports
// false when the region does not define that range.
//
// Where a country does not separate its fixed-line and mobile ranges, asking
// for either returns a number the metadata calls FixedLineOrMobile, since
// that is as precise as the plan gets.
//
// See Generate for the warning that applies to the result.
func GenerateForType(region string, t PhoneType) (*Phone, bool) {
	return GenerateWith(region, t, rand.Intn)
}

// GenerateWith is Generate with the randomness supplied by the caller: intn
// must return a value in [0,n). A program that has to reproduce its output
// from a seed cannot use the global math/rand that Generate draws from.
//
//	r := rand.New(rand.NewSource(1))
//	p, ok := phonex.GenerateWith("GB", phonex.Mobile, r.Intn)
//
// Pass AnyType for t to accept any range the region defines, which is what
// Generate does. A nil intn falls back to the global source.
//
// See Generate for the warning that applies to the result.
func GenerateWith(region string, t PhoneType, intn func(n int) int) (*Phone, bool) {
	if intn == nil {
		intn = rand.Intn
	}
	region = normalizeRegion(region)

	if t == AnyType {
		if p, ok := generateOfType(region, Mobile, intn); ok {
			return p, true
		}
		// Regions without a mobile range still have some example to offer.
		return ExampleNumber(region)
	}
	return generateOfType(region, t, intn)
}

// generateOfType builds a number of one concrete range.
func generateOfType(region string, t PhoneType, intn func(n int) int) (*Phone, bool) {
	example, ok := ExampleNumberForType(region, t)
	if !ok {
		return nil, false
	}
	meta := example.Metadata()
	if meta == nil {
		return example, true
	}

	// Keep the leading digits, which carry the operator and area code, and
	// randomise the subscriber part. Randomising more than that would mostly
	// produce numbers outside the range.
	nsn := example.NSN()
	vary := len(nsn) / 2
	if vary > 6 {
		vary = 6
	}
	if vary < 2 {
		// Too short to randomise without leaving the range.
		return example, true
	}
	prefix := nsn[:len(nsn)-vary]

	if p, ok := fill(meta, region, prefix, len(nsn), t, intn); ok {
		return p, true
	}
	// Nothing better was found, so hand back the known-good example.
	return example, true
}

// fill completes prefix with random digits until the national number is
// length digits long, and returns the first result that is valid for the
// region and of the wanted type. Pass AnyType to accept any range.
func fill(meta *Metadata, region, prefix string, length int, t PhoneType, intn func(n int) int) (*Phone, bool) {
	vary := length - len(prefix)
	if vary <= 0 {
		return nil, false
	}

	buf := make([]byte, 0, 1+len(meta.DialCode)+length)
	var candidate Phone
	for i := 0; i < generateAttempts; i++ {
		buf = append(buf[:0], '+')
		buf = append(buf, meta.DialCode...)
		buf = append(buf, prefix...)
		for j := 0; j < vary; j++ {
			buf = append(buf, byte('0'+intn(10)))
		}

		if candidate.ParseBytes(buf) != nil {
			continue
		}
		if !candidate.IsValidForRegion(region) {
			continue
		}
		if !typeMatches(t, candidate.Type()) {
			continue
		}
		return candidate.Clone(), true
	}
	return nil, false
}

// typeMatches reports whether a generated number of type got satisfies a
// request for type want.
//
// Countries that do not separate their fixed-line and mobile ranges report
// every number as FixedLineOrMobile. Asking such a region for a mobile number
// and rejecting what it offers would leave nothing to randomise, so the
// example number would come back unchanged on every call.
func typeMatches(want, got PhoneType) bool {
	if want == AnyType || want == got {
		return true
	}
	return got == FixedLineOrMobile && (want == FixedLine || want == Mobile)
}

// GenerateForPrefix returns a valid number for the region whose national
// number starts with the given digits — an area or operator code, such as
// "20" for London or "416" for Toronto.
//
// The caller knows the code but not the shape around it: the national
// number's length varies within a country (London's 20 takes eight more
// digits where most UK codes take seven), and some plans count the trunk
// digit as part of the national number, so Rome is "06" here and 6 in an
// atlas. Both are resolved here, so the prefix may be written either way. It
// reports false when no shape the plan defines accepts the prefix.
//
// intn supplies the randomness, as in GenerateWith; a nil intn uses the
// global source.
//
// See Generate for the warning that applies to the result.
func GenerateForPrefix(region, prefix string, intn func(n int) int) (*Phone, bool) {
	if intn == nil {
		intn = rand.Intn
	}
	region = normalizeRegion(region)
	meta, ok := countries.Data[region]
	if !ok {
		return nil, false
	}

	digits := digitsOf(prefix)
	if digits == "" || len(digits) > meta.MaxLength {
		return nil, false
	}

	// The shape that fits a prefix does not change between calls, and finding
	// it costs a scan over every length the plan defines with up to
	// generateAttempts parses each. Remember it — including the verdict that
	// no shape fits, which is the expensive case to repeat.
	key := region + "|" + digits
	if v, hit := prefixShapes.Load(key); hit {
		s := v.(prefixShape)
		if !s.ok {
			return nil, false
		}
		if p, found := fill(meta, region, s.nsnPrefix, s.length, AnyType, intn); found {
			return p, true
		}
		// A cached shape that suddenly yields nothing means the range is
		// sparse rather than absent, so fall through and search again.
	}

	for _, s := range candidateShapes(meta, digits) {
		if p, found := fill(meta, region, s.nsnPrefix, s.length, AnyType, intn); found {
			prefixShapes.Store(key, s)
			return p, true
		}
	}
	prefixShapes.Store(key, prefixShape{})
	return nil, false
}

// prefixShape is a national number length together with the leading digits
// that reach it, as resolved from a caller's prefix.
type prefixShape struct {
	nsnPrefix string
	length    int
	ok        bool
}

// prefixShapes caches the resolution of a region and prefix, both the shape
// that worked and the fact that none did.
var prefixShapes sync.Map // string -> prefixShape

// candidateShapes returns the shapes to try for a prefix, most likely first.
func candidateShapes(meta *Metadata, digits string) []prefixShape {
	// The prefix may be written with or without the trunk digit, so try it
	// both ways. The literal form goes first: where both readings are valid
	// the caller's own is the one they meant, and only after that does this
	// function get to reinterpret it.
	forms := []string{digits}
	add := func(f string) {
		if f == "" || f == digits {
			return
		}
		for _, seen := range forms {
			if seen == f {
				return
			}
		}
		forms = append(forms, f)
	}

	np := meta.NationalPrefix
	if np == "" {
		// Italy and Uzbekistan have no trunk prefix to strip, but Italy still
		// writes the leading zero as part of the national number, so an area
		// code quoted from an atlas is short by one digit. Nothing invalid
		// comes of trying: every candidate is validated before it is
		// returned.
		np = "0"
	}
	add(strings.TrimPrefix(digits, np))
	add(np + digits)

	// Prefer lengths close to the region's own example, which is the shape a
	// caller almost always means. Without that, a plan offering both nine and
	// eleven digits would answer with whichever the metadata lists first.
	typical := meta.MaxLength
	if ex, ok := ExampleNumber(meta.ISO2); ok {
		typical = len(ex.NSN())
	}
	lengths := make([]int, 0, len(meta.General.Lengths))
	for _, l := range meta.General.Lengths {
		lengths = append(lengths, int(l))
	}
	sort.SliceStable(lengths, func(i, j int) bool {
		return abs(lengths[i]-typical) < abs(lengths[j]-typical)
	})

	out := make([]prefixShape, 0, len(forms)*len(lengths))
	for _, form := range forms {
		for _, l := range lengths {
			// A prefix as long as the number itself leaves nothing to
			// randomise, and fill would reject it anyway.
			if l > len(form) {
				out = append(out, prefixShape{nsnPrefix: form, length: l, ok: true})
			}
		}
	}
	return out
}

// digitsOf keeps the digits of a written prefix, so that "(020)" and "020"
// mean the same thing. A leading '+' is not accepted: a prefix is a fragment
// of a national number, not a number.
func digitsOf(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "+") {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if c := s[i]; c >= '0' && c <= '9' {
			b.WriteByte(c)
		}
	}
	return b.String()
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
