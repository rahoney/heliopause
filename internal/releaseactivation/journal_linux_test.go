//go:build linux

package releaseactivation

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func journalFixture() updateJournal {
	body := []byte("root-owned config fixture")
	config := savedConfig{body, bodySHA256(body), 0644, 0}
	return updateJournal{Schema: journalSchema, State: "PREPARED", Old: registration{registrationSchema, "NATIVE", "v1.0.0", strings.Repeat("a", 64), 1000, 1000}, Version: "v1.0.1", ManifestSHA256: strings.Repeat("b", 64), Stage: ".prepared-fixture", Configs: [3]savedConfig{config, config, config}}
}

func TestJournalBindsCanonicalFixedSnapshot(t *testing.T) {
	journal := journalFixture()
	body, err := json.Marshal(journal)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := parseJournal(body)
	if err != nil || decoded.Version != journal.Version || !bytes.Equal(decoded.Configs[2].Body, journal.Configs[2].Body) {
		t.Fatal("canonical snapshot changed", err)
	}
	for name, body := range map[string][]byte{
		"unknown":    bytes.Replace(body, []byte(`"schema":`), []byte(`"target":"/etc/other","schema":`), 1),
		"duplicate":  bytes.Replace(body, []byte(`"state":"PREPARED"`), []byte(`"state":"PREPARED","state":"COMMITTED"`), 1),
		"case":       bytes.Replace(body, []byte(`"state":`), []byte(`"STATE":`), 1),
		"trailing":   append(append([]byte(nil), body...), []byte(`{}`)...),
		"whitespace": append([]byte(" "), body...),
		"limit":      bytes.Repeat([]byte{' '}, 128<<10+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseJournal(body); err == nil {
				t.Fatal("foreign journal accepted")
			}
		})
	}
	for _, name := range []string{"state", "schema", "path", "version", "downgrade", "same-version-substitution", "old", "hash", "body", "mode-write", "mode-exec", "mode-no-read", "gid"} {
		t.Run(name, func(t *testing.T) {
			value := journalFixture()
			switch name {
			case "state":
				value.State = "ALLOW"
			case "schema":
				value.Schema = "other"
			case "path":
				value.Stage = ".prepared-../../elsewhere"
			case "version":
				value.Version = "latest"
			case "downgrade":
				value.Version = "v0.9.9"
			case "same-version-substitution":
				value.Version = value.Old.Version
			case "old":
				value.Old.ClientUID = 0
			case "hash":
				value.Configs[0].SHA256 = strings.Repeat("0", 64)
			case "body":
				value.Configs[0].Body = nil
			case "mode-write":
				value.Configs[1].Mode = 0666
			case "mode-exec":
				value.Configs[1].Mode = 0755
			case "mode-no-read":
				value.Configs[1].Mode = 0200
			case "gid":
				value.Configs[2].GID = 1 << 31
			}
			body, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := parseJournal(body); err == nil {
				t.Fatal("invalid identity/state/snapshot accepted")
			}
		})
	}
}

func TestJournalTransitionsDoNotFabricateCompletion(t *testing.T) {
	allowed := map[string][]string{
		"PREPARED":          {"SWITCHING", "ROLLING_BACK", "RECOVERY_REQUIRED"},
		"SWITCHING":         {"VERIFYING", "ROLLING_BACK", "RECOVERY_REQUIRED"},
		"VERIFYING":         {"COMMITTED", "ROLLING_BACK", "RECOVERY_REQUIRED"},
		"ROLLING_BACK":      {"ROLLED_BACK", "RECOVERY_REQUIRED"},
		"RECOVERY_REQUIRED": {"ROLLING_BACK", "RECOVERY_REQUIRED"},
	}
	states := []string{"PREPARED", "SWITCHING", "VERIFYING", "COMMITTED", "ROLLING_BACK", "ROLLED_BACK", "RECOVERY_REQUIRED", "UNKNOWN"}
	for _, from := range states {
		for _, to := range states {
			want := false
			for _, permitted := range allowed[from] {
				want = want || to == permitted
			}
			if allowedJournalTransition(from, to) != want {
				t.Fatal("unexpected transition", from, to, want)
			}
		}
	}
}
