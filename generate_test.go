package phonex

import (
	"testing"
)

func TestGenerate(t *testing.T) {
	// Generate for UZ
	p, ok := Generate("UZ")
	if !ok || p == nil {
		t.Fatal("Failed to generate for UZ")
	}
	if p.Country() != "UZ" {
		t.Errorf("Expected UZ, got %v", p.Country())
	}
	if !p.IsValid() {
		t.Errorf("Generated number is not valid: %v", p.String())
	}

	// Generate multiple times to ensure randomness (or at least no panics)
	var generated = make(map[string]bool)
	for i := 0; i < 10; i++ {
		p, ok = Generate("US")
		if !ok || p == nil {
			t.Fatal("Failed to generate for US")
		}
		generated[p.String()] = true
	}

	if len(generated) < 2 {
		t.Logf("Warning: Generator might not be random enough, or only 1 example exists. Unique values: %v", len(generated))
	}

	// Invalid region
	_, ok = Generate("INVALID")
	if ok {
		t.Error("Expected failure for invalid region")
	}

	// FixedLine specific
	pFix, ok := GenerateForType("GB", FixedLine)
	if !ok || pFix == nil {
		t.Fatal("Failed to generate FixedLine for GB")
	}
	if pFix.Type() != FixedLine && pFix.Type() != FixedLineOrMobile {
		t.Errorf("Expected FixedLine or FixedLineOrMobile, got %v", pFix.Type())
	}
}
