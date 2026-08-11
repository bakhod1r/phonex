// Package prefixmap looks up the longest number prefix that a data set knows
// about.
//
// The geocoding, carrier and time zone data sets all have the same shape: a
// list of number prefixes, each mapped to a value, where the answer for a
// number is whatever the longest matching prefix says. They share this
// implementation so that only the values differ between them.
package prefixmap

// Map is a set of number prefixes, sorted by their numeric value.
//
// A prefix is the calling code followed by the leading digits of the national
// number, so it never has a leading zero and never exceeds nine digits. That
// makes its numeric value a faithful stand-in for the digit string, which is
// what lets the lookup be a binary search over a flat array rather than a
// hash of strings.
type Map struct {
	// Prefixes holds the prefix values in ascending order.
	Prefixes []uint32
	// Values holds, for each prefix, the index of its value in whatever
	// table the calling package owns.
	Values []uint32
	// MinLen and MaxLen are the shortest and longest prefix lengths present,
	// which bound the probing.
	MinLen int
	MaxLen int
}

// Lookup returns the value index of the longest prefix of digits present in
// the map, or -1 if none is. digits must be the calling code followed by the
// national number, with no '+' and no separators.
func (m *Map) Lookup(digits string) int {
	if len(m.Prefixes) == 0 {
		return -1
	}
	high := m.MaxLen
	if len(digits) < high {
		high = len(digits)
	}
	// The longest prefix wins, so probe downwards and stop at the first hit.
	for l := high; l >= m.MinLen; l-- {
		v, ok := prefixValue(digits, l)
		if !ok {
			return -1
		}
		if i := m.search(v); i >= 0 {
			return int(m.Values[i])
		}
	}
	return -1
}

// search returns the index of v in Prefixes, or -1.
func (m *Map) search(v uint32) int {
	lo, hi := 0, len(m.Prefixes)-1
	for lo <= hi {
		mid := int(uint(lo+hi) >> 1)
		switch p := m.Prefixes[mid]; {
		case p < v:
			lo = mid + 1
		case p > v:
			hi = mid - 1
		default:
			return mid
		}
	}
	return -1
}

// prefixValue reads the first n digits of s as a number. It reports false if
// any of them is not a digit, which means the caller was handed something
// that is not a normalised number.
func prefixValue(s string, n int) (uint32, bool) {
	var v uint32
	for i := 0; i < n; i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		v = v*10 + uint32(c-'0')
	}
	return v, true
}
