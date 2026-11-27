//go:build linux

package releaseactivation

import (
	"context"
	"testing"
)

func TestEnrollmentRejectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Register(ctx); err == nil {
		t.Fatal("canceled registration accepted")
	}
}

func TestNativeConfigSharedValidationRetainsJournalGate(t *testing.T) {
	journal := runtimeConfigFixture(t)
	if _, _, err := parseNativeInstallationConfigs(journal.Old, journal.Configs); err != nil {
		t.Fatal(err)
	}
	journal.State = "unknown"
	if _, _, err := parseNativeConfigs(journal); err == nil {
		t.Fatal("invalid journal bypassed")
	}
	for _, field := range []string{"hash", "mode", "gid", "body", "record"} {
		t.Run(field, func(t *testing.T) {
			j := runtimeConfigFixture(t)
			switch field {
			case "hash":
				j.Configs[0].SHA256 = "invalid"
			case "mode":
				j.Configs[1].Mode = 0666
			case "gid":
				j.Configs[2].GID = 1 << 31
			case "body":
				j.Configs[1].Body = nil
			case "record":
				j.Old.ClientUID = 0
			}
			if _, _, err := parseNativeConfigs(j); err == nil {
				t.Fatal("journal accepted invalid config")
			}
			if _, _, err := parseNativeInstallationConfigs(j.Old, j.Configs); err == nil {
				t.Fatal("enrollment accepted invalid config")
			}
		})
	}
}
