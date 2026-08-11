package phonex

import (
	"database/sql/driver"
	"fmt"
)

// Scan reads a number from a database column holding text. A NULL column
// leaves the Phone untouched.
func (p *Phone) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		return p.Parse(v)
	case []byte:
		return p.ParseBytes(v)
	default:
		return fmt.Errorf("phonex: cannot scan %T into Phone", value)
	}
}

// Value writes the number to a database column as its E.164 string.
func (p Phone) Value() (driver.Value, error) {
	return p.E164(), nil
}
