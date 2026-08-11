package phonex

import (
	"sync/atomic"

	"github.com/bakhod1r/phonex/countries"
)

// typeOrder is the order libphonenumber resolves ranges in. The specific
// ranges are tested before the broad fixed-line and mobile ones, because
// several regions define a premium-rate or toll-free block inside a range the
// general fixed-line pattern also covers.
var typeOrder = [...]PhoneType{
	PremiumRate, TollFree, SharedCost, VoIP, PersonalNumber, Pager, UAN, Voicemail,
}

// typeFor resolves the range a national significant number falls into.
func typeFor(m *countries.Metadata, nsn string) PhoneType {
	if m == nil || !m.General.Match(nsn) {
		return Unknown
	}
	for _, t := range typeOrder {
		if m.Descs[t].Match(nsn) {
			return t
		}
	}
	fixed := m.Descs[FixedLine].Match(nsn)
	if fixed {
		// Where a region does not distinguish the two, the metadata repeats
		// the same pattern for both ranges.
		if sameRange(m, FixedLine, Mobile) || m.Descs[Mobile].Match(nsn) {
			return FixedLineOrMobile
		}
		return FixedLine
	}
	if m.Descs[Mobile].Match(nsn) {
		if sameRange(m, FixedLine, Mobile) {
			return FixedLineOrMobile
		}
		return Mobile
	}
	return Unknown
}

func sameRange(m *countries.Metadata, a, b PhoneType) bool {
	sa := m.Descs[a].Pattern.Source()
	return sa != "" && sa == m.Descs[b].Pattern.Source()
}

// Type returns the range this number falls into. The result is computed on
// first use and cached.
func (p *Phone) Type() PhoneType {
	if p == nil || p.meta == nil {
		return Unknown
	}
	if cached := atomic.LoadUint32(&p.typ); cached != 0 {
		return PhoneType(cached - 1)
	}
	t := typeFor(p.meta, p.nsnRef())
	atomic.StoreUint32(&p.typ, uint32(t)+1)
	return t
}

// IsValid reports whether the number matches one of its region's number
// ranges. This is the check to use before storing or dialling a number.
func (p *Phone) IsValid() bool {
	return p != nil && p.meta != nil && p.Type() != Unknown
}

// IsValidForRegion reports whether the number is valid and belongs to the
// given region. Use it when a number must come from one specific country
// rather than from anywhere sharing its calling code.
func (p *Phone) IsValidForRegion(region string) bool {
	if p == nil || p.meta == nil {
		return false
	}
	m, ok := countries.Data[normalizeRegion(region)]
	if !ok || m.DialCode != p.meta.DialCode {
		return false
	}
	return typeFor(m, p.nsnRef()) != Unknown
}

// Possibility reports whether the number has a plausible length, without
// checking it against the detailed range patterns. It is the cheap check: it
// never compiles a pattern.
//
// Lengths are judged against the region that owns the calling code rather
// than the specific region the number resolves to, because a length is a
// property of the numbering plan as a whole. A Curaçao-length number written
// with a Bonaire prefix is therefore possible but not valid.
func (p *Phone) Possibility() Possibility {
	if p == nil || p.meta == nil {
		return InvalidCountryCode
	}
	return testPossibility(mainRegionFor(p.meta), p.nsnRef(), Unknown)
}

// PossibilityForType is Possibility restricted to one range.
func (p *Phone) PossibilityForType(t PhoneType) Possibility {
	if p == nil || p.meta == nil {
		return InvalidCountryCode
	}
	return testPossibility(mainRegionFor(p.meta), p.nsnRef(), t)
}

// IsPossible reports whether the number's length is one the region uses,
// including lengths that only work when dialled locally. A number can be
// possible without being valid: "+1 555 555 5555" has the right shape for the
// US but is not an assigned range.
//
// Use Possibility to tell a full number from a local-only one.
func (p *Phone) IsPossible() bool {
	switch p.Possibility() {
	case IsPossibleNumber, IsPossibleLocalOnly:
		return true
	default:
		return false
	}
}

// testPossibility compares the digit count against the lengths the region
// allows, optionally narrowed to a single range.
func testPossibility(m *countries.Metadata, nsn string, t PhoneType) Possibility {
	if m == nil {
		return InvalidCountryCode
	}
	lengths, localOnly := possibleLengths(m, t)
	if len(lengths) == 0 {
		// A region that defines no lengths for this range does not have it.
		return InvalidLength
	}

	actual := len(nsn)
	if actual < int(lengths[0]) {
		if containsLen(localOnly, actual) {
			return IsPossibleLocalOnly
		}
		return TooShort
	}
	if actual > int(lengths[len(lengths)-1]) {
		return TooLong
	}
	if containsLen(lengths, actual) {
		return IsPossibleNumber
	}
	if containsLen(localOnly, actual) {
		return IsPossibleLocalOnly
	}
	return InvalidLength
}

