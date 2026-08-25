//go:build !linux

package bootstrap

import (
	"context"

	"github.com/rahoney/heliopause/internal/cli"
)

func doctorHostChecks(context.Context) []cli.DoctorCheck {
	return []cli.DoctorCheck{
		{Name: "trusted-host-runtime", Detail: "LINUX_AMD64_REQUIRED"},
		{Name: "observer-helper", Detail: "LINUX_AMD64_REQUIRED"},
		{Name: "network-policy-service", Detail: "LINUX_AMD64_REQUIRED"},
		{Name: "runtime-images", Detail: "LINUX_AMD64_REQUIRED"},
	}
}
