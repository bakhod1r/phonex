package phonex

import (
	"strings"
	"unsafe"

	"github.com/bakhod1r/phonex/countries"
)

// ParseOptions configures parsing. Build one with the With* options.
type ParseOptions struct {
	// DefaultCountry is the ISO-3166 alpha-2 region assumed when the input
	// carries no calling code.
	DefaultCountry string
	// AllowAlpha maps vanity letters onto their keypad digits, so that
	// "1-800-FLOWERS" parses as "+18003569377".
	AllowAlpha bool
	// KeepRawInput retains the original string on the parsed number. It is
	// on by default; turning it off lets a Phone outlive a large input
	// buffer without pinning it.
	KeepRawInput bool
}

// ParseOption customises parsing. Options take and return the option struct
// by value so that applying them never forces it onto the heap, which is what
// keeps Parse allocation-free.
type ParseOption func(ParseOptions) ParseOptions

// WithDefaultCountry sets the region assumed for numbers written without a
// calling code, e.g. "90 123 45 67" with "UZ".
func WithDefaultCountry(region string) ParseOption {
	region = normalizeRegion(region)
	return func(o ParseOptions) ParseOptions { o.DefaultCountry = region; return o }
}

// WithAlphaCharacters enables vanity-number parsing, mapping letters to the
// digits they share a telephone key with.
func WithAlphaCharacters() ParseOption {
	return func(o ParseOptions) ParseOptions { o.AllowAlpha = true; return o }
}

// WithoutRawInput drops the original input from the parsed number.
func WithoutRawInput() ParseOption {
	return func(o ParseOptions) ParseOptions { o.KeepRawInput = false; return o }
}

// DefaultParseOptions is the configuration Parse uses when given no options.
func DefaultParseOptions() ParseOptions {
	return ParseOptions{KeepRawInput: true}
}

// Parse parses a phone number.
//
// The input may be in international form ("+998901234567", "00998901234567"),
// in national form when a default country is given, or in RFC 3966 form
// ("tel:+1-202-555-0123;ext=42"). Punctuation and spacing are ignored.
//
// Parse fails when no region can be determined, when the input holds
// characters a number cannot, or when the digit count is outside the bounds
// E.164 sets for any number. It deliberately does not judge the number
// against its own country's rules: a wrong-length or unassigned number still
// parses, and Possibility and IsValid are what report on it.
//
// Callers that only want numbers they can dial should check IsValid.
func Parse(input string, opts ...ParseOption) (*Phone, error) {
	p := new(Phone)
	if err := p.Parse(input, opts...); err != nil {
		return nil, err
	}
	return p, nil
}

// ParseWith parses a phone number using an explicit option struct.
func ParseWith(input string, options ParseOptions) (*Phone, error) {
	p := new(Phone)
	if err := p.ParseWith(input, options); err != nil {
		return nil, err
	}
	return p, nil
}

// ParseBytes parses a phone number held in a byte slice. The slice is not
// retained unless raw input is kept.
func ParseBytes(input []byte, opts ...ParseOption) (*Phone, error) {
	p := new(Phone)
	if err := p.ParseBytes(input, opts...); err != nil {
		return nil, err
	}
	return p, nil
}

// ParseBytes parses into the receiver, reusing its storage.
func (p *Phone) ParseBytes(input []byte, opts ...ParseOption) error {
	// The conversion is what makes the raw input safe to retain; the input
	// slice itself is never referenced afterwards.
	return p.Parse(string(input), opts...)
}

// Parse parses into the receiver, reusing its storage. Parsing into an
// existing Phone performs no allocation for numbers in international or
// plain national form.
func (p *Phone) Parse(input string, opts ...ParseOption) error {
	options := DefaultParseOptions()
	for _, opt := range opts {
		options = opt(options)
	}
	return p.ParseWith(input, options)
}

