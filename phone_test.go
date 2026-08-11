package phonex

import "testing"

func TestPhoneAccessors(t *testing.T) {
	p, err := Parse("+998 90 123 45 67")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checks := []struct {
		name      string
		got, want string
	}{
		{"Country", p.Country(), "UZ"},
		{"ISO2", p.ISO2(), "UZ"},
		{"ISO3", p.ISO3(), "UZB"},
		{"CountryName", p.CountryName(), "Uzbekistan"},
		{"DialCode", p.DialCode(), "998"},
		{"CountryCode", p.CountryCode(), "+998"},
		{"NSN", p.NSN(), "901234567"},
		{"NationalDigits", p.NationalDigits(), "901234567"},
		{"Digits", p.Digits(), "998901234567"},
		{"String", p.String(), "+998901234567"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s() = %q, want %q", c.name, c.got, c.want)
		}
	}
	if p.Metadata() == nil {
		t.Error("Metadata() = nil")
	}
}

func TestClone(t *testing.T) {
	p, err := Parse("+998901234567")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c := p.Clone()
	if err := p.Parse("+12025550123"); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := c.E164(), "+998901234567"; got != want {
		t.Errorf("clone changed with the original: %q, want %q", got, want)
	}
	if (*Phone)(nil).Clone() != nil {
		t.Error("cloning nil should give nil")
	}
}
