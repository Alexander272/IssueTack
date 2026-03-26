package hasher

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"time"
)

// PasswordHasher provides hashing logic to securely store passwords.
type PasswordHasher interface {
	Hash(password, salt string) (string, error)
	GenerateSalt() (string, error)
}

// SHA256Hasher uses SHA256 to hash passwords with provided salt size.
type SHA256Hasher struct {
	saltSize uint8
}

func NewSHA256Hasher(saltSize uint8) *SHA256Hasher {
	return &SHA256Hasher{saltSize: saltSize}
}

// Hash creates SHA1 hash of given password.
func (h *SHA256Hasher) Hash(password, salt string) (string, error) {
	hash := sha256.New()

	if _, err := hash.Write([]byte(password)); err != nil {
		return "", err
	}

	// salt, err := h.GenerateSalt()
	// if err != nil {
	// 	return "", "", fmt.Errorf("failed to generate salt. err: %w", err)
	// }

	return fmt.Sprintf("%x", hash.Sum([]byte(salt))), nil
}

func (h *SHA256Hasher) GenerateSalt() (string, error) {
	b := make([]byte, h.saltSize)

	s := rand.NewSource(time.Now().Unix())
	r := rand.New(s)

	if _, err := r.Read(b); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", b), nil
}
