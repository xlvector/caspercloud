package caspercloud

import (
	"testing"
)

func TestRSA(t *testing.T) {
	pk, err := generateRSAKey()
	if err != nil {
		t.Error(err)
	}
	t.Log(string(privateKeyString(pk)))
	t.Log(string(publicKeyString(&pk.PublicKey)))
}
