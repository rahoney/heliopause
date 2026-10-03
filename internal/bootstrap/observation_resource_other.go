//go:build !linux

package bootstrap

import "github.com/rahoney/heliopause/internal/sandbox"

func newObservationResourceAdapter() sandbox.ObservationResourceClient { return nil }
