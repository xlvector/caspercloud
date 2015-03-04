package caspercloud

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

func generateRSAKey() (*rsa.PrivateKey, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	privKey.Precompute()
	err = privKey.Validate()
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

func publicKeyString(key *rsa.PublicKey) []byte {
	b, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil
	}
	pbs := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: b,
	})
	return pbs
}

func privateKeyString(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		})
}
