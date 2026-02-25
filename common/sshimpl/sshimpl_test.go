package sshimpl

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
)

func generateSSHKey(keytype string) (*SSHKey, error) {
	var privateKey ssh.Signer
	switch keytype {
	case "rsa":
		pk, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("error generating RSA key: %w", err)
		}
		privateKey, err = ssh.NewSignerFromKey(pk)
		if err != nil {
			return nil, fmt.Errorf("error generating RSA key: %w", err)
		}
	case "ed25519":
		_, pk, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("error generating Ed25519 key: %w", err)
		}
		privateKey, err = ssh.NewSignerFromKey(pk)
		if err != nil {
			return nil, fmt.Errorf("error generating signer Ed25519 key: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keytype)
	}
	return &SSHKey{
		PrivateKey: privateKey,
		PublicKey:  privateKey.PublicKey(),
	}, nil
}

func TestDecryptRSA(t *testing.T) {
	key, err := generateSSHKey("rsa")
	assert.NoError(t, err)
	crypt, err := Crypt(key, []byte("test data"))
	assert.NoError(t, err)
	err = Verify(key.PublicKey, crypt, []byte("test data"))
	assert.NoError(t, err)
}

func TestDecryptED25519(t *testing.T) {
	key, err := generateSSHKey("ed25519")
	assert.NoError(t, err)
	crypt, err := Crypt(key, []byte("test data"))
	assert.NoError(t, err)
	err = Verify(key.PublicKey, crypt, []byte("test data"))
	assert.NoError(t, err)
}
