package difftest

import (
	"testing"

	"github.com/bakhod1r/phonex"
	"github.com/bakhod1r/phonex/carrier"
	"github.com/bakhod1r/phonex/geocoding"
	"github.com/bakhod1r/phonex/timezone"
	lpn "github.com/nyaruka/phonenumbers"
)

// TestPrefixDataAgreement compares the geocoding, carrier and time zone
// lookups against libphonenumber over the shared corpus.
//
// These three data sets are independent of the number metadata, but they are
// versioned independently too: nyaruka bundles its own copies, and they are
// not always cut from the same upstream commit as ours. Ghana's AirtelTigo
// has since rebranded to "AT" and Norway gained ranges, so a handful of
// carrier names differ. The data files here are copied verbatim from
// libphonenumber v9.0.32 and each disagreement has been checked against them.
func TestPrefixDataAgreement(t *testing.T) {
	cases := corpus(t)

	geo := &mismatches{t: t, name: "geocoding Area"}
	// Carrier names are the one place the two data sets drift, so this check
	// gets its own, still tight, budget.
	carr := &mismatches{t: t, name: "carrier Name", budget: 20}
	tz := &mismatches{t: t, name: "timezone For"}
	checked := 0

	for _, c := range cases {
		p := parseBoth(c)
		if p.ourErr != nil || p.theirE != nil {
			continue
		}
		// The lookups key off the E.164 digits, so a case where the two
		// disagree about those would be reported here as a data mismatch
		// when it is really a parsing one. diff_test.go covers that.
		if p.ours.E164() != lpn.Format(p.theirs, lpn.E164) {
			continue
		}
		checked++

		theirGeo, err := lpn.GetGeocodingForNumber(p.theirs, "en")
		if err == nil {
			// Compare the area lookups only, which is the part that comes
			// from the shared data set. When neither has an area,
			// libphonenumber falls back to a country name taken from CLDR
			// and phonex to the one in its own region table, so the two
			// differ on spelling ("Antigua & Barbuda" against "Antigua and
			// Barbuda") without either being wrong. An empty area on our
			// side means we are in that fallback territory.
			if ours := geocoding.Area(p.ours); ours != "" && ours != theirGeo {
				geo.add(c, "phonex %q, libphonenumber %q (number %s)", ours, theirGeo, p.ours.E164())
			}
		}

		theirCarrier, err := lpn.GetCarrierForNumber(p.theirs, "en")
		if err == nil {
			if ours := carrier.Name(p.ours); ours != theirCarrier {
				carr.add(c, "phonex %q, libphonenumber %q (number %s)", ours, theirCarrier, p.ours.E164())
			}
		}

		theirZones, err := lpn.GetTimezonesForNumber(p.theirs)
		if err == nil {
			if ours := timezone.For(p.ours); !sameZones(ours, theirZones) {
				tz.add(c, "phonex %v, libphonenumber %v (number %s)", ours, theirZones, p.ours.E164())
			}
		}
	}

	geo.report(checked)
	carr.report(checked)
	tz.report(checked)
}

// sameZones compares two zone lists, treating the "Etc/Unknown" placeholder
// libphonenumber returns for an unknown prefix as the absence of an answer,
// which is how phonex reports it.
func sameZones(ours, theirs []string) bool {
	if len(theirs) == 1 && theirs[0] == "Etc/Unknown" {
		theirs = nil
	}
	if len(ours) != len(theirs) {
		return false
	}
	for i := range ours {
		if ours[i] != theirs[i] {
			return false
		}
	}
	return true
}

// TestPrefixDataSizes guards against a generated table being emptied by a bad
// regeneration, which would make every comparison above trivially agree.
func TestPrefixDataSizes(t *testing.T) {
	if got := timezone.Count(); got < 3000 {
		t.Errorf("time zone data has only %d prefixes", got)
	}
	if got := carrier.Count(); got < 25000 {
		t.Errorf("carrier data has only %d prefixes", got)
	}
	if got := geocoding.Count(); got < 250000 {
		t.Errorf("geocoding data has only %d prefixes", got)
	}
}

// TestKnownPrefixData spot-checks answers a reader can verify without
// consulting the data files.
func TestKnownPrefixData(t *testing.T) {
	tests := []struct {
		number, area, zone string
	}{
		{"+1 212 555 0123", "New York, NY", "America/New_York"},
		{"+44 20 7031 3000", "London", "Europe/London"},
		{"+81 3 3224 9999", "Tokyo", "Asia/Tokyo"},
		{"+86 10 6552 9988", "Beijing", "Asia/Shanghai"},
		{"+7 495 123 45 67", "Moscow", "Europe/Moscow"},
	}
	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			p, err := phonex.Parse(tt.number)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := geocoding.Area(p); got != tt.area {
				t.Errorf("Area() = %q, want %q", got, tt.area)
			}
			zones := timezone.For(p)
			if len(zones) != 1 || zones[0] != tt.zone {
				t.Errorf("For() = %v, want [%s]", zones, tt.zone)
			}
		})
	}
}
