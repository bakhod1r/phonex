package carrier_test

import (
	"fmt"

	"github.com/bakhod1r/phonex"
	"github.com/bakhod1r/phonex/carrier"
)

func ExampleName() {
	p, _ := phonex.Parse("+44 7400 123456")
	fmt.Println(carrier.Name(p))

	// The name is the network the range was issued to. Britain has number
	// portability, so it is not necessarily the network today, and
	// SafeDisplayName withholds it.
	fmt.Printf("%q\n", carrier.SafeDisplayName(p))
	// Output:
	// Three
	// ""
}
