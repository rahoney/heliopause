//go:build !linux

package main

import (
	"context"
	"errors"
)

func runOperation(context.Context, string) error {
	return errors.New("protected release activation requires Linux amd64")
}
