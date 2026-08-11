package phonex

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct{ in, want string }{
		{"+998 (90) 123-45-67", "+998901234567"},
		{"998901234567", "998901234567"},
		{"(202) 555-0123", "2025550123"},
		{"", ""},
		{"+ + 1", "+1"},
	}
	for _, tt := range tests {
		if got := Normalize(tt.in); got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
		if got := string(NormalizeBytes([]byte(tt.in))); got != tt.want {
			t.Errorf("NormalizeBytes(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestToE164(t *testing.T) {
	got, err := ToE164("90 123 45 67", WithDefaultCountry("UZ"))
	if err != nil {
		t.Fatalf("ToE164: %v", err)
	}
	if want := "+998901234567"; got != want {
		t.Errorf("ToE164() = %q, want %q", got, want)
	}
	if _, err := ToE164("nonsense"); err == nil {
		t.Error("ToE164 should reject an unparsable number")
	}
}
