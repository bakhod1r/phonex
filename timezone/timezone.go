// Package timezone reports the IANA time zones a phone number reaches.
//
// The answer comes from the number's prefix, so it is a property of where the
// number was issued, not of where its owner is now. A mobile number keeps its
// zone when its owner emigrates. Treat the result as a good default for
// displaying a local time, not as a fact about a person.
//
// The data is about a hundred kilobytes and lives in its own package, so a
// program that never calls this does not link it in.
//
//	p, _ := phonex.Parse("+1 212 555 0123")
//	timezone.For(p) // ["America/New_York"]
package timezone

import "github.com/bakhod1r/phonex"

// For returns the time zones the number reaches, most likely first as
// upstream orders them, or nil when the prefix is not in the data.
//
// A prefix can map to several zones: a calling code that spans a continent,
// such as +1 or +7, resolves to a long list until enough digits are known.
// The returned slice is shared and must not be modified.
func For(p *phonex.Phone) []string {
	if p == nil {
		return nil
	}
	return ForDigits(p.Digits())
}

// ForDigits is For for a number already reduced to its calling code followed
// by its national number, with no '+' and no separators.
func ForDigits(digits string) []string {
	i := numbers.Lookup(digits)
	if i < 0 {
		return nil
	}
	return values[i]
}

// ForNumber parses a number and returns its time zones. It is the one-call
// form for callers that hold a string.
func ForNumber(number string, opts ...phonex.ParseOption) ([]string, error) {
	p, err := phonex.Parse(number, opts...)
	if err != nil {
		return nil, err
	}
	return For(p), nil
}

// Count returns the number of prefixes in the data set. It exists so that a
// caller can assert the data was linked in as expected.
func Count() int { return len(numbers.Prefixes) }
