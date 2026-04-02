// Package passphrase provides a concurrency-safe store for the shared passphrase.
package passphrase

import (
	"errors"
	"strings"
	"sync"
)

const minLength = 8
const maxLength = 128

var (
	ErrTooShort        = errors.New("passphrase must be at least 8 characters")
	ErrTooLong         = errors.New("passphrase must be at most 128 characters")
	ErrLeadingSpace    = errors.New("passphrase must not start with whitespace")
	ErrTrailingSpace   = errors.New("passphrase must not end with whitespace")
)

// Validate returns an error if the passphrase does not meet requirements.
func Validate(p string) error {
	if len(p) < minLength {
		return ErrTooShort
	}
	if len(p) > maxLength {
		return ErrTooLong
	}
	if p != strings.TrimLeft(p, " \t") {
		return ErrLeadingSpace
	}
	if p != strings.TrimRight(p, " \t") {
		return ErrTrailingSpace
	}
	return nil
}

// Store holds the current passphrase and allows safe concurrent access.
type Store struct {
	mu    sync.RWMutex
	value string
}

// Get returns the current passphrase.
func (s *Store) Get() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

// Set updates the passphrase.
func (s *Store) Set(passphrase string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = passphrase
}
