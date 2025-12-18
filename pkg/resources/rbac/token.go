package rbac

import (
	"crypto/rand"
	"fmt"
)

const (
	tokenIDLen     = 6
	tokenSecretLen = 16
	tokenAlphabet  = "abcdefghijklmnopqrstuvwxyz0123456789"
)

type BootstrapToken struct {
	ID     string
	Secret string
}

func NewBootstrapToken() (BootstrapToken, error) {
	id, err := randString(tokenIDLen)
	if err != nil {
		return BootstrapToken{}, err
	}
	sec, err := randString(tokenSecretLen)
	if err != nil {
		return BootstrapToken{}, err
	}
	return BootstrapToken{ID: id, Secret: sec}, nil
}

func randString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = tokenAlphabet[int(b[i])%len(tokenAlphabet)]
	}
	return string(out), nil
}

func (t BootstrapToken) String() string {
	return fmt.Sprintf("%s.%s", t.ID, t.Secret)
}
