package phonex

import "math/rand"

// generateAttempts bounds the search for a random number that lands inside
// the requested range. The example number's prefix is kept, so a handful of
// attempts is normally enough; the bound only matters for ranges that are
// sparse within their prefix.
const generateAttempts = 50

// Generate returns a random valid number for a region, preferring a mobile
// number. It reports false when the region is unknown or its metadata carries
// no example to build on.
//
// Generated numbers are for tests and demos. They are valid in the sense that
// IsValid accepts them, which means they may well belong to a real subscriber
// — never dial or message them.
func Generate(region string) (*Phone, bool) {
	if p, ok := GenerateForType(region, Mobile); ok {
		return p, true
	}
	// Regions without a mobile range still have some example to offer.
	return ExampleNumber(region)
}

// GenerateForType returns a random valid number of a given range. It reports
// false when the region does not define that range.
//
// See Generate for the warning that applies to the result.
func GenerateForType(region string, t PhoneType) (*Phone, bool) {
	region = normalizeRegion(region)
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

	buf := make([]byte, 0, 1+len(meta.DialCode)+len(nsn))
	var candidate Phone
	for i := 0; i < generateAttempts; i++ {
		buf = append(buf[:0], '+')
		buf = append(buf, meta.DialCode...)
		buf = append(buf, prefix...)
		for j := 0; j < vary; j++ {
			// rand's top-level functions are safe for concurrent use, which
			// a package-level *rand.Rand would not be.
			buf = append(buf, byte('0'+rand.Intn(10)))
		}

		if candidate.ParseBytes(buf) != nil {
			continue
		}
		if candidate.IsValidForRegion(region) && candidate.Type() == t {
			return candidate.Clone(), true
		}
	}

	// Nothing better was found, so hand back the known-good example.
	return example, true
}
