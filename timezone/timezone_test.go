package timezone

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/bakhod1r/phonex"
)

func TestFor(t *testing.T) {
	tests := []struct {
		number string
		want   []string
	}{
		{"+1 212 555 0123", []string{"America/New_York"}},
		{"+44 20 7031 3000", []string{"Europe/London"}},
		{"+81 3 3224 9999", []string{"Asia/Tokyo"}},
		{"+998 90 123 45 67", []string{"Asia/Tashkent"}},
		{"+7 495 123 45 67", []string{"Europe/Moscow"}},
	}
	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			p, err := phonex.Parse(tt.number)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			got := For(p)
			if len(got) != len(tt.want) {
				t.Fatalf("For() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("For()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestSharedPrefixReturnsSeveralZones covers the case callers forget: a
// calling code that spans a continent has no single zone.
func TestSharedPrefixReturnsSeveralZones(t *testing.T) {
	// +44 7 mobile numbers cover the Crown dependencies as well as Britain.
	p, err := phonex.Parse("+44 7400 123456")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := For(p); len(got) < 2 {
		t.Errorf("For() = %v, want several zones", got)
	}
}

func TestForNilAndUnknown(t *testing.T) {
	if got := For(nil); got != nil {
		t.Errorf("For(nil) = %v, want nil", got)
	}
	if got := ForDigits(""); got != nil {
		t.Errorf("ForDigits(\"\") = %v, want nil", got)
	}
	if got := ForDigits("abc"); got != nil {
		t.Errorf("ForDigits(non-digits) = %v, want nil", got)
	}
}

func TestForNumber(t *testing.T) {
	got, err := ForNumber("+1 212 555 0123")
	if err != nil {
		t.Fatalf("ForNumber: %v", err)
	}
	if len(got) != 1 || got[0] != "America/New_York" {
		t.Errorf("ForNumber() = %v", got)
	}
	if _, err := ForNumber("nonsense"); err == nil {
		t.Error("ForNumber should reject an unparsable number")
	}
}

// TestZonesAreWellFormed checks the whole table: every zone name must look
// like an IANA identifier, since callers feed these straight to
// time.LoadLocation.
func TestZonesAreWellFormed(t *testing.T) {
	for i, set := range values {
		if len(set) == 0 {
			t.Fatalf("value %d is an empty zone set", i)
		}
		for _, zone := range set {
			if zone == "" {
				t.Errorf("value %d holds an empty zone name", i)
				continue
			}
			for j := 0; j < len(zone); j++ {
				switch c := zone[j]; {
				case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z',
					c >= '0' && c <= '9', c == '/', c == '_', c == '-', c == '+':
				default:
					t.Errorf("zone %q holds an unexpected character %q", zone, c)
				}
			}
		}
	}
}

func TestCount(t *testing.T) {
	if got := Count(); got < 3000 {
		t.Errorf("Count() = %d, want the full data set", got)
	}
}

// TestMatchesVendoredSource confirms the table was generated from the data in
// the tree.
func TestMatchesVendoredSource(t *testing.T) {
	hash := sha256.New()
	names, err := filepath.Glob(filepath.Join("..", "internal", "metadata", "timezones", "*.txt"))
	if err != nil || len(names) == 0 {
		t.Skipf("vendored data not readable: %v", err)
	}
	for _, name := range names {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Skipf("vendored data not readable: %v", err)
		}
		hash.Write([]byte(filepath.Base(name)))
		hash.Write(raw)
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != SourceHash {
		t.Fatalf("generated.go was built from different data\n vendored: %s\ngenerated: %s\nrun `make generate`", got, SourceHash)
	}
}
