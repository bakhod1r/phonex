package carrier

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/bakhod1r/phonex"
)

func TestName(t *testing.T) {
	tests := []struct {
		number string
		want   string
	}{
		{"+44 7400 123456", "Three"},
		{"+998 90 123 45 67", "Beeline"},
		// Landline ranges carry no carrier.
		{"+44 20 7031 3000", ""},
	}
	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			p, err := phonex.Parse(tt.number)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := Name(p); got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSafeDisplayName covers the guard that matters: in a country with number
// portability the prefix no longer identifies the network, so no name is
// returned however confident the data looks.
func TestSafeDisplayName(t *testing.T) {
	portable, err := phonex.Parse("+44 7400 123456")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !portable.MobileNumberPortable() {
		t.Fatal("GB should be a number-portable region")
	}
	if Name(portable) == "" {
		t.Fatal("the test needs a number the data does have a carrier for")
	}
	if got := SafeDisplayName(portable); got != "" {
		t.Errorf("SafeDisplayName() = %q, want empty in a portable region", got)
	}

	fixed, err := phonex.Parse("+998 90 123 45 67")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if fixed.MobileNumberPortable() {
		t.Skip("UZ has become a number-portable region; pick another example")
	}
	if got, want := SafeDisplayName(fixed), "Beeline"; got != want {
		t.Errorf("SafeDisplayName() = %q, want %q", got, want)
	}
}

func TestSafeDisplayNameRejectsInvalid(t *testing.T) {
	// A number that is not valid must never produce a carrier name, even
	// where the prefix happens to match.
	p, err := phonex.Parse("+998 90 000 00 00")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.IsValid() {
		t.Skip("the example number has become valid; pick another")
	}
	if got := SafeDisplayName(p); got != "" {
		t.Errorf("SafeDisplayName() = %q for an invalid number", got)
	}
}

func TestNilAndUnknown(t *testing.T) {
	if got := Name(nil); got != "" {
		t.Errorf("Name(nil) = %q", got)
	}
	if got := SafeDisplayName(nil); got != "" {
		t.Errorf("SafeDisplayName(nil) = %q", got)
	}
	if got := NameForDigits(""); got != "" {
		t.Errorf("NameForDigits(\"\") = %q", got)
	}
	if got := NameForDigits("zzz"); got != "" {
		t.Errorf("NameForDigits(non-digits) = %q", got)
	}
}

func TestNameForNumber(t *testing.T) {
	got, err := NameForNumber("+44 7400 123456")
	if err != nil {
		t.Fatalf("NameForNumber: %v", err)
	}
	if got != "Three" {
		t.Errorf("NameForNumber() = %q", got)
	}
	if _, err := NameForNumber("nonsense"); err == nil {
		t.Error("NameForNumber should reject an unparsable number")
	}
}

func TestValuesAreNonEmpty(t *testing.T) {
	for i, v := range values {
		if v == "" {
			t.Fatalf("value %d is empty; empty values should not be generated", i)
		}
	}
}

func TestCount(t *testing.T) {
	if got := Count(); got < 25000 {
		t.Errorf("Count() = %d, want the full data set", got)
	}
}

func TestMatchesVendoredSource(t *testing.T) {
	hash := sha256.New()
	names, err := filepath.Glob(filepath.Join("..", "internal", "metadata", "carrier", "en", "*.txt"))
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
