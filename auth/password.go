// Package auth provides password hashing, session-backed login, and user
// middleware.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLength  = 16
	argonKeyLength   = 32
)

// ErrEmptyPassword is returned when hashing an empty password.
var ErrEmptyPassword = errors.New("auth: password is empty")

// HashPassword hashes password with argon2id and returns a PHC string.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: read salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonIterations,
		argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// CheckPassword reports whether password matches the encoded hash.
func CheckPassword(encoded, password string) bool {
	params, salt, key, err := decodeHash(encoded)
	if err != nil {
		return false
	}
	derived := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, uint32(len(key)))
	return subtle.ConstantTimeCompare(derived, key) == 1
}

type hashParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func decodeHash(encoded string) (hashParams, []byte, []byte, error) {
	var params hashParams
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return params, nil, nil, errors.New("auth: invalid hash")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return params, nil, nil, errors.New("auth: unsupported hash version")
	}
	memory, iterations, parallelism := 0, 0, 0
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return params, nil, nil, errors.New("auth: invalid hash parameters")
	}
	if memory <= 0 || memory > 256*1024 || iterations <= 0 || iterations > 10 || parallelism <= 0 || parallelism > 16 {
		return params, nil, nil, errors.New("auth: hash parameters out of range")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return params, nil, nil, errors.New("auth: invalid salt")
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return params, nil, nil, errors.New("auth: invalid key")
	}
	params.memory = uint32(memory)
	params.iterations = uint32(iterations)
	params.parallelism = uint8(parallelism)
	return params, salt, key, nil
}
