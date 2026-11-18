package main

const ciResponderBuild = `CGO_ENABLED=0 go build -trimpath -buildvcs=true -ldflags='-s -w' -o "$RUNNER_TEMP/haa-sandbox-responder" ./cmd/haa-sandbox-responder`

func validateResponderSourceOrder(contents string) []string {
	commands := []string{ciResponderBuild, "source scripts/ci-responder-source.sh", "haa_ci_prepare_responder_base"}
	if findings := validateLifecycleIntegrationCommands(contents, "responder source", commands); len(findings) != 0 {
		return findings
	}
	blocks, err := activeJobRunLines(contents, "gvisor-integration", "Install pinned runtime and run lifecycle integration")
	if err != nil {
		return []string{"CI responder source requires an available parsed lifecycle run block"}
	}
	for _, lines := range blocks {
		next := 0
		for _, line := range lines {
			if line == "haa_start_policy_helper" {
				if next == len(commands) {
					return nil
				}
				break
			}
			if next < len(commands) && line == commands[next] {
				next++
			}
		}
	}
	return []string{"CI responder source requires the fixed static build and protected fixture before the first helper/client cohort"}
}
