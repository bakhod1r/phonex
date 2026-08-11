package phonex

import "testing"

func TestFormats(t *testing.T) {
	tests := []struct {
		number        string
		e164          string
		international string
		national      string
		rfc3966       string
	}{
		{
			number:        "+998 90 123 45 67",
			e164:          "+998901234567",
			international: "+998 90 123 45 67",
			national:      "90 123 45 67",
			rfc3966:       "tel:+998-90-123-45-67",
		},
		{
			number:        "+1 202 555 0123",
			e164:          "+12025550123",
			international: "+1 202-555-0123",
			national:      "(202) 555-0123",
			rfc3966:       "tel:+1-202-555-0123",
		},
		{
			number:        "+44 20 7031 3000",
			e164:          "+442070313000",
			international: "+44 20 7031 3000",
			national:      "020 7031 3000",
			rfc3966:       "tel:+44-20-7031-3000",
		},
		{
			number:        "+61 2 3456 7890",
			e164:          "+61234567890",
			international: "+61 2 3456 7890",
			national:      "(02) 3456 7890",
			rfc3966:       "tel:+61-2-3456-7890",
		},
		{
			number:        "+7 495 123 45 67",
			e164:          "+74951234567",
			international: "+7 495 123-45-67",
			national:      "8 (495) 123-45-67",
			rfc3966:       "tel:+7-495-123-45-67",
		},
		{
			// A region that shares +1 is written the way the main region is.
			number:        "+1 242 302 1234",
			e164:          "+12423021234",
			international: "+1 242-302-1234",
			national:      "(242) 302-1234",
			rfc3966:       "tel:+1-242-302-1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			p, err := Parse(tt.number)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := p.E164(); got != tt.e164 {
				t.Errorf("E164() = %q, want %q", got, tt.e164)
			}
			if got := p.International(); got != tt.international {
				t.Errorf("International() = %q, want %q", got, tt.international)
			}
			if got := p.National(); got != tt.national {
				t.Errorf("National() = %q, want %q", got, tt.national)
			}
			if got := p.RFC3966(); got != tt.rfc3966 {
				t.Errorf("RFC3966() = %q, want %q", got, tt.rfc3966)
			}
			if got := p.Format(FormatE164); got != tt.e164 {
				t.Errorf("Format(FormatE164) = %q, want %q", got, tt.e164)
			}
			if got := p.Format(FormatNational); got != tt.national {
				t.Errorf("Format(FormatNational) = %q, want %q", got, tt.national)
			}
		})
	}
}

func TestRFC3966IncludesExtension(t *testing.T) {
	p, err := Parse("+1 202 555 0123 ext. 42")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := p.RFC3966(), "tel:+1-202-555-0123;ext=42"; got != want {
		t.Errorf("RFC3966() = %q, want %q", got, want)
	}
	if got, want := p.E164(), "+12025550123"; got != want {
		t.Errorf("E164() = %q, want %q (E.164 has no extension)", got, want)
	}
}

func TestOutOfCountry(t *testing.T) {
	tests := []struct {
		number string
		from   string
		want   string
	}{
		{"+44 20 7031 3000", "US", "011 44 20 7031 3000"},
		{"+44 20 7031 3000", "GB", "020 7031 3000"},
		{"+1 202 555 0123", "GB", "00 1 202-555-0123"},
		{"+1 202 555 0123", "US", "1 (202) 555-0123"},
		{"+998 90 123 45 67", "GB", "00 998 90 123 45 67"},
	}
	for _, tt := range tests {
		t.Run(tt.number+" from "+tt.from, func(t *testing.T) {
			p, err := Parse(tt.number)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := p.OutOfCountry(tt.from); got != tt.want {
				t.Errorf("OutOfCountry(%q) = %q, want %q", tt.from, got, tt.want)
			}
		})
	}
}

func TestAppendE164(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector allocates on its own, so the count is meaningless")
	}
	p, err := Parse("+998901234567")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	buf := make([]byte, 0, 32)
	allocs := testing.AllocsPerRun(100, func() { _ = p.AppendE164(buf[:0]) })
	if allocs != 0 {
		t.Errorf("AppendE164 allocated %v times per run, want 0", allocs)
	}
	if got := string(p.AppendE164(buf[:0])); got != "+998901234567" {
		t.Errorf("AppendE164 = %q", got)
	}
}
