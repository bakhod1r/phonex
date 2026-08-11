package phonex

import (
	"encoding/json"
	"testing"
)

func TestJSONRoundTrip(t *testing.T) {
	type record struct {
		Phone *Phone `json:"phone"`
	}

	in := record{}
	if err := json.Unmarshal([]byte(`{"phone":"+998 90 123 45 67"}`), &in); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got, want := in.Phone.E164(), "+998901234567"; got != want {
		t.Errorf("E164() = %q, want %q", got, want)
	}

	out, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(out), `{"phone":"+998901234567"}`; got != want {
		t.Errorf("Marshal = %s, want %s", got, want)
	}
}

func TestJSONRejectsInvalid(t *testing.T) {
	var p Phone
	if err := json.Unmarshal([]byte(`"nonsense"`), &p); err == nil {
		t.Error("Unmarshal should reject an unparsable number")
	}
	if err := json.Unmarshal([]byte(`42`), &p); err == nil {
		t.Error("Unmarshal should reject a non-string")
	}
}
