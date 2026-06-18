// Package auth handles password hashing, server-side sessions, CSRF tokens,
// and login rate limiting for the admin API.
package auth

import "golang.org/x/crypto/bcrypt"

// bcryptCost is the work factor for password hashing. 12 is a reasonable
// public-facing default (~250ms/hash on typical hardware).
const bcryptCost = 12

// Hash returns a bcrypt hash of the password.
func Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(b), err
}

// Verify reports whether password matches the stored bcrypt hash.
func Verify(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
