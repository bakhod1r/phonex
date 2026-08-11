package phonex

import "github.com/bakhod1r/phonex/countries"

// Metadata describes one calling region. It is always handled through a
// pointer: it carries lazily compiled patterns and must not be copied.
type Metadata = countries.Metadata

// PhoneType identifies the range a number falls into.
type PhoneType = countries.PhoneType

// Number ranges, mirroring libphonenumber's PhoneNumberType.
const (
	FixedLine         = countries.FixedLine
	Mobile            = countries.Mobile
	TollFree          = countries.TollFree
	PremiumRate       = countries.PremiumRate
	SharedCost        = countries.SharedCost
	VoIP              = countries.VoIP
	PersonalNumber    = countries.PersonalNumber
	Pager             = countries.Pager
	UAN               = countries.UAN
	Voicemail         = countries.Voicemail
	FixedLineOrMobile = countries.FixedLineOrMobile
	Unknown           = countries.Unknown
)

// CountryCodeSource records how the calling code was determined while parsing.
type CountryCodeSource uint8

const (
	// FromDefaultCountry means the number carried no calling code and the
	// default region supplied it.
	FromDefaultCountry CountryCodeSource = iota
	// FromNumberWithPlusSign means the number started with '+'.
	FromNumberWithPlusSign
	// FromNumberWithIDD means the number started with the default region's
	// international dialling prefix, e.g. "00" or "011".
	FromNumberWithIDD
	// FromNumberWithoutPlusSign means the number started with the calling
	// code but no '+' and no IDD.
	FromNumberWithoutPlusSign
)

func (s CountryCodeSource) String() string {
	switch s {
	case FromNumberWithPlusSign:
		return "FROM_NUMBER_WITH_PLUS_SIGN"
	case FromNumberWithIDD:
		return "FROM_NUMBER_WITH_IDD"
	case FromNumberWithoutPlusSign:
		return "FROM_NUMBER_WITHOUT_PLUS_SIGN"
	default:
		return "FROM_DEFAULT_COUNTRY"
	}
}

// Possibility is the outcome of a length-only check. It distinguishes the
// reasons a number cannot be valid, which callers use to write precise error
// messages without re-deriving them.
type Possibility uint8

const (
	// IsPossibleNumber means the length is valid for the region.
	IsPossibleNumber Possibility = iota
	// IsPossibleLocalOnly means the length is only valid when dialled from
	// inside the same local area.
	IsPossibleLocalOnly
	// InvalidCountryCode means no region uses the calling code.
	InvalidCountryCode
	// TooShort means the number has fewer digits than any valid length.
	TooShort
	// InvalidLength means the digit count falls between two valid lengths.
	InvalidLength
	// TooLong means the number has more digits than any valid length.
	TooLong
)

func (p Possibility) String() string {
	switch p {
	case IsPossibleNumber:
		return "IS_POSSIBLE"
	case IsPossibleLocalOnly:
		return "IS_POSSIBLE_LOCAL_ONLY"
	case InvalidCountryCode:
		return "INVALID_COUNTRY_CODE"
	case TooShort:
		return "TOO_SHORT"
	case InvalidLength:
		return "INVALID_LENGTH"
	case TooLong:
		return "TOO_LONG"
	default:
		return "UNKNOWN"
	}
}

// MatchType grades how closely two numbers correspond.
type MatchType uint8

const (
	// NoMatch means the numbers cannot be the same.
	NoMatch MatchType = iota
	// ShortNSNMatch means one national number is a suffix of the other, but
	// neither carries enough context to be sure.
	ShortNSNMatch
	// NSNMatch means the national numbers are equal but the calling codes
	// could not both be confirmed.
	NSNMatch
	// ExactMatch means calling code, national number and extension all agree.
	ExactMatch
)

func (m MatchType) String() string {
	switch m {
	case ShortNSNMatch:
		return "SHORT_NSN_MATCH"
	case NSNMatch:
		return "NSN_MATCH"
	case ExactMatch:
		return "EXACT_MATCH"
	default:
		return "NO_MATCH"
	}
}

const (
	// minNSNLen and maxNSNLen bound the national significant number for any
	// region, as E.164 does. Parse enforces only these; whether a length is
	// right for a particular country is what Possibility and IsValid answer.
	minNSNLen = 2
	// maxNSNLen is the longest national significant number E.164 allows,
	// with headroom for the leading zeros some regions keep significant.
	maxNSNLen = 17
	// maxExtLen is the longest extension retained while parsing.
	maxExtLen = 12
	// maxRawLen bounds the input accepted, to keep a malformed or hostile
	// input from driving the scanner.
	maxRawLen = 250
	// maxDigitsLen is the most significant characters any number can have:
	// a leading '+', the calling code and the national significant number.
	maxDigitsLen = 1 + countries.MaxCallingCodeLen + maxNSNLen
)