// ParseWith parses into the receiver using an explicit option struct. It is
// the form to reach for in hot loops, where building the variadic option
// slice would itself cost an allocation.
func (p *Phone) ParseWith(input string, options ParseOptions) error {
	var defaultMeta *countries.Metadata
	if options.DefaultCountry != "" {
		m, ok := countries.Data[options.DefaultCountry]
		if !ok {
			return ErrInvalidCountry
		}
		defaultMeta = m
	}

	raw := input
	p.reset()
	if options.KeepRawInput {
		p.raw = raw
	}

	if len(input) > maxRawLen {
		return ErrTooLong
	}

	number, ext, err := splitExtension(input)
	if err != nil {
		return err
	}
	if len(ext) > maxExtLen {
		return ErrInvalidExtension
	}
	p.extLen = uint8(copy(p.ext[:], ext))

	// digits holds the significant characters of the number: an optional
	// leading '+' followed by digits. It aliases p.scratch, so every value
	// derived from it is copied before Parse returns.
	buf, err := normalizeInto(p.scratch[:0], number, options.AllowAlpha)
	if err != nil {
		return err
	}
	if len(buf) == 0 {
		return ErrTooShort
	}
	digits := unsafe.String(&buf[0], len(buf))

	hasPlus := digits[0] == '+'
	if hasPlus {
		digits = digits[1:]
		p.source = FromNumberWithPlusSign
	}
	if len(digits) == 0 {
		return ErrTooShort
	}

	if !hasPlus && defaultMeta != nil {
		// "00998..." or "011202..." — an international call placed from the
		// default region.
		if rest, stripped := stripIDD(digits, defaultMeta); stripped {
			if len(rest) == 0 {
				return ErrTooShort
			}
			digits = rest
			hasPlus = true
			p.source = FromNumberWithIDD
		}
	}

	var meta *countries.Metadata
	var nsn string

	switch {
	case hasPlus:
		meta, nsn, err = regionForInternational(digits)
		if err != nil {
			return err
		}

	case defaultMeta != nil:
		// A number written nationally may still repeat the calling code,
		// e.g. "998901234567" typed in Uzbekistan. Accept that only when
		// dropping the code leaves something the region recognises.
		if rest, okCC := stripOwnCallingCode(digits, defaultMeta); okCC {
			meta, nsn = defaultMeta, rest
			p.source = FromNumberWithoutPlusSign
		} else {
			meta, nsn = defaultMeta, digits
			p.source = FromDefaultCountry
		}

	default:
		// No '+' and no default region: the only remaining reading is that
		// the number starts with its calling code. Guessing is only safe
		// when the result is a number the region actually recognises, so a
		// bare "123" is rejected rather than read as a stub US number.
		meta, nsn, err = regionForInternational(digits)
		if err != nil || !meta.General.Match(nsn) {
			return ErrMissingCountry
		}
		p.source = FromNumberWithoutPlusSign
	}

	// The trunk prefix and any carrier selection code are not part of the
	// national significant number. People write them even in international
	// form — "+44 (0)20 7031 3000" is on plenty of business cards — so this
	// runs whichever way the number was written.
	stripped, carrier := stripNationalPrefix(nsn, meta)
	if len(stripped) != len(nsn) {
		// Stripping must leave a number of a length the region allows.
		// Otherwise the leading digits were significant after all, as they
		// are for a short service number.
		switch testPossibility(meta, stripped, Unknown) {
		case TooShort, IsPossibleLocalOnly, InvalidLength:
		default:
			nsn = stripped
			// carrier aliases the scratch buffer and must be copied out.
			if carrier != "" {
				p.carrierCode = string(append([]byte(nil), carrier...))
			}
		}
	}

	// Regions sharing a calling code are only distinguishable once the
	// national significant number is known: "242 302 1234" is Bahamian
	// however it was typed, and a +1 toll-free number belongs to the main
	// region rather than to whoever dialled it.
	if regions := countries.RegionsForCode(meta.DialCode); len(regions) > 1 {
		meta = pickRegion(regions, nsn)
	}

	// Only the bounds E.164 sets are enforced here. A number whose length is
	// wrong for its own country still parses, so that the caller can ask
	// Possibility why, and IsValid whether it is a real number at all.
	if len(nsn) < minNSNLen {
		return ErrTooShort
	}
	if len(nsn) > maxNSNLen {
		return ErrTooLong
	}

	p.nsnLen = uint8(copy(p.nsn[:], nsn))
	p.meta = meta
	return nil
}

