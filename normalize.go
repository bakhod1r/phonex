package phonex

import "strings"

// Normalize strips punctuation and spacing from input, keeping the digits and
// a leading '+'. It does not parse or validate: use E164 for a canonical
// number, and this for cheap cleanup of an input field.
func Normalize(input string) string {
	var b strings.Builder
	b.Grow(len(input))
	for i := 0; i < len(input); i++ {
		c := input[i]
		switch {
		case c >= '0' && c <= '9':
			b.WriteByte(c)
		case c == '+' && b.Len() == 0:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// NormalizeBytes is Normalize for a byte slice. The result is a fresh slice.
func NormalizeBytes(input []byte) []byte {
	out := make([]byte, 0, len(input))
	for _, c := range input {
		switch {
		case c >= '0' && c <= '9':
			out = append(out, c)
		case c == '+' && len(out) == 0:
			out = append(out, c)
		}
	}
	return out
}

// ToE164 parses input and returns its canonical E.164 form, or an error if it
// cannot be parsed. It is the one-call form of Parse followed by E164.
func ToE164(input string, opts ...ParseOption) (string, error) {
	var p Phone
	if err := p.Parse(input, opts...); err != nil {
		return "", err
	}
	return p.E164(), nil
}
