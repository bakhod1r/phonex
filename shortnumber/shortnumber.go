// Package shortnumber answers questions about short numbers: the three- to
// six-digit codes such as 112, 911 or 10086 that only work inside one country
// and carry no calling code.
//
// Short numbers are a different problem from ordinary phone numbers. They have
// no international form, the same digits mean different things in different
// countries, and they are dialled rather than stored. The main phonex package
// therefore rejects them as too short, and this package handles them from its
// own metadata, generated from libphonenumber's ShortNumberMetadata.xml.
//
// Every function takes the region the number is being dialled in, because
// without it the digits mean nothing:
//
//	shortnumber.IsEmergency("112", "GB")  // true
//	shortnumber.IsEmergency("112", "US")  // true
//	shortnumber.IsValid("100", "GB")      // true, the BT operator
//	shortnumber.IsValid("100", "UZ")      // false
//
// The data set is about half a megabyte. It lives in its own package so that
// a program which never calls these functions does not link it in.
package shortnumber

import (
	"sort"
	"strings"

	"github.com/bakhod1r/phonex/internal/lazyre"
)

// Desc describes one range of short numbers.
type Desc struct {
	// Pattern matches the whole number.
	Pattern lazyre.Re
	// Prefix matches the number as a prefix. It is only generated for the
	// emergency range, which is the one tested that way.
	Prefix lazyre.Re
	// Lengths holds the sorted possible lengths.
	Lengths []int32
	// Example is a number from this range, or "".
	Example string
}

// Exists reports whether the region defines this range.
func (d *Desc) Exists() bool { return d != nil && (!d.Pattern.Empty() || len(d.Lengths) > 0) }

// hasLength reports whether n is a length this range uses.
func (d *Desc) hasLength(n int) bool {
	for _, l := range d.Lengths {
		if int(l) == n {
			return true
		}
		if int(l) > n {
			break
		}
	}
	return false
}

// matches reports whether the number is both a possible length and a pattern
// match. Length alone is not enough, and a pattern match on a number of the
// wrong length would be a metadata inconsistency.
func (d *Desc) matches(number string) bool {
	if d == nil || !d.Exists() {
		return false
	}
	if len(d.Lengths) > 0 && !d.hasLength(len(number)) {
		return false
	}
	return d.Pattern.Match(number)
}

// Metadata holds the short number ranges of one region.
type Metadata struct {
	Region string

	// General matches any short number the region uses.
	General Desc
	// ShortCode is the range of assigned short numbers.
	ShortCode Desc
	// TollFree numbers are free to the caller.
	TollFree Desc
	// PremiumRate numbers are charged above the standard rate.
	PremiumRate Desc
	// Emergency reaches the emergency services.
	Emergency Desc
	// ExpandedEmergency reaches services that are not strictly emergency
	// numbers but are dialled in the same situations.
	ExpandedEmergency Desc
	// StandardRate numbers are charged at the normal rate.
	StandardRate Desc
	// CarrierSpecific numbers only work on some networks.
	CarrierSpecific Desc
	// SMSServices numbers accept text messages.
	SMSServices Desc
}

// Cost is what a caller is charged for reaching a short number.
type Cost uint8

const (
	// UnknownCost means the metadata does not say, which is the common case:
	// most regions only classify part of their short number range.
	UnknownCost Cost = iota
	// TollFree means the call is free to the caller.
	TollFree
	// StandardRate means the call is charged as a normal call.
	StandardRate
	// PremiumRate means the call is charged above the normal rate.
	PremiumRate
)

func (c Cost) String() string {
	switch c {
	case TollFree:
		return "TOLL_FREE"
	case StandardRate:
		return "STANDARD_RATE"
	case PremiumRate:
		return "PREMIUM_RATE"
	default:
		return "UNKNOWN_COST"
	}
}

// exactEmergencyRegions are the regions where dialling an emergency number
// with extra digits appended does not connect. Everywhere else the network
// acts on the emergency prefix alone.
var exactEmergencyRegions = map[string]bool{"BR": true, "CL": true, "NI": true}

// Region returns the short number metadata for a region, or nil.
func Region(region string) *Metadata {
	return data[normalizeRegion(region)]
}

