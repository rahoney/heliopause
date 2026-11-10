package terraformprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func upgradeVersionDocument(t *testing.T, doc versionDocument) []byte {
	t.Helper()
	body, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	// Platform is an existing decode DTO without marshal tags; registry wire
	// keys are lowercase, which the independent strict JSON validator requires.
	body = bytes.ReplaceAll(body, []byte(`"OS":`), []byte(`"os":`))
	return bytes.ReplaceAll(body, []byte(`"Arch":`), []byte(`"arch":`))
}

func upgradeVersions(t *testing.T, versions ...string) []byte {
	t.Helper()
	doc := versionDocument{}
	for _, version := range versions {
		doc.Versions = append(doc.Versions, versionEntry{Version: version, Protocols: []string{"5.0"}, Platforms: []Platform{{OS: "linux", Arch: "amd64"}}})
	}
	return upgradeVersionDocument(t, doc)
}

func TestTerraformUpgradeSelectsNewestStableAcrossAllConstraints(t *testing.T) {
	body := upgradeVersions(t, "1.9.0", "2.0.0", "1.10.0", "1.11.0-rc.1")
	for _, tc := range []struct {
		constraints []string
		want        string
	}{
		{[]string{">=1.0.0", "<2.0.0"}, "1.10.0"},
		{[]string{"~>1.9.0"}, "1.9.0"},
		{nil, "2.0.0"},
	} {
		got, err := selectUpgradeVersion(context.Background(), body, tc.constraints, Platform{"linux", "amd64"})
		if err != nil || got != tc.want {
			t.Fatalf("selection=%q error=%v want=%q", got, err, tc.want)
		}
	}
	for _, constraints := range [][]string{{">3.0.0"}, {">=1.11.0-rc.1"}, {"not a version"}} {
		if _, err := selectUpgradeVersion(context.Background(), body, constraints, Platform{"linux", "amd64"}); err == nil {
			t.Fatal("unsupported/conflicting constraint accepted", constraints)
		}
	}
}

func TestTerraformUpgradeDoesNotDowngradeForUnsupportedNewest(t *testing.T) {
	for _, fault := range []string{"platform", "protocol", "duplicate", "identity", "trailing"} {
		t.Run(fault, func(t *testing.T) {
			var doc versionDocument
			if err := json.Unmarshal(upgradeVersions(t, "1.0.0", "1.1.0"), &doc); err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "platform":
				doc.Versions[1].Platforms = []Platform{{"darwin", "arm64"}}
			case "protocol":
				doc.Versions[1].Protocols = nil
			case "duplicate":
				doc.Versions = append(doc.Versions, doc.Versions[1])
			case "identity":
				doc.Versions = append(doc.Versions, versionEntry{Version: "latest"})
			}
			body := upgradeVersionDocument(t, doc)
			if fault == "trailing" {
				body = append(body, []byte(" {}")...)
			}
			if _, err := selectUpgradeVersion(context.Background(), body, nil, Platform{"linux", "amd64"}); err == nil {
				t.Fatal("invalid newest/list silently selected an older provider")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := selectUpgradeVersion(ctx, upgradeVersions(t, "1.0.0"), nil, Platform{"linux", "amd64"}); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation cause lost", err)
	}
}
