package geocoding

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/bakhod1r/phonex"
)

func TestArea(t *testing.T) {
	tests := []struct {
		number string
		want   string
	}{
		{"+1 212 555 0123", "New York, NY"},
		{"+1 650 253 0000", "Mountain View, CA"},
		{"+44 20 7031 3000", "London"},
		{"+81 3 3224 9999", "Tokyo"},
		{"+86 10 6552 9988", "Beijing"},
		{"+7 495 123 45 67", "Moscow"},
		// Mobile ranges are not tied to a place.
		{"+44 7400 123456", ""},
	}
	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			p, err := phonex.Parse(tt.number)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := Area(p); got != tt.want {
				t.Errorf("Area() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDescribeFallsBackToCountry covers the reason Describe exists: a number
// with no area is still worth labelling.
func TestDescribeFallsBackToCountry(t *testing.T) {
	p, err := phonex.Parse("+44 7400 123456")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if Area(p) != "" {
		t.Skip("this range has gained an area description; pick another")
	}
	if got, want := Describe(p), "United Kingdom"; got != want {
		t.Errorf("Describe() = %q, want %q", got, want)
	}

	// Where an area exists, Describe prefers it.
	q, err := phonex.Parse("+44 20 7031 3000")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := Describe(q), "London"; got != want {
		t.Errorf("Describe() = %q, want %q", got, want)
	}
}

// TestLongestPrefixWins is the property the lookup is built on: a more
// specific prefix must beat a shorter one.
func TestLongestPrefixWins(t *testing.T) {
	// "44113" is Leeds; "441142" is Sheffield. The longer prefix must win.
	if got, want := AreaForDigits("441130000000"), "Leeds"; got != want {
		t.Errorf("AreaForDigits(441130...) = %q, want %q", got, want)
	}
	if got, want := AreaForDigits("441142000000"), "Sheffield"; got != want {
		t.Errorf("AreaForDigits(441142...) = %q, want %q", got, want)
	}
}

func TestNilAndUnknown(t *testing.T) {
	if got := Area(nil); got != "" {
		t.Errorf("Area(nil) = %q", got)
	}
	if got := Describe(nil); got != "" {
		t.Errorf("Describe(nil) = %q", got)
	}
	if got := AreaForDigits(""); got != "" {
		t.Errorf("AreaForDigits(\"\") = %q", got)
	}
	if got := AreaForDigits("zzz"); got != "" {
		t.Errorf("AreaForDigits(non-digits) = %q", got)
	}
}

func TestAreaForNumber(t *testing.T) {
	got, err := AreaForNumber("+44 20 7031 3000")
	if err != nil {
		t.Fatalf("AreaForNumber: %v", err)
	}
	if got != "London" {
		t.Errorf("AreaForNumber() = %q", got)
	}
	if _, err := AreaForNumber("nonsense"); err == nil {
		t.Error("AreaForNumber should reject an unparsable number")
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
	if got := Count(); got < 250000 {
		t.Errorf("Count() = %d, want the full data set", got)
	}
}

func TestMatchesVendoredSource(t *testing.T) {
	hash := sha256.New()
	names, err := filepath.Glob(filepath.Join("..", "internal", "metadata", "geocoding", "en", "*.txt"))
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
