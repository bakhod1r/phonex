package phonex

import "testing"

func BenchmarkParseInternational(b *testing.B) {
	var p Phone
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := p.Parse("+998901234567"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseNational(b *testing.B) {
	var p Phone
	options := DefaultParseOptions()
	options.DefaultCountry = "GB"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := p.ParseWith("020 7031 3000", options); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseSharedCallingCode(b *testing.B) {
	var p Phone
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := p.Parse("+14155552671"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseAllocating(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Parse("+998901234567"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIsValid(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !IsValid("+998901234567") {
			b.Fatal("expected a valid number")
		}
	}
}

func BenchmarkE164(b *testing.B) {
	p, err := Parse("+998901234567")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = p.E164()
	}
}

func BenchmarkAppendE164(b *testing.B) {
	p, err := Parse("+998901234567")
	if err != nil {
		b.Fatal(err)
	}
	buf := make([]byte, 0, 32)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = p.AppendE164(buf[:0])
	}
}

func BenchmarkFormatNational(b *testing.B) {
	p, err := Parse("+442070313000")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = p.National()
	}
}

func BenchmarkAsYouType(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f := NewFormatter("US")
		f.Input("2025550123")
	}
}
