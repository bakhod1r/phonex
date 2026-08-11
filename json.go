package phonex

import "encoding/json"

// MarshalJSON encodes the number as its E.164 string.
func (p *Phone) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.E164())
}

// UnmarshalJSON parses a JSON string into the number. Anything that does not
// parse is rejected, so a Phone field can never hold an invalid value.
func (p *Phone) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return p.Parse(s)
}
