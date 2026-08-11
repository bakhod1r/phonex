package phonex

import (
	"regexp"
	"strings"
	"sync"

	"github.com/bakhod1r/phonex/countries"
)

// digitPlaceholder marks a position in a formatting template that a typed
// digit will fill.
const digitPlaceholder = ' '

// minLeadingDigits is the point from which a format's leading digits become
// discriminating. Below it every format of the region is still a candidate.
const minLeadingDigits = 3

// Formatter formats a number while it is being typed, one digit at a time.
//
// It is the counterpart of libphonenumber's AsYouTypeFormatter and is meant
// to drive an input field:
//
//	f := phonex.NewFormatter("US")
//	for _, r := range "2025550123" {
//	    out = f.InputDigit(r)
//	}
//	// out == "(202) 555-0123"
//
// A Formatter is not safe for concurrent use.
type Formatter struct {
	region string
	meta   *countries.Metadata

	// digits holds the digits typed so far, without the leading '+'.
	digits []byte
	plus   bool
	// out caches the rendering of the current digits.
	out string
}

// NewFormatter returns a Formatter for numbers typed in region. An unknown
// region still yields a usable Formatter: it formats numbers typed in
// international form and echoes anything else.
func NewFormatter(region string) *Formatter {
	f := &Formatter{region: normalizeRegion(region)}
	f.meta = countries.Data[f.region]
	f.digits = make([]byte, 0, maxNSNLen+countries.MaxCallingCodeLen)
	return f
}

// Clear resets the Formatter so it can format another number.
func (f *Formatter) Clear() {
	f.digits = f.digits[:0]
	f.plus = false
	f.out = ""
}

// InputDigit appends one typed character and returns the number formatted so
// far. Characters other than digits and a leading '+' are ignored.
func (f *Formatter) InputDigit(r rune) string {
	switch {
	case r >= '0' && r <= '9':
		if len(f.digits) < cap(f.digits) {
			f.digits = append(f.digits, byte(r))
		}
	case r == '+' && len(f.digits) == 0 && !f.plus:
		f.plus = true
	default:
		return f.out
	}
	f.out = f.render()
	return f.out
}

// Input appends every character of s and returns the result.
func (f *Formatter) Input(s string) string {
	for _, r := range s {
		f.InputDigit(r)
	}
	return f.out
}

// RemoveLastDigit removes the most recently typed character and returns the
// number formatted so far.
func (f *Formatter) RemoveLastDigit() string {
	switch {
	case len(f.digits) > 0:
		f.digits = f.digits[:len(f.digits)-1]
	case f.plus:
		f.plus = false
	}
	f.out = f.render()
	return f.out
}

// String returns the current formatted number.
func (f *Formatter) String() string { return f.out }

// Digits returns the digits typed so far, without punctuation.
func (f *Formatter) Digits() string { return string(f.digits) }

func (f *Formatter) render() string {
	typed := string(f.digits)
	if f.plus {
		return f.renderInternational(typed)
	}
	if f.meta == nil {
		return typed
	}
	return f.renderNational(f.meta, typed)
}

// renderInternational formats a number typed with a leading '+', where the
// calling code has to be recognised before the rest can be grouped.
func (f *Formatter) renderInternational(typed string) string {
	if typed == "" {
		return "+"
	}
	for n := 1; n <= countries.MaxCallingCodeLen && n <= len(typed); n++ {
		regions := countries.RegionsForCode(typed[:n])
		if len(regions) == 0 {
			continue
		}
		nsn := typed[n:]
		if nsn == "" {
			return "+" + typed
		}
		meta := mainRegionFor(regions[0])
		if body, ok := formatPartial(meta, nsn, "", true); ok {
			return "+" + typed[:n] + " " + body
		}
		return "+" + typed[:n] + " " + nsn
	}
	return "+" + typed
}

// renderNational formats a number typed without a calling code, keeping any
// trunk prefix the user typed.
func (f *Formatter) renderNational(meta *countries.Metadata, typed string) string {
	fm := mainRegionFor(meta)
	np := meta.NationalPrefix
	if np != "" && strings.HasPrefix(typed, np) {
		rest := typed[len(np):]
		if rest == "" {
			return typed
		}
		if body, ok := formatPartial(fm, rest, np, false); ok {
			return body
		}
		return typed
	}
	if body, ok := formatPartial(fm, typed, "", false); ok {
		return body
	}
	return typed
}

// formatPartial groups the digits typed so far using the first of the
// region's formats that can accommodate them. nationalPrefix, when non-empty,
// is the trunk prefix the user typed and is reinserted by the format's own
// national prefix rule. intl selects the international grouping, which some
// regions write differently from the national one.
func formatPartial(m *countries.Metadata, digits, nationalPrefix string, intl bool) (string, bool) {
	if digits == "" {
		return "", false
	}
	for i := range m.Formats {
		rule := &m.Formats[i]
		if len(digits) >= minLeadingDigits && !rule.LeadingDigits.Empty() && !rule.LeadingDigits.Match(digits) {
			continue
		}
		tpl := formatTemplate(rule, nationalPrefix, intl)
		if tpl == "" || countPlaceholders(tpl) < len(digits) {
			continue
		}
		return fillTemplate(tpl, digits), true
	}
	return "", false
}

