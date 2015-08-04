package caspercloud

import (
	"crypto/rsa"
	"fmt"
	"math/big"
	"strconv"
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

func TestTaobaoRSA(t *testing.T) {
	pbk := "9a39c3fefeadf3d194850ef3a1d707dfa7bec0609a60bfcc7fe4ce2c615908b9599c8911e800aff684f804413324dc6d9f982f437e95ad60327d221a00a2575324263477e4f6a15e3b56a315e0434266e092b2dd5a496d109cb15875256c73a2f0237c5332de28388693c643c8764f137e28e8220437f05b7659f58c4df94685"
	n := new(big.Int)
	_, err := fmt.Sscanf(pbk, "%x", n)
	if err != nil {
		t.Error(err)
	}
	t.Log(n.String())
	e, err := strconv.ParseInt("10001", 16, 64)
	if err != nil {
		t.Error(err)
	}
	t.Log(e)
	publicKey := &rsa.PublicKey{
		N: n,
		E: int(e),
	}
	password := []byte("xxxxxxxx")
	out, err := PKCS1Pad2Encrypt(password, publicKey)
	if err != nil {
		t.Error(err)
	}
	t.Log(out)
}
