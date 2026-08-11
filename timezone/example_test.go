package timezone_test

import (
	"fmt"

	"github.com/bakhod1r/phonex"
	"github.com/bakhod1r/phonex/timezone"
)

func ExampleFor() {
	p, _ := phonex.Parse("+1 212 555 0123")
	fmt.Println(timezone.For(p))

	// A prefix that spans several zones returns all of them.
	q, _ := phonex.Parse("+44 7400 123456")
	fmt.Println(timezone.For(q))
	// Output:
	// [America/New_York]
	// [Europe/Guernsey Europe/Isle_of_Man Europe/London]
}
