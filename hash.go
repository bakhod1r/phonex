package phonex

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
)

type HashAlgorithm int

const (
	SHA256 HashAlgorithm = iota
	SHA512
	SHA1
	MD5
)

// Hash returns a hash of the phone number.
func (p *Phone) Hash(algo ...HashAlgorithm) string {
	a := SHA256
	if len(algo) > 0 {
		a = algo[0]
	}

	e164 := []byte(p.E164())

	switch a {
	case SHA512:
		h := sha512.Sum512(e164)
		return hex.EncodeToString(h[:])
	case SHA1:
		h := sha1.Sum(e164)
		return hex.EncodeToString(h[:])
	case MD5:
		h := md5.Sum(e164)
		return hex.EncodeToString(h[:])
	case SHA256:
		fallthrough
	default:
		h := sha256.Sum256(e164)
		return hex.EncodeToString(h[:])
	}
}

type HashOptions struct {
	Secret string
}

type HashOption func(*HashOptions)

func WithSecret(s string) HashOption {
	return func(o *HashOptions) { o.Secret = s }
}

// Fingerprint returns a stable hash of the phone number.
// If WithSecret is provided, it uses HMAC-SHA256, otherwise it uses standard SHA256.
func Fingerprint(p *Phone, opts ...HashOption) string {
	if p == nil {
		return ""
	}
	options := HashOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	e164 := p.E164()
	if options.Secret != "" {
		h := hmac.New(sha256.New, []byte(options.Secret))
		h.Write([]byte(e164))
		return hex.EncodeToString(h.Sum(nil))
	}
	return p.Hash(SHA256)
}