// possibleLengths returns the lengths allowed for a range, falling back to the
// region-wide lengths when the range does not narrow them.
func possibleLengths(m *countries.Metadata, t PhoneType) (lengths, localOnly []int32) {
	switch {
	case t == Unknown:
		return m.General.Lengths, m.General.LocalOnly
	case t == FixedLineOrMobile:
		// Either range will do, so the union applies.
		f, fl := m.Descs[FixedLine].Lengths, m.Descs[FixedLine].LocalOnly
		mo, ml := m.Descs[Mobile].Lengths, m.Descs[Mobile].LocalOnly
		return mergeLens(f, mo), mergeLens(fl, ml)
	case t < countries.NumDescs:
		d := &m.Descs[t]
		if len(d.Lengths) == 0 {
			if !d.Exists() {
				return nil, nil
			}
			return m.General.Lengths, m.General.LocalOnly
		}
		return d.Lengths, d.LocalOnly
	default:
		return m.General.Lengths, m.General.LocalOnly
	}
}

// mergeLens merges two sorted length lists. The inputs are at most a handful
// of entries, so a simple merge beats anything cleverer.
func mergeLens(a, b []int32) []int32 {
	switch {
	case len(a) == 0:
		return b
	case len(b) == 0:
		return a
	}
	out := make([]int32, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] < b[j]:
			out = append(out, a[i])
			i++
		case a[i] > b[j]:
			out = append(out, b[j])
			j++
		default:
			out = append(out, a[i])
			i++
			j++
		}
	}
	out = append(out, a[i:]...)
	return append(out, b[j:]...)
}

func containsLen(ls []int32, n int) bool {
	for _, l := range ls {
		if int(l) == n {
			return true
		}
		if int(l) > n {
			return false
		}
	}
	return false
}

// CanBeInternationallyDialled reports whether the number is reachable from
// outside its own country. Some ranges, such as short service numbers, are
// domestic only.
func (p *Phone) CanBeInternationallyDialled() bool {
	if p == nil || p.meta == nil {
		return false
	}
	if !p.meta.NoIntlDialling.Exists() {
		return true
	}
	return !p.meta.NoIntlDialling.Match(p.nsnRef())
}

// IsMobile reports whether the number is a mobile number. Numbers in regions
// that do not separate mobile from fixed-line ranges report false; use Type
// when that distinction matters.
func (p *Phone) IsMobile() bool { return p.Type() == Mobile }

// IsLandline reports whether the number is a fixed-line number.
func (p *Phone) IsLandline() bool { return p.Type() == FixedLine }

// IsFixedLineOrMobile reports whether the number falls in a range the region
// uses for both fixed-line and mobile numbers.
func (p *Phone) IsFixedLineOrMobile() bool { return p.Type() == FixedLineOrMobile }

// IsVoIP reports whether the number is a VoIP number.
func (p *Phone) IsVoIP() bool { return p.Type() == VoIP }

// IsTollFree reports whether calls to the number are free to the caller.
func (p *Phone) IsTollFree() bool { return p.Type() == TollFree }

// IsPremiumRate reports whether calls to the number are charged at a premium.
func (p *Phone) IsPremiumRate() bool { return p.Type() == PremiumRate }

// IsSharedCost reports whether the cost of calls is shared with the callee.
func (p *Phone) IsSharedCost() bool { return p.Type() == SharedCost }

// IsPager reports whether the number is a pager.
func (p *Phone) IsPager() bool { return p.Type() == Pager }

// IsUAN reports whether the number is a universal access number.
func (p *Phone) IsUAN() bool { return p.Type() == UAN }

// IsVoicemail reports whether the number reaches a voicemail service.
func (p *Phone) IsVoicemail() bool { return p.Type() == Voicemail }

// IsPersonalNumber reports whether the number is a personal ("follow me")
// number.
func (p *Phone) IsPersonalNumber() bool { return p.Type() == PersonalNumber }

// IsValid parses input and reports whether it is a valid number.
func IsValid(input string, opts ...ParseOption) bool {
	var p Phone
	return p.Parse(input, opts...) == nil && p.IsValid()
}

// IsPossible parses input and reports whether its length is plausible.
func IsPossible(input string, opts ...ParseOption) bool {
	var p Phone
	return p.Parse(input, opts...) == nil && p.IsPossible()
}
