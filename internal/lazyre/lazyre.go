// Package lazyre provides a regular expression that is compiled on first use.
//
// The phone number metadata holds several thousand patterns, and a typical
// program touches a handful of them. Compiling them all at init would cost
// tens of milliseconds of start-up and a few megabytes of resident memory for
// patterns that are never matched against anything.
package lazyre

import (
	"regexp"
	"sync"
)

// Re is a lazily compiled regular expression.
//
// The zero value holds no pattern and never matches. A Re must not be copied
// once used: it carries a sync.Once. Generated metadata is therefore always
// reached through a pointer.
type Re struct {
	// Src is the pattern, already anchored by whoever generated it.
	Src string

	once sync.Once
	re   *regexp.Regexp
}

// Empty reports whether no pattern was defined.
func (l *Re) Empty() bool { return l.Src == "" }

// Source returns the pattern this expression was built from.
func (l *Re) Source() string { return l.Src }

// Regexp compiles the pattern on first use and returns it, or nil if no
// pattern was defined. It is safe to call from several goroutines.
func (l *Re) Regexp() *regexp.Regexp {
	if l.Src == "" {
		return nil
	}
	l.once.Do(func() { l.re = regexp.MustCompile(l.Src) })
	return l.re
}

// Match reports whether s matches. An undefined pattern never matches.
func (l *Re) Match(s string) bool {
	re := l.Regexp()
	return re != nil && re.MatchString(s)
}
