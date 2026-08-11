package phonex

import (
	"sync"
	"testing"
)

// TestConcurrentReadersAndParsers exercises the shared state the library
// keeps: the lazily compiled metadata patterns and the template cache. Each
// goroutine owns its own Phone, which is the documented usage.
func TestConcurrentReadersAndParsers(t *testing.T) {
	numbers := []string{
		"+998901234567", "+12025550123", "+442070313000", "+61234567890",
		"+74951234567", "+4930901820", "+81332249999", "+33612345678",
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var p Phone
			f := NewFormatter("US")
			for n := 0; n < 200; n++ {
				number := numbers[(i+n)%len(numbers)]
				if err := p.Parse(number); err != nil {
					t.Errorf("Parse(%q): %v", number, err)
					return
				}
				_ = p.Type()
				_ = p.IsValid()
				_ = p.National()
				_ = p.International()
				f.Clear()
				f.Input("2025550123")
			}
		}(i)
	}
	wg.Wait()
}

// TestConcurrentReadsOfOnePhone covers the guarantee the documentation makes:
// a Phone that is no longer being parsed into may be read from several
// goroutines at once.
func TestConcurrentReadsOfOnePhone(t *testing.T) {
	p, err := Parse("+442070313000")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 200; n++ {
				if p.Type() != FixedLine {
					t.Error("Type() disagreed between goroutines")
					return
				}
				_ = p.IsValid()
				_ = p.E164()
			}
		}()
	}
	wg.Wait()
}
