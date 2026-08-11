package phonex

import (
	"math/rand"
	"strings"
	"sync"
	"testing"
)

func TestGenerate(t *testing.T) {
	p, ok := Generate("UZ")
	if !ok || p == nil {
		t.Fatal("Generate(UZ) failed")
	}
	if p.Country() != "UZ" {
		t.Errorf("Country() = %v, want UZ", p.Country())
	}
	if !p.IsValid() {
		t.Errorf("generated %v is not valid", p)
	}

	generated := map[string]bool{}
	for i := 0; i < 10; i++ {
		p, ok := Generate("US")
		if !ok || p == nil {
			t.Fatal("Generate(US) failed")
		}
		if !p.IsValid() {
			t.Errorf("generated %v is not valid", p)
		}
		generated[p.String()] = true
	}
	if len(generated) < 2 {
		t.Errorf("10 calls produced %d distinct numbers; the subscriber part is not being randomised", len(generated))
	}

	if _, ok := Generate("INVALID"); ok {
		t.Error("Generate accepted an unknown region")
	}

	fixed, ok := GenerateForType("GB", FixedLine)
	if !ok || fixed == nil {
		t.Fatal("GenerateForType(GB, FixedLine) failed")
	}
	if fixed.Type() != FixedLine && fixed.Type() != FixedLineOrMobile {
		t.Errorf("Type() = %v, want FIXED_LINE", fixed.Type())
	}
}

// TestGenerateWithIsDeterministic is the point of taking the randomness as an
// argument: the same seed has to produce the same numbers, so that a test or
// a fixture built on it can be reproduced.
func TestGenerateWithIsDeterministic(t *testing.T) {
	draw := func(seed int64) []string {
		r := rand.New(rand.NewSource(seed))
		var out []string
		for _, region := range []string{"GB", "US", "UZ", "DE"} {
			p, ok := GenerateWith(region, Mobile, r.Intn)
			if !ok {
				out = append(out, region+":none")
				continue
			}
			out = append(out, p.E164())
		}
		return out
	}

	first, second := draw(1), draw(1)
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("seed 1 produced %q then %q", first, second)
		}
	}
	if other := draw(2); strings.Join(other, ",") == strings.Join(first, ",") {
		t.Error("two different seeds produced the same numbers")
	}
}

func TestGenerateWithAnyType(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for _, region := range SupportedRegions() {
		p, ok := GenerateWith(region, AnyType, r.Intn)
		if !ok {
			// A handful of regions carry no example to build on.
			continue
		}
		if !p.IsValid() {
			t.Errorf("%s: generated %v is not valid", region, p)
		}
	}
}

// TestGenerateWithNilIntn covers the documented fallback to the global
// source, so that a caller who has no RNG of their own still gets a number.
func TestGenerateWithNilIntn(t *testing.T) {
	p, ok := GenerateWith("GB", Mobile, nil)
	if !ok || !p.IsValid() {
		t.Fatalf("GenerateWith(GB, Mobile, nil) = %v, %v", p, ok)
	}
	q, ok := GenerateForPrefix("GB", "20", nil)
	if !ok || !q.IsValid() {
		t.Fatalf("GenerateForPrefix(GB, 20, nil) = %v, %v", q, ok)
	}
}

func TestGenerateForPrefix(t *testing.T) {
	tests := []struct {
		region, prefix string
		// nsnPrefix is what the national number must start with, once the
		// prefix has been read the way the plan writes it.
		nsnPrefix string
	}{
		{"GB", "20", "20"},    // London, eight further digits
		{"GB", "020", "20"},   // the same, written with the trunk digit
		{"GB", "161", "161"},  // Manchester, seven further digits
		{"US", "212", "212"},  // no trunk prefix at all
		{"CA", "416", "416"},  // a region sharing its calling code
		{"IT", "06", "06"},    // Italy keeps the zero in the national number
		{"IT", "6", "06"},     // …so an atlas area code is short by one
		{"DE", "30", "30"},    // Berlin, variable subscriber length
		{"FR", "1", "1"},      // one-digit area code
		{"RU", "495", "495"},  // Moscow
		{"JP", "3", "3"},      // Tokyo
		{"UZ", "93", "93"},    // an operator code rather than an area
		{"UZ", "71", "71"},    // Tashkent
		{"GB", "(020)", "20"}, // punctuation is ignored
	}

	r := rand.New(rand.NewSource(11))
	for _, tt := range tests {
		t.Run(tt.region+"/"+tt.prefix, func(t *testing.T) {
			p, ok := GenerateForPrefix(tt.region, tt.prefix, r.Intn)
			if !ok {
				t.Fatalf("GenerateForPrefix(%q, %q) reported no shape", tt.region, tt.prefix)
			}
			if !p.IsValid() {
				t.Errorf("generated %v is not valid", p)
			}
			if !p.IsValidForRegion(tt.region) {
				t.Errorf("generated %v is not valid for %s", p, tt.region)
			}
			if !strings.HasPrefix(p.NSN(), tt.nsnPrefix) {
				t.Errorf("NSN() = %q, want it to start with %q", p.NSN(), tt.nsnPrefix)
			}
		})
	}
}

func TestGenerateForPrefixRejects(t *testing.T) {
	r := rand.New(rand.NewSource(13))
	tests := []struct{ region, prefix, why string }{
		{"XX", "20", "unknown region"},
		{"GB", "", "empty prefix"},
		{"GB", "   ", "no digits"},
		{"GB", "+20", "a number, not a prefix"},
		{"GB", "99999", "no such area code"},
		{"GB", "12345678901234567890", "longer than any national number"},
	}
	for _, tt := range tests {
		if p, ok := GenerateForPrefix(tt.region, tt.prefix, r.Intn); ok {
			t.Errorf("GenerateForPrefix(%q, %q) returned %v; %s", tt.region, tt.prefix, p, tt.why)
		}
	}
}

// TestGenerateForPrefixRepeats covers the cached shape: the second call must
// answer as well as the first, and a prefix that failed must keep failing
// rather than being remembered as a success.
func TestGenerateForPrefixRepeats(t *testing.T) {
	r := rand.New(rand.NewSource(17))
	for i := 0; i < 5; i++ {
		p, ok := GenerateForPrefix("GB", "20", r.Intn)
		if !ok || !p.IsValid() {
			t.Fatalf("call %d: %v, %v", i, p, ok)
		}
		if !strings.HasPrefix(p.NSN(), "20") {
			t.Fatalf("call %d: NSN() = %q", i, p.NSN())
		}
		if _, ok := GenerateForPrefix("GB", "99999", r.Intn); ok {
			t.Fatalf("call %d: an impossible prefix succeeded", i)
		}
	}
}

// TestGenerateForPrefixConcurrent exercises the shape cache from several
// goroutines, which is where a plain map would be caught.
func TestGenerateForPrefixConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			r := rand.New(rand.NewSource(seed))
			for _, prefix := range []string{"20", "161", "1483", "99999"} {
				p, ok := GenerateForPrefix("GB", prefix, r.Intn)
				if ok && !p.IsValid() {
					t.Errorf("prefix %s: generated %v is not valid", prefix, p)
				}
			}
		}(int64(i))
	}
	wg.Wait()
}

func BenchmarkGenerateForPrefix(b *testing.B) {
	r := rand.New(rand.NewSource(1))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, ok := GenerateForPrefix("GB", "20", r.Intn); !ok {
			b.Fatal("no shape")
		}
	}
}
