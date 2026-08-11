//go:build !race

package phonex

// raceEnabled reports whether the binary was built with the race detector,
// which adds allocations of its own and so invalidates allocation counts.
const raceEnabled = false
