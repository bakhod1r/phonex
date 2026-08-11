package phonex

import "strings"

// Match grades how closely two numbers correspond, in the spirit of
// libphonenumber's isNumberMatch. It is the comparison to use when one side
// is written in national form and the other in international form.
func (p *Phone) Match(other *Phone) MatchType {
	if p == nil || other == nil || p.meta == nil || other.meta == nil {
		return NoMatch
	}

	a, b := p.nsnRef(), other.nsnRef()
	sameCode := p.meta.DialCode == other.meta.DialCode

	if sameCode && a == b {
		if p.Extension() != other.Extension() {
			return NoMatch
		}
		return ExactMatch
	}

	// Once the calling codes are both known and differ, nothing else can
	// make the numbers equal.
	if sameCode {
		return matchBySuffix(a, b)
	}
	if bothCodesCertain(p, other) {
		return NoMatch
	}
	if a == b {
		return NSNMatch
	}
	return matchBySuffix(a, b)
}

// bothCodesCertain reports whether each number carried its own calling code,
// which makes a mismatch conclusive.
func bothCodesCertain(a, b *Phone) bool {
	return a.source != FromDefaultCountry && b.source != FromDefaultCountry
}

// matchBySuffix reports a short match when one national number ends with the
// other and the shared part is long enough to be meaningful. Callers should
// treat ShortNSNMatch as "possibly the same", never as equality.
func matchBySuffix(a, b string) MatchType {
	const minSharedDigits = 7
	if len(a) < minSharedDigits || len(b) < minSharedDigits {
		return NoMatch
	}
	if strings.HasSuffix(a, b) || strings.HasSuffix(b, a) {
		return ShortNSNMatch
	}
	return NoMatch
}

// MatchNumbers parses both inputs and grades how closely they correspond.
// Inputs that fail to parse yield NoMatch.
func MatchNumbers(a, b string, opts ...ParseOption) MatchType {
	var pa, pb Phone
	if pa.Parse(a, opts...) != nil || pb.Parse(b, opts...) != nil {
		return NoMatch
	}
	return pa.Match(&pb)
}

// Equal reports whether two numbers are the same number, extension included.
func (p *Phone) Equal(other *Phone) bool {
	return p.Match(other) == ExactMatch
}

// EqualExact reports whether two numbers are equal and were written
// identically. Use it only when the original spelling is itself significant.
func (p *Phone) EqualExact(other *Phone) bool {
	if p == nil || other == nil {
		return false
	}
	return p.Equal(other) && p.raw == other.raw
}

// Equal parses both inputs and reports whether they are the same number.
func Equal(a, b string, opts ...ParseOption) bool {
	return MatchNumbers(a, b, opts...) == ExactMatch
}
