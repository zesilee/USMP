package netsim

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

// generateHostKey generates a new RSA host key for the SSH server
func generateHostKey() (ssh.Signer, error) {
	// Generate a new RSA key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Marshal the private key to PEM format
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	pemBytes := pem.EncodeToMemory(privBlock)

	// Parse the private key into SSH signer
	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return nil, err
	}

	return signer, nil
}
