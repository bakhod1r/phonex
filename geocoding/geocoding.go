// Package geocoding reports the area a phone number was issued in.
//
// The answer comes from the number's prefix, so it describes where the number
// was allocated, not where its owner is. A mobile number keeps its area
// forever, and in many countries mobile prefixes carry no area at all. Treat
// the result as a label to display, never as a location.
//
// The data is English only and is by far the largest in the library, several
// megabytes. It lives in its own package so a program that never calls this
// does not link it in.
//
//	p, _ := phonex.Parse("+44 20 7031 3000")
//	geocoding.Area(p) // "London"
package geocoding

import "github.com/bakhod1r/phonex"

// Area returns the description of the area the number's prefix belongs to, or
// "" when the data has none. Mobile ranges usually have none, since they are
// not tied to a place.
func Area(p *phonex.Phone) string {
	if p == nil {
		return ""
	}
	return AreaForDigits(p.Digits())
}

// AreaForDigits is Area for a number already reduced to its calling code
// followed by its national number, with no '+' and no separators.
func AreaForDigits(digits string) string {
	i := numbers.Lookup(digits)
	if i < 0 {
		return ""
	}
	return values[i]
}

// Describe returns the most specific description available: the area if the
// data has one, otherwise the country name, otherwise "".
//
// This is the function to use for display, since a number with no area is
// still worth labelling with its country.
func Describe(p *phonex.Phone) string {
	if p == nil {
		return ""
	}
	if area := Area(p); area != "" {
		return area
	}
	return p.CountryName()
}

// AreaForNumber parses a number and returns its area. It is the one-call form
// for callers that hold a string.
func AreaForNumber(number string, opts ...phonex.ParseOption) (string, error) {
	p, err := phonex.Parse(number, opts...)
	if err != nil {
		return "", err
	}
	return Area(p), nil
}

// Count returns the number of prefixes in the data set.
func Count() int { return len(numbers.Prefixes) }
