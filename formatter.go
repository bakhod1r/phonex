package phonex

import (
	"strings"

	"github.com/bakhod1r/phonex/countries"
)

// FormatType selects an output format.
type FormatType int

const (
	// FormatE164 is the canonical machine format, e.g. "+998901234567".
	FormatE164 FormatType = iota
	// FormatInternational is the human-readable international format,
	// e.g. "+998 90 123 45 67".
	FormatInternational
	// FormatNational is the format used inside the country, including the
	// trunk prefix, e.g. "(202) 555-0123" or "090 123 45 67".
	FormatNational
	// FormatRFC3966 is the "tel:" URI form, e.g. "tel:+998-90-123-45-67".
	FormatRFC3966
)

// E164 returns the number in E.164 format. Extensions are not part of E.164
// and are omitted.
func (p *Phone) E164() string {
	if p == nil || p.meta == nil || p.nsnLen == 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(1 + len(p.meta.DialCode) + int(p.nsnLen))
	b.WriteByte('+')
	b.WriteString(p.meta.DialCode)
	b.Write(p.nsn[:p.nsnLen])
	return b.String()
}

// AppendE164 appends the E.164 form to dst without allocating.
func (p *Phone) AppendE164(dst []byte) []byte {
	if p == nil || p.meta == nil || p.nsnLen == 0 {
		return dst
	}
	dst = append(dst, '+')
	dst = append(dst, p.meta.DialCode...)
	return append(dst, p.nsn[:p.nsnLen]...)
}

// International returns the number grouped for international display,
// e.g. "+44 20 7031 3000".
func (p *Phone) International() string {
	if p == nil || p.meta == nil {
		return ""
	}
	body := formatNSN(p.meta, p.nsnRef(), FormatInternational, "")
	return "+" + p.meta.DialCode + " " + body
}

// National returns the number as it is written inside its own country,
// e.g. "020 7031 3000".
func (p *Phone) National() string {
	if p == nil || p.meta == nil {
		return ""
	}
	return formatNSN(p.meta, p.nsnRef(), FormatNational, "")
}

// NationalWithCarrier returns the national format with a carrier selection
// code applied, falling back to the plain national format in regions that do
// not use one.
func (p *Phone) NationalWithCarrier(carrierCode string) string {
	if p == nil || p.meta == nil {
		return ""
	}
	return formatNSN(p.meta, p.nsnRef(), FormatNational, carrierCode)
}

// RFC3966 returns the number as a "tel:" URI, including any extension.
func (p *Phone) RFC3966() string {
	if p == nil || p.meta == nil {
		return ""
	}
	body := formatNSN(p.meta, p.nsnRef(), FormatInternational, "")
	s := "tel:+" + p.meta.DialCode + "-" + dashSeparated(body)
	if p.extLen > 0 {
		s += ";ext=" + p.Extension()
	}
	return s
}

// dashSeparated rewrites a formatted national number so that every run of
// grouping punctuation becomes a single '-', which is the only separator
// RFC 3966 allows. Regions group with spaces, dots, brackets or slashes, so
// replacing spaces alone would leave "20.12.34" intact.
func dashSeparated(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	pendingDash := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			if pendingDash && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingDash = false
			b.WriteByte(c)
			continue
		}
		// Anything else is grouping punctuation, including the multi-byte
		// dashes and spaces some regions use; a run of it collapses to one
		// separator, and a leading or trailing run disappears.
		pendingDash = true
	}
	return b.String()
}

// OutOfCountry returns the number as it must be dialled from fromRegion,
// including that region's international dialling prefix. Dialling from the
// same country yields the national format.
func (p *Phone) OutOfCountry(fromRegion string) string {
	if p == nil || p.meta == nil {
		return ""
	}
	from, ok := countries.Data[normalizeRegion(fromRegion)]
	if !ok {
		return p.International()
	}
	if from.DialCode == p.meta.DialCode {
		// Within the same calling code the number is dialled nationally.
		// The exception is +1, where all regions dial each other with the
		// full national number and the trunk prefix "1".
		if p.meta.DialCode == "1" {
			return "1 " + formatNSN(p.meta, p.nsnRef(), FormatNational, "")
		}
		return p.National()
	}
	idd := from.PreferredInternationalPrefix
	if idd == "" {
		idd = literalPrefix(from.InternationalPrefix.Source())
	}
	body := formatNSN(p.meta, p.nsnRef(), FormatInternational, "")
	if idd == "" {
		return "+" + p.meta.DialCode + " " + body
	}
	return idd + " " + p.meta.DialCode + " " + body
}

