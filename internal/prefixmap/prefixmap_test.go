package prefixmap

import "testing"

// A small map standing in for the generated ones: prefixes 1, 12, 1234 and 9.
var testMap = Map{
	Prefixes: []uint32{1, 9, 12, 1234},
	Values:   []uint32{10, 90, 120, 12340},
	MinLen:   1,
	MaxLen:   4,
}

func TestLookupPrefersTheLongestMatch(t *testing.T) {
	tests := []struct {
		digits string
		want   int
	}{
		{"1234567", 12340}, // the four-digit prefix wins
		{"123", 120},       // falls back to "12"
		{"12", 120},
		{"1", 10},
		{"19", 10}, // "19" is absent, so "1" answers
		{"98765", 90},
		{"5", -1},
		{"", -1},
	}
	for _, tt := range tests {
		t.Run(tt.digits, func(t *testing.T) {
			if got := testMap.Lookup(tt.digits); got != tt.want {
				t.Errorf("Lookup(%q) = %d, want %d", tt.digits, got, tt.want)
			}
		})
	}
}

func TestLookupRejectsNonDigits(t *testing.T) {
	if got := testMap.Lookup("1a34"); got != -1 {
		t.Errorf("Lookup with a non-digit = %d, want -1", got)
	}
}

func TestEmptyMap(t *testing.T) {
	var m Map
	if got := m.Lookup("123"); got != -1 {
		t.Errorf("Lookup on an empty map = %d, want -1", got)
	}
}

func TestMinLenIsRespected(t *testing.T) {
	// A map whose shortest prefix is three digits must not answer for two.
	m := Map{Prefixes: []uint32{123}, Values: []uint32{7}, MinLen: 3, MaxLen: 3}
	if got := m.Lookup("12"); got != -1 {
		t.Errorf("Lookup(\"12\") = %d, want -1", got)
	}
	if got := m.Lookup("1234"); got != 7 {
		t.Errorf("Lookup(\"1234\") = %d, want 7", got)
	}
}

func BenchmarkLookup(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = testMap.Lookup("1234567")
	}
}
