package phonex

import (
	"testing"
)

func TestFingerprint(t *testing.T) {
	p, _ := Parse("+998901234567")

	fp1 := Fingerprint(p)
	if fp1 == "" {
		t.Error("Fingerprint is empty")
	}

	fp2 := Fingerprint(p, WithSecret("my-secret"))
	if fp2 == "" || fp1 == fp2 {
		t.Errorf("Fingerprint with secret is invalid")
	}

	var nilPhone *Phone
	if got := Fingerprint(nilPhone); got != "" {
		t.Errorf("Nil phone fingerprint = %v", got)
	}

	p.Hash(MD5)
	p.Hash(SHA1)
	p.Hash(SHA512)
}
