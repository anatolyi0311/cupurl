package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"

	"github.com/anatolyi0311/cupurl/internal/service/random"
)

// GCMAESCryptographer is cryptographer based on AES with Galois/Counter Mode.
//
// AES with Galois/Counter Mode (AES-GCM) provides both authenticated encryption (confidentiality and authentication)
// and the ability to check the integrity and authentication of additional
// authenticated data (AAD) that is sent in the clear.
type GCMAESCryptographer struct {
	Random random.Generator // random number generator. It's used to generate the nonce
	Key    []byte           // key used to encrypt and decrypt the data
}

// Encrypt encrypts the plaintext using AES-GCM.
func (c *GCMAESCryptographer) Encrypt(plaintext []byte) ([]byte, error) {
	aesblock, err := aes.NewCipher(c.Key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return nil, err
	}

	nonce, err := c.Random.GenerateRandomBytes(aesgcm.NonceSize())
	if err != nil {
		return nil, err
	}

	return aesgcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts the ciphertext using AES-GCM.
func (c *GCMAESCryptographer) Decrypt(ciphertext []byte) ([]byte, error) {
	aesblock, err := aes.NewCipher(c.Key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext is too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	return aesgcm.Open(nil, nonce, ciphertext, nil) // расшифровываем
}
