package phonex

import "testing"

func TestSQLScanAndValue(t *testing.T) {
	var p Phone
	if err := p.Scan("+998 90 123 45 67"); err != nil {
		t.Fatalf("Scan(string): %v", err)
	}
	if got, want := p.E164(), "+998901234567"; got != want {
		t.Errorf("E164() = %q, want %q", got, want)
	}

	if err := p.Scan([]byte("+12025550123")); err != nil {
		t.Fatalf("Scan([]byte): %v", err)
	}
	v, err := p.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if v != "+12025550123" {
		t.Errorf("Value() = %v, want +12025550123", v)
	}

	if err := p.Scan(42); err == nil {
		t.Error("Scan should reject an unsupported type")
	}
	if err := p.Scan("nonsense"); err == nil {
		t.Error("Scan should reject an unparsable number")
	}
}

func TestSQLScanNull(t *testing.T) {
	var p Phone
	if err := p.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if p.E164() != "" {
		t.Errorf("a NULL scan should leave the number empty, got %q", p.E164())
	}
}
