package phonex

import "testing"

func TestTextRoundTrip(t *testing.T) {
	var p Phone
	if err := p.UnmarshalText([]byte("+998 90 123 45 67")); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	b, err := p.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if got, want := string(b), "+998901234567"; got != want {
		t.Errorf("MarshalText = %q, want %q", got, want)
	}
	if err := p.UnmarshalText([]byte("nonsense")); err == nil {
		t.Error("UnmarshalText should reject an unparsable number")
	}
}
