package phonex

import (
	"regexp"
	"strings"
)

// extMarker matches the trailing "ext. 1234" style suffixes people write in
// free text. The marker is required to stand alone as a word so that a vanity
// number such as "1-800-PAINT-411" is not mistaken for an extension.
var extMarker = regexp.MustCompile(
	`(?i)[ \t,]*(?:\b(?:extension|extensión|extensao|extensão|anexo|interno|ramal|e?xtn?|int)\b|\bx|#|~)[ \t:.,\-]*([0-9]{1,12})[ \t#]*$`)

// splitExtension separates a number from its extension.
//
// It understands the RFC 3966 ";ext=" parameter as well as the "ext"/"x"/"#"
// suffixes used in free text, and applies an RFC 3966 "phone-context" when the
// number itself carries no calling code.
func splitExtension(s string) (number, ext string, err error) {
	s = strings.TrimSpace(s)
	if len(s) >= 4 && strings.EqualFold(s[:4], "tel:") {
		s = s[4:]
	}

	if i := strings.IndexByte(s, ';'); i >= 0 {
		params := s[i:]
		number = s[:i]
		ext = rfcParam(params, "ext=")
		if ext != "" && !allDigits(ext) {
			return "", "", ErrInvalidExtension
		}
		if ctx := rfcParam(params, "phone-context="); strings.HasPrefix(ctx, "+") {
			if !strings.HasPrefix(strings.TrimSpace(number), "+") {
				number = ctx + number
			}
		}
		return number, ext, nil
	}

	// The regexp only earns its cost when a marker could be present.
	if !mayHaveExtension(s) {
		return s, "", nil
	}
	if m := extMarker.FindStringSubmatchIndex(s); m != nil {
		return s[:m[0]], s[m[2]:m[3]], nil
	}
	return s, "", nil
}

// rfcParam returns the value of a ";name=value" parameter.
func rfcParam(params, name string) string {
	for _, part := range strings.Split(params, ";") {
		if len(part) > len(name) && strings.EqualFold(part[:len(name)], name) {
			return part[len(name):]
		}
	}
	return ""
}

func mayHaveExtension(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; isASCIILetter(c) || c == '#' || c == '~' {
			return true
		}
	}
	return false
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
