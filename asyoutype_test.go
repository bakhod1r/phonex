package phonex

import "testing"

func TestFormatterFinalOutput(t *testing.T) {
	tests := []struct {
		region string
		typed  string
		want   string
	}{
		{"US", "2025550123", "(202) 555-0123"},
		{"US", "+12025550123", "+1 202-555-0123"},
		{"GB", "02070313000", "020 7031 3000"},
		{"UZ", "901234567", "90 123 45 67"},
		{"AU", "0412345678", "0412 345 678"},
		{"RU", "84951234567", "8 (495) 123-45-67"},
		{"DE", "030901820", "030 901820"},
		{"", "+998901234567", "+998 90 123 45 67"},
	}
	for _, tt := range tests {
		t.Run(tt.region+"/"+tt.typed, func(t *testing.T) {
			f := NewFormatter(tt.region)
			got := f.Input(tt.typed)
			if got != tt.want {
				t.Errorf("Input(%q) = %q, want %q", tt.typed, got, tt.want)
			}
			if f.String() != got {
				t.Errorf("String() = %q, want %q", f.String(), got)
			}
		})
	}
}

// TestFormatterKeepsEveryDigit is the invariant that matters most in an input
// field: whatever punctuation is added, no typed digit may be lost.
func TestFormatterKeepsEveryDigit(t *testing.T) {
	for _, region := range []string{"US", "GB", "DE", "UZ", "RU", "AU", "BR", "IN", "JP", "FR"} {
		p, ok := ExampleNumber(region)
		if !ok {
			continue
		}
		typed := p.NSN()
		f := NewFormatter(region)
		for i, r := range typed {
			out := f.InputDigit(r)
			if got := onlyDigits(out); got != typed[:i+1] {
				t.Fatalf("%s: after typing %q the output %q holds digits %q", region, typed[:i+1], out, got)
			}
		}
	}
}

func TestFormatterRemoveLastDigit(t *testing.T) {
	f := NewFormatter("US")
	f.Input("2025550123")
	if got, want := f.RemoveLastDigit(), "(202) 555-012"; got != want {
		t.Errorf("RemoveLastDigit() = %q, want %q", got, want)
	}
	if got, want := f.Digits(), "202555012"; got != want {
		t.Errorf("Digits() = %q, want %q", got, want)
	}
}

func TestFormatterClear(t *testing.T) {
	f := NewFormatter("US")
	f.Input("2025550123")
	f.Clear()
	if f.String() != "" || f.Digits() != "" {
		t.Errorf("Clear() left %q / %q", f.String(), f.Digits())
	}
	if got, want := f.Input("2025550123"), "(202) 555-0123"; got != want {
		t.Errorf("after Clear, Input() = %q, want %q", got, want)
	}
}

func TestFormatterIgnoresPunctuation(t *testing.T) {
	f := NewFormatter("US")
	if got, want := f.Input("(202) 555-0123"), "(202) 555-0123"; got != want {
		t.Errorf("Input() = %q, want %q", got, want)
	}
}

func TestFormatterUnknownRegionEchoesDigits(t *testing.T) {
	f := NewFormatter("ZZ")
	if got, want := f.Input("12345"), "12345"; got != want {
		t.Errorf("Input() = %q, want %q", got, want)
	}
}

func onlyDigits(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			out = append(out, s[i])
		}
	}
	return string(out)
}
