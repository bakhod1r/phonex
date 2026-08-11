package phonex_test

import (
	"fmt"

	"github.com/bakhod1r/phonex"
)

func ExampleParse() {
	p, err := phonex.Parse("+998 90 123 45 67")
	if err != nil {
		panic(err)
	}
	fmt.Println(p.E164(), p.Country(), p.Type(), p.IsValid())
	// Output: +998901234567 UZ MOBILE true
}

func ExampleWithDefaultCountry() {
	p, err := phonex.Parse("(202) 555-0123", phonex.WithDefaultCountry("US"))
	if err != nil {
		panic(err)
	}
	fmt.Println(p.E164())
	// Output: +12025550123
}

func ExamplePhone_Format() {
	p, _ := phonex.Parse("+442070313000")
	fmt.Println(p.E164())
	fmt.Println(p.International())
	fmt.Println(p.National())
	fmt.Println(p.RFC3966())
	fmt.Println(p.OutOfCountry("US"))
	// Output:
	// +442070313000
	// +44 20 7031 3000
	// 020 7031 3000
	// tel:+44-20-7031-3000
	// 011 44 20 7031 3000
}

func ExamplePhone_IsValid() {
	// A number can have a plausible shape without being an assigned number.
	possible, _ := phonex.Parse("+1 000 000 0000")
	fmt.Println(possible.IsPossible(), possible.IsValid())
	// Output: true false
}

func ExampleNewFormatter() {
	f := phonex.NewFormatter("US")
	for _, r := range "2025550123" {
		fmt.Println(f.InputDigit(r))
	}
	// Output:
	// 2
	// 20
	// 202
	// 202-5
	// 202-55
	// 202-555
	// 202-5550
	// (202) 555-01
	// (202) 555-012
	// (202) 555-0123
}

func ExampleMatchNumbers() {
	// The same number written nationally and internationally.
	fmt.Println(phonex.MatchNumbers("901234567", "+998901234567", phonex.WithDefaultCountry("UZ")))
	fmt.Println(phonex.MatchNumbers("+998901234567", "+998901234568"))
	// Output:
	// EXACT_MATCH
	// NO_MATCH
}

func ExamplePhone_ParseWith() {
	// Reusing a Phone and an option struct keeps a hot loop allocation-free.
	var p phonex.Phone
	opts := phonex.DefaultParseOptions()
	opts.DefaultCountry = "GB"
	opts.KeepRawInput = false

	for _, s := range []string{"020 7031 3000", "07400 123456"} {
		if err := p.ParseWith(s, opts); err != nil {
			continue
		}
		fmt.Println(p.E164(), p.Type())
	}
	// Output:
	// +442070313000 FIXED_LINE
	// +447400123456 MOBILE
}
