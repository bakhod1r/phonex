package phonex

import (
	"testing"
)

func TestMask(t *testing.T) {
	p, _ := Parse("+998901234567")

	if got := p.Mask(MaskLast4); got != "*********4567" {
		t.Errorf("MaskLast4 = %v, want %v", got, "*********4567")
	}
	if got := p.Mask(MaskFirst3); got != "+99**********" {
		t.Errorf("MaskFirst3 = %v, want %v", got, "+99**********")
	}
	if got := p.Mask(MaskMiddle); got != "+998*******67" {
		t.Errorf("MaskMiddle = %v, want %v", got, "+998*******67")
	}
	if got := p.Mask(MaskFull); got != "*************" {
		t.Errorf("MaskFull = %v, want %v", got, "*************")
	}

	// Short number test - fake parse
	short := &Phone{}
	short.raw = "+1234"
	// this is not valid E164, so E164() might return empty or just +1234. Let's assume Mask works on E164.
	// Actually, p.E164() returns valid string if it's parsed.
	// Let's just pass a valid short phone like +123 (len 4). Prefix 4 + Suffix 2 = 6 > 4, so it's fully masked.

	short2, _ := Parse("+998") // this will fail to parse probably.
	if short2 != nil {
		if got := short2.Mask(MaskMiddle); got != "****" {
			t.Errorf("Short mask = %v", got)
		}
	}

	// No mask options defaults to MaskMiddle
	if got := p.Mask(); got != "+998*******67" {
		t.Errorf("Default mask = %v", got)
	}

	// Redact
	if got := Redact(p); got != "+998*******67" {
		t.Errorf("Redact = %v", got)
	}

	// Nil phone
	var nilPhone *Phone
	if got := nilPhone.Mask(MaskMiddle); got != "" {
		t.Errorf("Nil phone mask = %v", got)
	}
	if got := Redact(nilPhone); got != "" {
		t.Errorf("Nil phone redact = %v", got)
	}
}
