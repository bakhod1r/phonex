package geocoding

import (
	"testing"

	"github.com/bakhod1r/phonex"
)

func BenchmarkArea(b *testing.B) {
	p, err := phonex.Parse("+44 20 7031 3000")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Area(p)
	}
}

func BenchmarkAreaForDigits(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = AreaForDigits("442070313000")
	}
}