// countPlaceholders counts the digit slots in a template.
func countPlaceholders(tpl string) int {
	n := 0
	for _, r := range tpl {
		if r == digitPlaceholder {
			n++
		}
	}
	return n
}

// fillTemplate substitutes the typed digits into a template and drops
// everything from the first unfilled slot onwards, so the caret always sits
// after the last digit the user typed.
func fillTemplate(tpl, digits string) string {
	var b strings.Builder
	b.Grow(len(tpl))
	i := 0
	for _, r := range tpl {
		if r != digitPlaceholder {
			b.WriteRune(r)
			continue
		}
		if i == len(digits) {
			break
		}
		b.WriteByte(digits[i])
		i++
	}
	return strings.TrimRight(b.String(), " -()")
}

// templateCache memoises the (metadata, format, prefix) to template mapping.
// Templates depend only on the metadata, which is immutable, so caching them
// keeps each keystroke free of regexp work after the first.
var templateCache sync.Map // templateKey -> string

type templateKey struct {
	pattern string
	format  string
	prefix  string
	intl    bool
}

// formatTemplate renders a format rule as a template of placeholders, e.g.
// "(\d{3})(\d{4})" with format "$1-$2" becomes "XXX-XXXX" (with placeholders
// in place of X). It returns "" when the rule cannot be turned into a fixed
// template.
func formatTemplate(rule *countries.NumberFormat, nationalPrefix string, intl bool) string {
	key := templateKey{pattern: rule.Pattern.Source(), format: rule.Format, prefix: nationalPrefix, intl: intl}
	if v, ok := templateCache.Load(key); ok {
		return v.(string)
	}
	tpl := buildTemplate(rule, nationalPrefix, intl)
	templateCache.Store(key, tpl)
	return tpl
}

func buildTemplate(rule *countries.NumberFormat, nationalPrefix string, intl bool) string {
	pattern := unanchor(rule.Pattern.Source())
	if pattern == "" {
		return ""
	}
	// Relaxing the pattern lets it match a run of 9s of any admissible
	// length, which is what reveals the grouping.
	relaxed, err := regexp.Compile("^(?:" + relaxPattern(pattern) + ")")
	if err != nil {
		return ""
	}

	const longestNumber = "999999999999999"
	match := relaxed.FindString(longestNumber)
	if match == "" {
		return ""
	}

	base := rule.Format
	if intl && rule.IntlFormat != "" {
		if rule.IntlFormat == "NA" {
			return ""
		}
		base = rule.IntlFormat
	}

	replacement := base
	if nationalPrefix != "" && rule.NationalPrefixFormattingRule != "" {
		r := strings.ReplaceAll(rule.NationalPrefixFormattingRule, "$NP", nationalPrefix)
		r = strings.ReplaceAll(r, "$FG", "$1")
		replacement = applyPrefixRule(base, r)
	} else if nationalPrefix != "" {
		replacement = nationalPrefix + base
	}

	grouped := relaxed.ReplaceAllString(match, goTemplate(replacement))
	if grouped == "" {
		return ""
	}
	return strings.Map(func(r rune) rune {
		if r == '9' {
			return digitPlaceholder
		}
		return r
	}, grouped)
}

// unanchor strips the "^(?:" and ")$" that generated patterns carry.
func unanchor(src string) string {
	s := strings.TrimPrefix(src, "^(?:")
	if len(s) == len(src) {
		return src
	}
	return strings.TrimSuffix(s, ")$")
}

// relaxPattern turns a number pattern into one that matches any digits:
// character classes and literal digits both become "\d", while the counts
// inside "{...}" quantifiers are left alone. Without this, a pattern such as
// "(9\d{2})(\d{4})" would only match numbers that really start with 9.
func relaxPattern(p string) string {
	var b strings.Builder
	b.Grow(len(p) + 8)
	inBraces := false
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch {
		case c == '\\' && i+1 < len(p):
			// Escapes such as "\d" pass through untouched.
			b.WriteByte(c)
			i++
			b.WriteByte(p[i])
		case c == '{':
			inBraces = true
			b.WriteByte(c)
		case c == '}':
			inBraces = false
			b.WriteByte(c)
		case c == '[':
			// A character class stands for exactly one digit.
			j := strings.IndexByte(p[i:], ']')
			if j < 0 {
				return p
			}
			i += j
			b.WriteString(`\d`)
		case c >= '0' && c <= '9' && !inBraces:
			b.WriteString(`\d`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
