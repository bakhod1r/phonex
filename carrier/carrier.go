// Package carrier reports the network a phone number was issued on.
//
// The answer is a guess, and the package says so in its API. Number
// portability means a number can be moved between networks while keeping its
// prefix, so in most countries the prefix identifies the network the number
// was *issued* on, not the one it is on today. Use SafeDisplayName when the
// answer will be shown to a user: it withholds a name in every region where
// portability makes it unreliable.
//
// The data is English only, about half a megabyte, and lives in its own
// package so a program that never calls this does not link it in.
//
//	p, _ := phonex.Parse("+44 7400 123456")
//	carrier.Name(p) // "EE"
package carrier

import "github.com/bakhod1r/phonex"

// Name returns the network the number's prefix was issued to, or "" when the
// prefix is not in the data.
//
// The name is not necessarily the network the number is on now. Prefer
// SafeDisplayName for anything a user will see.
func Name(p *phonex.Phone) string {
	if p == nil {
		return ""
	}
	return NameForDigits(p.Digits())
}

// NameForDigits is Name for a number already reduced to its calling code
// followed by its national number, with no '+' and no separators.
func NameForDigits(digits string) string {
	i := numbers.Lookup(digits)
	if i < 0 {
		return ""
	}
	return values[i]
}

// SafeDisplayName returns the carrier name only when it is safe to show: the
// number must be valid, and its region must not support number portability.
// Everywhere else it returns "", because the prefix no longer identifies the
// network and showing a stale name would mislead.
func SafeDisplayName(p *phonex.Phone) string {
	if p == nil || !p.IsValid() || p.MobileNumberPortable() {
		return ""
	}
	return Name(p)
}

// NameForNumber parses a number and returns its carrier. It is the one-call
// form for callers that hold a string.
func NameForNumber(number string, opts ...phonex.ParseOption) (string, error) {
	p, err := phonex.Parse(number, opts...)
	if err != nil {
		return "", err
	}
	return Name(p), nil
}

// Count returns the number of prefixes in the data set.
func Count() int { return len(numbers.Prefixes) }
