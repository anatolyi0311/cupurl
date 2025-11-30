// Package crypto is used for encrypting and decrypting values.
package crypto

// Cryptographer encrypts and decrypts values.
type Cryptographer interface {
	Encrypt(src []byte) ([]byte, error)
	Decrypt(src []byte) ([]byte, error)
}
