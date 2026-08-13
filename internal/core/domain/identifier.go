// Package domain owns ecosystem-neutral Heliopause value types and lifecycle rules.
package domain

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"strings"
)

const randomIDBytes = 16

var idEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// OperationID identifies one user-requested operation. Its zero value is invalid.
type OperationID struct{ value string }

// NewOperationID creates a cryptographically random operation identifier.
func NewOperationID() (OperationID, error) {
	value, err := generateID("op_", rand.Reader)
	if err != nil {
		return OperationID{}, fmt.Errorf("generate operation ID: %w", err)
	}
	return OperationID{value: value}, nil
}

// ParseOperationID validates a serialized operation identifier.
func ParseOperationID(value string) (OperationID, error) {
	if err := validateID(value, "op_"); err != nil {
		return OperationID{}, err
	}
	return OperationID{value: value}, nil
}

// String returns the canonical serialized identifier.
func (id OperationID) String() string { return id.value }

// RunID identifies one inspection run. Its zero value is invalid.
type RunID struct{ value string }

// NewRunID creates a cryptographically random run identifier.
func NewRunID() (RunID, error) {
	value, err := generateID("run_", rand.Reader)
	if err != nil {
		return RunID{}, fmt.Errorf("generate run ID: %w", err)
	}
	return RunID{value: value}, nil
}

// ParseRunID validates a serialized run identifier.
func ParseRunID(value string) (RunID, error) {
	if err := validateID(value, "run_"); err != nil {
		return RunID{}, err
	}
	return RunID{value: value}, nil
}

// String returns the canonical serialized identifier.
func (id RunID) String() string { return id.value }

func generateID(prefix string, reader io.Reader) (string, error) {
	if reader == nil {
		return "", errors.New("random reader is nil")
	}
	payload := make([]byte, randomIDBytes)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return "", fmt.Errorf("read random payload: %w", err)
	}
	return prefix + strings.ToLower(idEncoding.EncodeToString(payload)), nil
}

func validateID(value, prefix string) error {
	encoded := strings.TrimPrefix(value, prefix)
	if encoded == value || len(encoded) != idEncoding.EncodedLen(randomIDBytes) || encoded != strings.ToLower(encoded) {
		return fmt.Errorf("invalid %s identifier", strings.TrimSuffix(prefix, "_"))
	}
	payload, err := idEncoding.DecodeString(strings.ToUpper(encoded))
	if err != nil || len(payload) != randomIDBytes || strings.ToLower(idEncoding.EncodeToString(payload)) != encoded {
		return fmt.Errorf("invalid %s identifier", strings.TrimSuffix(prefix, "_"))
	}
	return nil
}
