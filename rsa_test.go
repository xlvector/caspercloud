package caspercloud

import (
	"testing"
)

func TestRSA(t *testing.T) {
	pk, err := GenerateRSAKey()
	if err != nil {
		t.Error(err)
	}
	t.Log(string(PrivateKeyString(pk)))
	t.Log(string(PublicKeyString(&pk.PublicKey)))
}
