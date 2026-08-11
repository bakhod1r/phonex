package phonex

import "strings"

type MaskOptions struct {
	Prefix int
	Suffix int
	Mask   string
}

var (
	MaskLast4  = MaskOptions{Prefix: 0, Suffix: 4, Mask: "*"} // masks everything except last 4
	MaskFirst3 = MaskOptions{Prefix: 3, Suffix: 0, Mask: "*"} // masks everything except first 3
	MaskMiddle = MaskOptions{Prefix: 4, Suffix: 2, Mask: "*"} // e.g. +99890****67
	MaskFull   = MaskOptions{Prefix: 0, Suffix: 0, Mask: "*"}
)

func (p *Phone) Mask(opts ...MaskOptions) string {
	e164 := p.E164()
	if len(e164) == 0 {
		return ""
	}

	opt := MaskMiddle // Default to middle masking for safety
	if len(opts) > 0 {
		opt = opts[0]
	}

	if len(e164) <= opt.Prefix+opt.Suffix {
		return strings.Repeat(opt.Mask, len(e164))
	}

	prefixStr := e164[:opt.Prefix]
	suffixStr := e164[len(e164)-opt.Suffix:]
	maskLen := len(e164) - opt.Prefix - opt.Suffix

	return prefixStr + strings.Repeat(opt.Mask, maskLen) + suffixStr
}

// Redact is a convenience function for masking a phone number for logs/output.
func Redact(p *Phone) string {
	if p == nil {
		return ""
	}
	return p.Mask(MaskMiddle)
}
