package phonex

// MarshalText encodes the number as its E.164 string, which makes Phone usable
// as a map key in encoding/json and as a value in any text-based encoder.
func (p *Phone) MarshalText() ([]byte, error) {
	return []byte(p.E164()), nil
}

// UnmarshalText parses the number from its text form.
func (p *Phone) UnmarshalText(text []byte) error {
	return p.ParseBytes(text)
}
