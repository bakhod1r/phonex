package phonex

import "sort"

// Result pairs a parsed number with the error that parsing it produced.
// Exactly one of the two fields is set.
type Result struct {
	Phone *Phone
	Error error
}

// ParseMany parses each input, returning one Result per input in the same
// order. Inputs that fail to parse do not stop the others.
func ParseMany(numbers []string, opts ...ParseOption) []Result {
	options := DefaultParseOptions()
	for _, opt := range opts {
		options = opt(options)
	}

	results := make([]Result, len(numbers))
	for i, number := range numbers {
		p := new(Phone)
		if err := p.ParseWith(number, options); err != nil {
			results[i] = Result{Error: err}
			continue
		}
		results[i] = Result{Phone: p}
	}
	return results
}

// Unique returns the E.164 form of each distinct number, in the order it was
// first seen. Inputs that fail to parse are skipped.
func Unique(numbers []string, opts ...ParseOption) []string {
	options := DefaultParseOptions()
	for _, opt := range opts {
		options = opt(options)
	}
	options.KeepRawInput = false

	seen := make(map[string]struct{}, len(numbers))
	out := make([]string, 0, len(numbers))

	var p Phone
	for _, number := range numbers {
		if p.ParseWith(number, options) != nil {
			continue
		}
		e164 := p.E164()
		if _, dup := seen[e164]; dup {
			continue
		}
		seen[e164] = struct{}{}
		out = append(out, e164)
	}
	return out
}

// SortNumbers returns the E.164 form of each input in ascending order.
// Sorting the canonical form groups numbers by calling code, which is what
// makes the result useful for display. Inputs that fail to parse are skipped.
func SortNumbers(numbers []string, opts ...ParseOption) []string {
	options := DefaultParseOptions()
	for _, opt := range opts {
		options = opt(options)
	}
	options.KeepRawInput = false

	out := make([]string, 0, len(numbers))
	var p Phone
	for _, number := range numbers {
		if p.ParseWith(number, options) != nil {
			continue
		}
		out = append(out, p.E164())
	}
	sort.Strings(out)
	return out
}
