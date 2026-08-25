//go:build linux

package bootstrap

import (
	"context"
	"strings"

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func doctorHostChecks(ctx context.Context) []cli.DoctorCheck {
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		return []cli.DoctorCheck{{Name: "trusted-host-runtime", Detail: "TRUSTED_HOST_RUNTIME_UNAVAILABLE"}, {Name: "observer-helper", Detail: "OBSERVER_HELPER_UNAVAILABLE"}, {Name: "network-policy-service", Detail: "NETWORK_POLICY_SERVICE_UNAVAILABLE"}, {Name: "runtime-images", Detail: "RUNTIME_IMAGES_UNAVAILABLE"}}
	}
	defer executor.Close()
	checks := []cli.DoctorCheck{{Name: "trusted-host-runtime", Healthy: true, Detail: "OK"}}
	if _, err := hosttool.NewSystemObserverLauncher(); err != nil {
		checks = append(checks, cli.DoctorCheck{Name: "observer-helper", Detail: "OBSERVER_HELPER_UNAVAILABLE"})
	} else {
		checks = append(checks, cli.DoctorCheck{Name: "observer-helper", Healthy: true, Detail: "OK"})
	}
	if _, err := hosttool.NewSystemNetworkPolicyClient(); err != nil {
		checks = append(checks, cli.DoctorCheck{Name: "network-policy-service", Detail: "NETWORK_POLICY_SERVICE_UNAVAILABLE"})
	} else {
		checks = append(checks, cli.DoctorCheck{Name: "network-policy-service", Healthy: true, Detail: "OK"})
	}
	node, nodeErr := executor.Output(ctx, "docker", "image", "inspect", runtimeidentity.NodeImageReference, "--format", "{{.Id}}")
	python, pythonErr := executor.Output(ctx, "docker", "image", "inspect", runtimeidentity.PythonImageReference, "--format", "{{.Id}}")
	if nodeErr != nil || pythonErr != nil || strings.TrimSpace(string(node)) == "" || strings.TrimSpace(string(python)) == "" {
		checks = append(checks, cli.DoctorCheck{Name: "runtime-images", Detail: "RUNTIME_IMAGES_UNAVAILABLE"})
	} else {
		checks = append(checks, cli.DoctorCheck{Name: "runtime-images", Healthy: true, Detail: "OK"})
	}
	return checks
}
