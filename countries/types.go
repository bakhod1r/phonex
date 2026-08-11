// Package countries holds the phone-number metadata generated from Google's
// libphonenumber PhoneNumberMetadata.xml.
//
// Metadata values are always used through a *Metadata pointer: they embed
// sync.Once guards for lazily compiled regular expressions and must not be
// copied.
package countries

import "github.com/bakhod1r/phonex/internal/lazyre"

// lazyRe is a regular expression compiled on first use. The alias keeps the
// generated table readable while the machinery lives in one place.
type lazyRe = lazyre.Re

// PhoneType identifies a number range within a country's metadata.
type PhoneType uint8

const (
	FixedLine PhoneType = iota
	Mobile
	TollFree
	PremiumRate
	SharedCost
	VoIP
	PersonalNumber
	Pager
	UAN
	Voicemail

	// NumDescs is the number of per-type descriptors stored in Metadata.Descs.
	NumDescs

	// FixedLineOrMobile is reported when a number matches both ranges.
	FixedLineOrMobile
	// Unknown is reported when no range matches.
	Unknown
)

func (t PhoneType) String() string {
	switch t {
	case FixedLine:
		return "FIXED_LINE"
	case Mobile:
		return "MOBILE"
	case TollFree:
		return "TOLL_FREE"
	case PremiumRate:
		return "PREMIUM_RATE"
	case SharedCost:
		return "SHARED_COST"
	case VoIP:
		return "VOIP"
	case PersonalNumber:
		return "PERSONAL_NUMBER"
	case Pager:
		return "PAGER"
	case UAN:
		return "UAN"
	case Voicemail:
		return "VOICEMAIL"
	case FixedLineOrMobile:
		return "FIXED_LINE_OR_MOBILE"
	default:
		return "UNKNOWN"
	}
}

// Desc describes a single number range (fixed line, mobile, toll free, ...).
type Desc struct {
	Pattern lazyRe
	// Lengths holds the sorted possible national number lengths.
	Lengths []int32
	// LocalOnly holds lengths that are only valid when dialled locally.
	LocalOnly []int32
	Example   string
}

// Exists reports whether the country defines this range at all.
func (d *Desc) Exists() bool { return !d.Pattern.Empty() || len(d.Lengths) > 0 }

// Match reports whether the national significant number matches this range.
func (d *Desc) Match(nsn string) bool { return d.Pattern.Match(nsn) }

// HasLength reports whether n is a possible national number length.
func (d *Desc) HasLength(n int) bool {
	for _, l := range d.Lengths {
		if int(l) == n {
			return true
		}
		if int(l) > n {
			break
		}
	}
	return false
}

// HasLocalOnlyLength reports whether n is valid only for local dialling.
func (d *Desc) HasLocalOnlyLength(n int) bool {
	for _, l := range d.LocalOnly {
		if int(l) == n {
			return true
		}
		if int(l) > n {
			break
		}
	}
	return false
}

// NumberFormat is one national formatting rule.
type NumberFormat struct {
	// Pattern is anchored and captures the groups referenced by Format.
	Pattern lazyRe
	// Format is the replacement template using $1..$9.
	Format string
	// LeadingDigits, when set, must match the start of the national number
	// for this rule to apply.
	LeadingDigits lazyRe
	// NationalPrefixFormattingRule wraps the formatted groups, e.g. "$NP$FG".
	NationalPrefixFormattingRule string
	// NationalPrefixOptional reports whether the national prefix may be
	// omitted when formatting nationally.
	NationalPrefixOptional bool
	// CarrierCodeFormattingRule wraps the groups when a carrier code is used.
	CarrierCodeFormattingRule string
	// IntlFormat overrides Format for international formatting. The value
	// "NA" means the number must not be formatted internationally.
	IntlFormat string
}

// Metadata holds everything known about one calling region.
type Metadata struct {
	ISO2 string
	ISO3 string
	Name string

	// DialCode is the country calling code without a leading '+'.
	DialCode string
	// IsMainCountry reports whether this region owns DialCode. Several
	// regions may share one code (e.g. +1); exactly one is the main region.
	IsMainCountry bool
	// LeadingDigits disambiguates regions sharing a DialCode.
	LeadingDigits lazyRe

	NationalPrefix string
	// SimpleNationalPrefix is set when the trunk prefix is a plain literal
	// with no carrier-code group and no transform rule, which lets the
	// parser strip it without the regexp engine.
	SimpleNationalPrefix string
	// NationalPrefixForParsing matches the trunk prefix (and any carrier
	// selection code) to strip before validation.
	NationalPrefixForParsing lazyRe
	// NationalPrefixTransformRule rewrites the number after stripping.
	NationalPrefixTransformRule string

	InternationalPrefix          lazyRe
	PreferredInternationalPrefix string
	PreferredExtnPrefix          string
	MobileNumberPortable         bool

	// General matches any valid number in the region.
	General Desc
	// Descs holds the per-type ranges, indexed by PhoneType.
	Descs [NumDescs]Desc
	// NoIntlDialling matches numbers that cannot be dialled from abroad.
	NoIntlDialling Desc

	Formats []NumberFormat

	// MinLength and MaxLength are the shortest and longest possible national
	// number lengths, derived from General.Lengths.
	MinLength int
	MaxLength int

	// Timezones holds the region's IANA time zones. It comes from
	// internal/metadata/metadata.json, not from libphonenumber, and is
	// empty in the bundled data.
	Timezones []string
}

// Desc returns the descriptor for t, or nil if t has no descriptor.
func (m *Metadata) Desc(t PhoneType) *Desc {
	if t >= NumDescs {
		return nil
	}
	return &m.Descs[t]
}
