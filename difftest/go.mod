// Module difftest holds the differential tests that compare phonex against
// Google's libphonenumber, through the nyaruka/phonenumbers port of it.
//
// It is a separate module so that the phonex module itself stays free of
// dependencies. Run it with:
//
//	cd difftest && go test ./...
module github.com/bakhod1r/phonex/difftest

go 1.23.0

require (
	github.com/bakhod1r/phonex v0.0.0
	github.com/nyaruka/phonenumbers v1.8.1
)

require (
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/bakhod1r/phonex => ../
