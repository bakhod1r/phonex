package phonex

import "testing"

// FuzzParse checks that no input can panic the parser and that whatever it
// accepts survives a round trip through E.164.
func FuzzParse(f *testing.F) {
	seeds := []string{
		"+998901234567", "020 7031 3000", "tel:+1-202-555-0123;ext=42",
		"1-800-FLOWERS", "+", "00", "+1", ";;;", "+800 1234 5678",
		"011 44 20 7031 3000", "\x00\x01", "+44(0)20 7031 3000",
	}
	for _, s := range seeds {
		f.Add(s, "US")
	}
	f.Fuzz(func(t *testing.T, input, region string) {
		var p Phone
		opts := DefaultParseOptions()
		opts.DefaultCountry = normalizeRegion(region)
		if _, ok := Country(opts.DefaultCountry); !ok {
			opts.DefaultCountry = ""
		}
		opts.AllowAlpha = true

		if err := p.ParseWith(input, opts); err != nil {
			return
		}
		e164 := p.E164()
		if e164 == "" {
			t.Fatalf("Parse(%q) succeeded but E164() is empty", input)
		}

		// Every accepted number must re-parse.
		var q Phone
		if err := q.Parse(e164); err != nil {
			t.Fatalf("Parse(%q) gave %q, which fails to re-parse: %v", input, e164, err)
		}
		// E.164 is only a stable identity for numbers that are valid. A
		// number that is not tells us nothing about where its digits end and
		// a trunk prefix begins, so "+358 0000000" legitimately normalises
		// to "+358 000000" on the way back in. Valid numbers never start
		// their national number with the region's trunk prefix, so they are
		// unaffected; TestValidNumbersRoundTripStably covers that at scale.
		if p.IsValid() && q.E164() != e164 {
			t.Fatalf("round trip of valid %q: %q became %q", input, e164, q.E164())
		}
		// The formatters must not panic or lose digits.
		for _, s := range []string{p.National(), p.International(), p.RFC3966()} {
			if s == "" {
				t.Fatalf("empty formatting for %q", e164)
			}
		}
	})
}
