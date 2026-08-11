package geocoding_test

import (
	"fmt"

	"github.com/bakhod1r/phonex"
	"github.com/bakhod1r/phonex/geocoding"
)

func ExampleArea() {
	p, _ := phonex.Parse("+44 20 7031 3000")
	fmt.Println(geocoding.Area(p))

	// Mobile ranges are not tied to a place, so Area is empty and Describe
	// falls back to the country.
	q, _ := phonex.Parse("+44 7400 123456")
	fmt.Printf("%q\n", geocoding.Area(q))
	fmt.Println(geocoding.Describe(q))
	// Output:
	// London
	// ""
	// United Kingdom
}
