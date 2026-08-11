package phonex

import (
	"sync/atomic"
	"unsafe"

	"github.com/bakhod1r/phonex/countries"
)

// Phone is a parsed phone number.
//
// A Phone stores its digits inline, so parsing into an existing value does not
// allocate. The zero value is not a usable number; obtain one from Parse or
// (*Phone).Parse.
//
// A Phone is not safe for concurrent modification, but a Phone that is no
// longer being parsed into may be read from several goroutines.
type Phone struct {
	// nsn holds the national significant number as ASCII digits, including
	// any leading zeros that are significant in the region.
	nsn    [maxNSNLen]byte
	nsnLen uint8

	// ext holds the extension digits, without any "ext"/"x" marker.
	ext    [maxExtLen]byte
	extLen uint8

	meta *countries.Metadata

	// typ caches the resolved range, offset by one so that zero means "not
	// resolved yet". It is read and written atomically: resolving the type
	// is lazy, and a finished Phone is documented as safe to read from
	// several goroutines at once.
	typ uint32

	source CountryCodeSource

	// carrierCode is the domestic carrier selection code that was stripped
	// while parsing, if the region's rules define one.
	carrierCode string

	// raw is the input Parse was given, kept for RawInput and EqualExact.
	raw string

	// scratch is the working buffer Parse normalises into. Keeping it on
	// the Phone is what lets repeated parses into the same value run
	// without allocating.
	scratch [maxDigitsLen]byte
}

// NSN returns the national significant number: the digits after the calling
// code, with the national (trunk) prefix removed.
func (p *Phone) NSN() string {
	if p == nil || p.nsnLen == 0 {
		return ""
	}
	return string(p.nsn[:p.nsnLen])
}

// nsnRef returns the national significant number without copying. The result
// is only valid until the Phone is parsed into again, so it must not escape
// beyond the call that produced it.
func (p *Phone) nsnRef() string {
	if p.nsnLen == 0 {
		return ""
	}
	return unsafe.String(&p.nsn[0], int(p.nsnLen))
}

// NationalDigits is an alias for NSN.
func (p *Phone) NationalDigits() string { return p.NSN() }

// Digits returns every significant digit: the calling code followed by the
// national significant number, with no '+'.
func (p *Phone) Digits() string {
	if p == nil || p.meta == nil {
		return ""
	}
	return p.meta.DialCode + p.NSN()
}

// Extension returns the extension digits, or "" if the number has none.
func (p *Phone) Extension() string {
	if p == nil || p.extLen == 0 {
		return ""
	}
	return string(p.ext[:p.extLen])
}

// HasExtension reports whether an extension was parsed.
func (p *Phone) HasExtension() bool { return p != nil && p.extLen > 0 }

// CarrierCode returns the domestic carrier selection code stripped while
// parsing, or "" if there was none.
func (p *Phone) CarrierCode() string {
	if p == nil {
		return ""
	}
	return p.carrierCode
}

// RawInput returns the string this number was parsed from.
func (p *Phone) RawInput() string {
	if p == nil {
		return ""
	}
	return p.raw
}

// Source reports how the calling code was determined.
func (p *Phone) Source() CountryCodeSource {
	if p == nil {
		return FromDefaultCountry
	}
	return p.source
}

// Metadata returns the region metadata backing this number. The caller must
// not modify it.
func (p *Phone) Metadata() *Metadata {
	if p == nil {
		return nil
	}
	return p.meta
}

// Country returns the ISO-3166 alpha-2 region code, or "001" for numbers in a
// non-geographical range such as +800.
func (p *Phone) Country() string {
	if p == nil || p.meta == nil {
		return ""
	}
	return p.meta.ISO2
}

// ISO2 is an alias for Country.
func (p *Phone) ISO2() string { return p.Country() }

// ISO3 returns the ISO-3166 alpha-3 region code.
func (p *Phone) ISO3() string {
	if p == nil || p.meta == nil {
		return ""
	}
	return p.meta.ISO3
}

// CountryName returns the English region name.
func (p *Phone) CountryName() string {
	if p == nil || p.meta == nil {
		return ""
	}
	return p.meta.Name
}

// DialCode returns the calling code without a leading '+'.
func (p *Phone) DialCode() string {
	if p == nil || p.meta == nil {
		return ""
	}
	return p.meta.DialCode
}

// CountryCode returns the calling code with a leading '+'.
func (p *Phone) CountryCode() string {
	if p == nil || p.meta == nil {
		return ""
	}
	return "+" + p.meta.DialCode
}

// Timezones returns the IANA time zones of the region. The result must not be
// modified.
//
// The bundled metadata carries no time zones, so this currently returns nil
// for every region: libphonenumber does not publish them, and the prefix
// level data set that would is not vendored here. The accessor exists so that
// filling internal/metadata/metadata.json is all it takes to enable it.
func (p *Phone) Timezones() []string {
	if p == nil || p.meta == nil {
		return nil
	}
	return p.meta.Timezones
}

// IsNonGeographical reports whether the number belongs to a global range such
// as +800 (universal freephone) rather than to a country.
func (p *Phone) IsNonGeographical() bool {
	return p != nil && p.meta != nil && p.meta.ISO2 == "001"
}

// MobileNumberPortable reports whether the region supports number portability,
// meaning the range a number falls into does not reliably identify its carrier.
func (p *Phone) MobileNumberPortable() bool {
	return p != nil && p.meta != nil && p.meta.MobileNumberPortable
}

// Clone returns an independent copy.
func (p *Phone) Clone() *Phone {
	if p == nil {
		return nil
	}
	c := new(Phone)
	c.nsn = p.nsn
	c.nsnLen = p.nsnLen
	c.ext = p.ext
	c.extLen = p.extLen
	c.meta = p.meta
	c.source = p.source
	c.carrierCode = p.carrierCode
	c.raw = p.raw
	atomic.StoreUint32(&c.typ, atomic.LoadUint32(&p.typ))
	return c
}

// String returns the number in E.164 format, with any extension appended in
// RFC 3966 style. It is meant for logs and tests, not for display to users;
// use Format for that.
func (p *Phone) String() string {
	if p == nil || p.meta == nil {
		return ""
	}
	s := p.E164()
	if p.extLen > 0 {
		s += ";ext=" + p.Extension()
	}
	return s
}