// reset clears everything except the inline buffers, whose contents are
// bounded by the length fields.
func (p *Phone) reset() {
	p.nsnLen = 0
	p.extLen = 0
	p.meta = nil
	p.typ = 0
	p.source = FromDefaultCountry
	p.carrierCode = ""
	p.raw = ""
}

// normalizeInto appends the significant characters of s to dst: a leading '+'
// if present, then digits. Letters are mapped to keypad digits when alpha is
// set and rejected otherwise, so that "+1 800 CALL" is never silently read as
// "+1800". Appending stops at dst's capacity, which bounds the work a hostile
// input can cause.
func normalizeInto(dst []byte, s string, alpha bool) ([]byte, error) {
	limit := cap(dst)
	seenPlus := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c == '+' && len(dst) == 0 && !seenPlus:
			seenPlus = true
		case isASCIILetter(c):
			if !alpha {
				return nil, ErrInvalidCharacters
			}
			c = alphaDigit(c)
			if c == 0 {
				continue
			}
		default:
			// Punctuation, spacing and grouping characters carry no
			// information and are dropped.
			continue
		}
		if len(dst) == limit {
			return nil, ErrTooLong
		}
		dst = append(dst, c)
	}
	return dst, nil
}

func isASCIILetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// alphaDigit maps a letter to the digit it shares a telephone key with,
// returning 0 for letters with no digit (there are none on the ITU keypad,
// but the table stays total for safety).
func alphaDigit(c byte) byte {
	if c >= 'a' && c <= 'z' {
		c -= 'a' - 'A'
	}
	switch c {
	case 'A', 'B', 'C':
		return '2'
	case 'D', 'E', 'F':
		return '3'
	case 'G', 'H', 'I':
		return '4'
	case 'J', 'K', 'L':
		return '5'
	case 'M', 'N', 'O':
		return '6'
	case 'P', 'Q', 'R', 'S':
		return '7'
	case 'T', 'U', 'V':
		return '8'
	case 'W', 'X', 'Y', 'Z':
		return '9'
	}
	return 0
}

// stripIDD removes the default region's international dialling prefix, so
// that a number dialled as "00 998 ..." is understood the same as "+998 ...".
func stripIDD(digits string, meta *countries.Metadata) (string, bool) {
	if meta.InternationalPrefix.Empty() {
		return digits, false
	}
	re := meta.InternationalPrefix.Regexp()
	// The stored pattern is fully anchored; match progressively longer
	// prefixes rather than re-compiling an unanchored variant.
	for n := 1; n <= len(digits) && n <= 5; n++ {
		if !re.MatchString(digits[:n]) {
			continue
		}
		rest := digits[n:]
		// A leading zero after the IDD is never part of a calling code, so
		// this was not really an international prefix.
		if len(rest) == 0 || rest[0] == '0' {
			return digits, false
		}
		return rest, true
	}
	return digits, false
}

// stripOwnCallingCode removes the region's own calling code from a number
// written without '+', but only when what remains is recognisable. Without
// that check "441234567" in GB (calling code 44) would lose its first digits.
func stripOwnCallingCode(digits string, meta *countries.Metadata) (string, bool) {
	cc := meta.DialCode
	if len(digits) <= len(cc) || !strings.HasPrefix(digits, cc) {
		return digits, false
	}
	rest := digits[len(cc):]
	if !meta.General.HasLength(len(rest)) {
		return digits, false
	}
	// Prefer the reading that the whole string is a national number when
	// that is also plausible and actually matches the region's ranges.
	stripped, _ := stripNationalPrefix(digits, meta)
	if meta.General.Match(stripped) && !meta.General.Match(rest) {
		return digits, false
	}
	return rest, true
}