// Regions returns every region with short number metadata, sorted.
func Regions() []string {
	out := make([]string, 0, len(data))
	for r := range data {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

func normalizeRegion(region string) string {
	return strings.ToUpper(strings.TrimSpace(region))
}

// digitsOnly keeps the digits of a dialled string. A leading '+' means the
// caller passed an international number, which can never be a short number,
// and is reported by the second result.
func digitsOnly(number string) (string, bool) {
	number = strings.TrimSpace(number)
	if strings.HasPrefix(number, "+") {
		return "", false
	}
	var b strings.Builder
	b.Grow(len(number))
	for i := 0; i < len(number); i++ {
		if c := number[i]; c >= '0' && c <= '9' {
			b.WriteByte(c)
		}
	}
	return b.String(), true
}

// IsPossible reports whether the number has a length the region uses for
// short numbers. It does not check that the number is assigned.
func IsPossible(number, region string) bool {
	m := Region(region)
	if m == nil {
		return false
	}
	digits, ok := digitsOnly(number)
	if !ok || digits == "" {
		return false
	}
	return m.General.hasLength(len(digits))
}

// IsValid reports whether the number is an assigned short number in the
// region.
func IsValid(number, region string) bool {
	m := Region(region)
	if m == nil {
		return false
	}
	digits, ok := digitsOnly(number)
	if !ok || digits == "" {
		return false
	}
	if !m.General.matches(digits) {
		return false
	}
	return m.ShortCode.matches(digits)
}

// IsEmergency reports whether the number is exactly an emergency number in
// the region.
func IsEmergency(number, region string) bool {
	return matchesEmergency(number, region, false)
}

// ConnectsToEmergency reports whether dialling the number reaches the
// emergency services, allowing for digits typed after the emergency number
// itself. Networks in most countries act on the emergency prefix alone, so
// "911123" still connects; in Brazil, Chile and Nicaragua it does not, and
// this reports false there.
//
// This is the check to use before deciding that a number is safe to dial
// automatically.
func ConnectsToEmergency(number, region string) bool {
	return matchesEmergency(number, region, true)
}

func matchesEmergency(number, region string, allowPrefix bool) bool {
	region = normalizeRegion(region)
	m := data[region]
	if m == nil || !m.Emergency.Exists() {
		return false
	}
	digits, ok := digitsOnly(number)
	if !ok || digits == "" {
		return false
	}
	if allowPrefix && !exactEmergencyRegions[region] {
		return m.Emergency.Prefix.Match(digits)
	}
	return m.Emergency.Pattern.Match(digits)
}

// ExpectedCost reports what reaching the number costs the caller. It returns
// UnknownCost for a number the region does not classify, which includes every
// number that is not a valid short number there.
func ExpectedCost(number, region string) Cost {
	m := Region(region)
	if m == nil {
		return UnknownCost
	}
	digits, ok := digitsOnly(number)
	if !ok || digits == "" {
		return UnknownCost
	}
	if !m.General.matches(digits) {
		return UnknownCost
	}
	switch {
	case m.PremiumRate.matches(digits):
		return PremiumRate
	case m.StandardRate.matches(digits):
		return StandardRate
	case m.TollFree.matches(digits):
		return TollFree
	case m.Emergency.Pattern.Match(digits):
		// Emergency numbers are free everywhere, whether or not the region
		// lists them under its toll-free range.
		return TollFree
	default:
		return UnknownCost
	}
}

// IsCarrierSpecific reports whether the number only works on some of the
// region's networks.
func IsCarrierSpecific(number, region string) bool {
	return matchesDesc(number, region, func(m *Metadata) *Desc { return &m.CarrierSpecific })
}

// IsSMSService reports whether the number accepts text messages.
func IsSMSService(number, region string) bool {
	return matchesDesc(number, region, func(m *Metadata) *Desc { return &m.SMSServices })
}

func matchesDesc(number, region string, pick func(*Metadata) *Desc) bool {
	m := Region(region)
	if m == nil {
		return false
	}
	digits, ok := digitsOnly(number)
	if !ok || digits == "" {
		return false
	}
	return pick(m).matches(digits)
}
