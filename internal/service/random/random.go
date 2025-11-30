// Package random is used for generating random bytes and new user id.
package random

import (
	"crypto/rand"

	"github.com/google/uuid"
)

// Generator generates random bytes and new user id.
type Generator interface {
	GenerateRandomBytes(size int) ([]byte, error)
	GenerateNewUserID() string
}

// TrulyRandomGenerator is used for generating truly random values.
type TrulyRandomGenerator struct{}

// GenerateRandomBytes generates size random bytes.
func (g *TrulyRandomGenerator) GenerateRandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateNewUserID generates new UUID.
func (g *TrulyRandomGenerator) GenerateNewUserID() string {
	return uuid.NewString()
}