// regionForInternational splits a number written in international form into
// its calling code and national significant number.
func regionForInternational(digits string) (*countries.Metadata, string, error) {
	if digits[0] == '0' {
		// Calling codes never start with zero.
		return nil, "", ErrInvalidCountryCode
	}
	for n := 1; n <= countries.MaxCallingCodeLen && n <= len(digits); n++ {
		regions := countries.RegionsForCode(digits[:n])
		if len(regions) == 0 {
			continue
		}
		// The main region owns the calling code, and its rules govern the
		// trunk prefix. Which region the number actually belongs to is
		// settled once that prefix is gone.
		return regions[0], digits[n:], nil
	}
	return nil, "", ErrInvalidCountryCode
}

// pickRegion chooses between regions sharing a calling code, such as the
// twenty-odd regions behind +1. regions[0] is the main region and is the
// fallback when nothing more specific matches.
func pickRegion(regions []*countries.Metadata, nsn string) *countries.Metadata {
	if len(regions) == 1 {
		return regions[0]
	}
	for _, m := range regions {
		if !m.LeadingDigits.Empty() {
			if m.LeadingDigits.Match(nsn) {
				return m
			}
			continue
		}
		if typeFor(m, nsn) != Unknown {
			return m
		}
	}
	return regions[0]
}

// stripNationalPrefix removes the trunk prefix and any carrier selection code
// from a national number, returning the national significant number and the
// carrier code that was removed.
func stripNationalPrefix(nsn string, meta *countries.Metadata) (string, string) {
	if len(nsn) == 0 {
		return nsn, ""
	}

	// The common case is a literal prefix such as "0" or "8"; handle it
	// without touching the regexp engine.
	if meta.SimpleNationalPrefix != "" {
		np := meta.SimpleNationalPrefix
		if !strings.HasPrefix(nsn, np) {
			return nsn, ""
		}
		candidate := nsn[len(np):]
		if candidate == "" || (meta.General.Match(nsn) && !meta.General.Match(candidate)) {
			return nsn, ""
		}
		return candidate, ""
	}

	if meta.NationalPrefixForParsing.Empty() {
		return nsn, ""
	}
	re := meta.NationalPrefixForParsing.Regexp()
	m := re.FindStringSubmatchIndex(nsn)
	if m == nil || m[1] == 0 {
		return nsn, ""
	}

	// The first capturing group, where there is one, is the carrier selection
	// code the caller dialled. Whether the groups participated at all is told
	// by the last one: Brazil's rule makes the whole carrier-plus-number tail
	// optional, so "011..." and "03111..." both match, and only the second
	// carries a carrier code.
	groups := len(m)/2 - 1
	lastGroupSet := groups > 0 && m[2*groups] >= 0
	firstGroup := func() string {
		if m[2] < 0 {
			return ""
		}
		return nsn[m[2]:m[3]]
	}

	var candidate, carrier string
	if meta.NationalPrefixTransformRule == "" || !lastGroupSet {
		candidate = nsn[m[1]:]
		if lastGroupSet {
			carrier = firstGroup()
		}
	} else {
		candidate = expandGroups(meta.NationalPrefixTransformRule, nsn, m) + nsn[m[1]:]
		// With a transform rule the last group is the number itself, so a
		// carrier code is only present when there is a group before it.
		if groups > 1 {
			carrier = firstGroup()
		}
	}

	if candidate == "" {
		return nsn, ""
	}
	// Stripping must not turn a number the region recognises into one it
	// does not.
	if meta.General.Match(nsn) && !meta.General.Match(candidate) {
		return nsn, ""
	}
	return candidate, carrier
}

// expandGroups substitutes $1..$9 in rule with the corresponding submatches
// of src described by the index pairs in m.
func expandGroups(rule, src string, m []int) string {
	var b strings.Builder
	b.Grow(len(rule) + 8)
	for i := 0; i < len(rule); i++ {
		if rule[i] != '$' || i+1 >= len(rule) || rule[i+1] < '1' || rule[i+1] > '9' {
			b.WriteByte(rule[i])
			continue
		}
		g := int(rule[i+1] - '0')
		i++
		if 2*g+1 < len(m) && m[2*g] >= 0 {
			b.WriteString(src[m[2*g]:m[2*g+1]])
		}
	}
	return b.String()
}
