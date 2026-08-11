package shortnumber_test

import (
	"fmt"

	"github.com/bakhod1r/phonex/shortnumber"
)

func ExampleIsEmergency() {
	fmt.Println(shortnumber.IsEmergency("112", "GB"))
	fmt.Println(shortnumber.IsEmergency("911", "US"))
	// The same digits mean nothing in Uzbekistan, which dials 01, 02 and 03.
	fmt.Println(shortnumber.IsEmergency("112", "UZ"))
	fmt.Println(shortnumber.IsEmergency("02", "UZ"))
	// Output:
	// true
	// true
	// false
	// true
}

func ExampleConnectsToEmergency() {
	// Networks act on the emergency prefix, so trailing digits still connect.
	fmt.Println(shortnumber.IsEmergency("911123", "US"))
	fmt.Println(shortnumber.ConnectsToEmergency("911123", "US"))
	// Brazil is one of three countries where the match must be exact.
	fmt.Println(shortnumber.ConnectsToEmergency("190123", "BR"))
	// Output:
	// false
	// true
	// false
}

func ExampleIsValid() {
	fmt.Println(shortnumber.IsValid("100", "GB"))  // the BT operator
	fmt.Println(shortnumber.IsValid("1234", "GB")) // right length, unassigned
	// Output:
	// true
	// false
}

func ExampleExpectedCost() {
	fmt.Println(shortnumber.ExpectedCost("911", "US"))
	fmt.Println(shortnumber.ExpectedCost("10086", "CN"))
	// Output:
	// TOLL_FREE
	// STANDARD_RATE
}
