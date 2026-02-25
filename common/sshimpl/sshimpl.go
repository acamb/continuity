package sshimpl

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

type SSHKey struct {
	PrivateKey ssh.Signer
	PublicKey  ssh.PublicKey
}

func ReadSshKey(file string) (*SSHKey, error) {
	keyData, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("error while reading key file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err == nil {
		return &SSHKey{
			PrivateKey: signer,
			PublicKey:  signer.PublicKey(),
		}, nil
	}
	return nil, fmt.Errorf("error while parsing SSH key: %w", err)
}

func ReadAuthorizedKeys(file string) ([]ssh.PublicKey, error) {
	keyData, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("error while reading authorized keys file: %w", err)
	}
	lines := strings.Split(string(keyData), "\n")
	keys := make([]ssh.PublicKey, 0)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
		if err != nil {
			return nil, fmt.Errorf("error while parsing authorized key: %w", err)
		}
		keys = append(keys, key)
	}
	if len(keys) > 0 {
		return keys, nil
	}
	return nil, fmt.Errorf("no valid keys found in authorized keys file")
}

func Crypt(key *SSHKey, data []byte) ([]byte, error) {
	signature, err := key.PrivateKey.Sign(rand.Reader, data)
	if err != nil {
		return nil, fmt.Errorf("error while signing data: %w", err)
	}
	return signature.Blob, nil
}

func Verify(key ssh.PublicKey, signature []byte, data []byte) error {
	return key.Verify(data, &ssh.Signature{
		Format: key.Type(),
		Blob:   signature,
	})
}