// Format renders the number in the requested format.
func (p *Phone) Format(f FormatType) string {
	switch f {
	case FormatInternational:
		return p.International()
	case FormatNational:
		return p.National()
	case FormatRFC3966:
		return p.RFC3966()
	default:
		return p.E164()
	}
}

// formatNSN applies the region's formatting rules to a national significant
// number, returning the digits unchanged when no rule matches.
func formatNSN(m *countries.Metadata, nsn string, f FormatType, carrierCode string) string {
	// Formatting rules belong to the calling code, not to the individual
	// region: every region behind +1 is written the way the main region is.
	m = mainRegionFor(m)
	rule := chooseFormat(m, nsn)
	if rule == nil {
		return nsn
	}

	template := rule.Format
	if f == FormatInternational {
		switch rule.IntlFormat {
		case "":
			// Fall through: the national grouping is reused abroad.
		case "NA":
			return nsn
		default:
			template = rule.IntlFormat
		}
	}

	// The national prefix and carrier code rules decorate the first group
	// only: "($NP$FG)" turns "$1 $2 $3" into "(0$1) $2 $3", not into
	// "(0$1 $2 $3)".
	replacement := template
	if f == FormatNational {
		var rewrite string
		switch {
		case carrierCode != "" && rule.CarrierCodeFormattingRule != "":
			rewrite = strings.ReplaceAll(rule.CarrierCodeFormattingRule, "$CC", carrierCode)
		case rule.NationalPrefixFormattingRule != "":
			rewrite = rule.NationalPrefixFormattingRule
		}
		if rewrite != "" {
			rewrite = strings.ReplaceAll(rewrite, "$NP", m.NationalPrefix)
			rewrite = strings.ReplaceAll(rewrite, "$FG", "$1")
			replacement = applyPrefixRule(template, rewrite)
		}
	}

	re := rule.Pattern.Regexp()
	if re == nil {
		return nsn
	}
	out := re.ReplaceAllString(nsn, goTemplate(replacement))
	if out == "" {
		return nsn
	}
	return out
}

// mainRegionFor returns the region that owns a calling code. Several things
// belong to the code rather than to the individual region behind it: the
// formatting rules, and the set of lengths a number may possibly have.
func mainRegionFor(m *countries.Metadata) *countries.Metadata {
	if m.IsMainCountry {
		return m
	}
	if regions := countries.RegionsForCode(m.DialCode); len(regions) > 0 {
		return regions[0]
	}
	return m
}

// applyPrefixRule splices a national prefix or carrier code rule into a format
// template, replacing the template's first group reference.
//
// The reference is not always "$1": Argentina formats mobile numbers as
// "$2 15-$3-$4", dropping the leading 9, so the prefix rule has to attach to
// "$2". Matching the first "$<digit>" rather than "$1" is what libphonenumber
// does, and it is the difference between "09 15-2345-6789" and a number with
// no trunk prefix at all.
func applyPrefixRule(format, rule string) string {
	i := firstGroupRef(format)
	if i < 0 {
		return format
	}
	return format[:i] + rule + format[i+2:]
}

// firstGroupRef returns the index of the first "$<digit>" in s, or -1.
func firstGroupRef(s string) int {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '$' && s[i+1] >= '1' && s[i+1] <= '9' {
			return i
		}
	}
	return -1
}

// chooseFormat returns the first rule whose leading digits and pattern both
// accept the number. Upstream orders the rules from most to least specific.
func chooseFormat(m *countries.Metadata, nsn string) *countries.NumberFormat {
	for i := range m.Formats {
		r := &m.Formats[i]
		if !r.LeadingDigits.Empty() && !r.LeadingDigits.Match(nsn) {
			continue
		}
		if r.Pattern.Match(nsn) {
			return r
		}
	}
	return nil
}

// goTemplate rewrites libphonenumber's "$1" group references into the
// "${1}" form Go's regexp expander requires, so that a group followed by a
// digit or letter is not read as a longer group name.
func goTemplate(s string) string {
	if !strings.ContainsRune(s, '$') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for i := 0; i < len(s); i++ {
		if s[i] == '$' && i+1 < len(s) && s[i+1] >= '1' && s[i+1] <= '9' {
			b.WriteString("${")
			b.WriteByte(s[i+1])
			b.WriteByte('}')
			i++
			continue
		}
		if s[i] == '$' {
			// A literal '$' must be escaped for the expander.
			b.WriteString("$$")
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// literalPrefix extracts a dialable prefix from an anchored pattern such as
// "^(?:00)$", returning "" when the pattern offers a choice.
func literalPrefix(src string) string {
	s := strings.TrimPrefix(src, "^(?:")
	s = strings.TrimSuffix(s, ")$")
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return ""
		}
	}
	return s
}
