// Package phonex parses, validates and formats international phone numbers.
//
// The metadata behind it is generated directly from Google's libphonenumber
// PhoneNumberMetadata.xml, so the number ranges, formatting rules and trunk
// prefixes are the same ones the reference implementation uses.
//
// # Parsing
//
// Parse accepts numbers in international form, in national form when a
// default region is given, and in RFC 3966 form:
//
//	p, err := phonex.Parse("+998 90 123 45 67")
//	p, err := phonex.Parse("90 123 45 67", phonex.WithDefaultCountry("UZ"))
//	p, err := phonex.Parse("tel:+1-202-555-0123;ext=42")
//
// Parse enforces only the bounds E.164 sets for any number. It does not judge
// the number against its own country's rules, so a wrong-length or unassigned
// number still parses and the checks below are what report on it. Gate on
// IsValid before trusting a parsed number.
//
//	p.IsPossible()  // the length is one the numbering plan uses
//	p.IsValid()     // the number falls in an assigned range
//	p.Possibility() // why a length check failed
//
// # Formatting
//
//	p.E164()             // +12025550123
//	p.International()    // +1 202-555-0123
//	p.National()         // (202) 555-0123
//	p.RFC3966()          // tel:+1-202-555-0123
//	p.OutOfCountry("GB") // 00 1 202-555-0123
//
// Formatter formats a number while it is being typed, for use in an input
// field:
//
//	f := phonex.NewFormatter("US")
//	f.Input("2025550123") // (202) 555-0123
//
// # Allocation behaviour
//
// A Phone stores its digits inline, so parsing into an existing value does no
// allocation at all for numbers in international form. In a hot loop, reuse a
// Phone and pass options as a struct:
//
//	var p phonex.Phone
//	opts := phonex.DefaultParseOptions()
//	opts.DefaultCountry = "UZ"
//	for _, s := range numbers {
//	    if err := p.ParseWith(s, opts); err == nil {
//	        use(p.E164())
//	    }
//	}
//
// # Short numbers
//
// Short numbers such as 112 and 911 are a different problem: they have no
// international form and the same digits mean different things in different
// countries. Parse rejects them as too short. They live in the shortnumber
// subpackage, which keeps its own metadata so that programs which never need
// it do not link it in:
//
//	shortnumber.IsEmergency("112", "GB")        // true
//	shortnumber.ConnectsToEmergency("911123", "US") // true
//
// # Where a number is, and whose it is
//
// The geocoding, carrier and timezone subpackages answer questions the core
// metadata cannot. They key off the number's prefix, so they describe where
// and how the number was issued, not where its owner is now:
//
//	geocoding.Area(p)          // "London"
//	timezone.For(p)            // ["Europe/London"]
//	carrier.SafeDisplayName(p) // "" where number portability makes it unreliable
//
// Each carries its own data and is a separate package so that a program which
// does not need it does not link it in: geocoding alone is several megabytes.
//
// # Regenerating the metadata
//
// countries/generated.go is produced by cmd/phonexgen from the vendored
// upstream XML, pinned to a tagged libphonenumber release. Refresh it with:
//
//	go generate ./...
//
// The difftest module compares this package against libphonenumber itself
// over the whole metadata; see difftest/ and "make diff".
package phonex

//go:generate go run ./cmd/phonexgen -xml internal/metadata/PhoneNumberMetadata.xml -regions internal/metadata/metadata.json -out countries/generated.go
