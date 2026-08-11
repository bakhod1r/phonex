package lazyre

import (
	"sync"
	"testing"
)

func TestZeroValueNeverMatches(t *testing.T) {
	var re Re
	if !re.Empty() {
		t.Error("Empty() = false for the zero value")
	}
	if re.Regexp() != nil {
		t.Error("Regexp() should be nil when no pattern is set")
	}
	if re.Match("anything") {
		t.Error("Match() = true for the zero value")
	}
	if re.Source() != "" {
		t.Errorf("Source() = %q, want empty", re.Source())
	}
}

func TestMatch(t *testing.T) {
	re := Re{Src: `^(?:[49]\d{7})$`}
	if !re.Match("44091103") {
		t.Error("Match() = false for a number the pattern accepts")
	}
	if re.Match("14091103") {
		t.Error("Match() = true for a number the pattern rejects")
	}
	if re.Source() != `^(?:[49]\d{7})$` {
		t.Errorf("Source() = %q", re.Source())
	}
}

// TestCompilesOnce checks that the pattern is compiled exactly once, which is
// the whole point of the type.
func TestCompilesOnce(t *testing.T) {
	re := Re{Src: `^(?:\d+)$`}
	first := re.Regexp()
	if first == nil {
		t.Fatal("Regexp() = nil")
	}
	if second := re.Regexp(); second != first {
		t.Error("Regexp() returned a different instance on the second call")
	}
}

// TestConcurrentUse is the guarantee the metadata relies on: many goroutines
// reaching the same pattern for the first time at once.
func TestConcurrentUse(t *testing.T) {
	re := Re{Src: `^(?:1\d{2})$`}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 100; n++ {
				if !re.Match("112") {
					t.Error("Match() = false under concurrent use")
					return
				}
			}
		}()
	}
	wg.Wait()
}
