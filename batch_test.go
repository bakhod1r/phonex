package phonex

import "testing"

func TestParseMany(t *testing.T) {
	results := ParseMany([]string{"+998901234567", "nonsense", "+12025550123"})
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	if results[0].Error != nil || results[0].Phone.E164() != "+998901234567" {
		t.Errorf("first result = %+v", results[0])
	}
	if results[1].Error == nil {
		t.Error("second result should carry an error")
	}
	if results[1].Phone != nil {
		t.Error("a failed result should not carry a phone")
	}
	if results[2].Error != nil || results[2].Phone.E164() != "+12025550123" {
		t.Errorf("third result = %+v", results[2])
	}
}

func TestUnique(t *testing.T) {
	got := Unique([]string{
		"+998901234567",
		"+998 90 123 45 67",
		"998901234567",
		"+12025550123",
		"nonsense",
	})
	want := []string{"+998901234567", "+12025550123"}
	if len(got) != len(want) {
		t.Fatalf("Unique() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Unique()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSortNumbers(t *testing.T) {
	got := SortNumbers([]string{"+998901234567", "+12025550123", "+442070313000"})
	want := []string{"+12025550123", "+442070313000", "+998901234567"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SortNumbers() = %v, want %v", got, want)
		}
	}
}
