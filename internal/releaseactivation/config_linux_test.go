//go:build linux

package releaseactivation

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/rahoney/heliopause/internal/hosttool"
)

func runtimeConfigFixture(t *testing.T) updateJournal {
	t.Helper()
	journal := journalFixture()
	policy := nativePolicyConfig{policySocketPath, nativeVersionPath(journal.Old.Version, "helox-linux-amd64"), journal.Old.ClientUID, journal.Old.ClientGID}
	host := hosttool.Config{DockerPath: "/usr/local/bin/docker", DockerEndpoint: "unix:///run/local-docker.sock", ObserverHelperPath: nativeVersionPath(journal.Old.Version, "haa_gvisor_observer-linux-amd64")}
	var bodies [3][]byte
	var err error
	bodies[0], err = json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	bodies[1], err = json.Marshal(host)
	if err != nil {
		t.Fatal(err)
	}
	bodies[2], err = nativePolicyUnit(journal.Old.Version)
	if err != nil {
		t.Fatal(err)
	}
	for index, body := range bodies {
		journal.Configs[index] = savedConfig{body, bodySHA256(body), 0644, 0}
	}
	return journal
}

func TestRuntimeConfigDerivationKeepsHostIdentity(t *testing.T) {
	journal := runtimeConfigFixture(t)
	policy, host, err := parseNativeConfigs(journal)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := derivedRuntimeConfigs(journal)
	if err != nil {
		t.Fatal(err)
	}
	var nextPolicy nativePolicyConfig
	var nextHost hosttool.Config
	if json.Unmarshal(derived[0], &nextPolicy) != nil || json.Unmarshal(derived[1], &nextHost) != nil {
		t.Fatal("derived config invalid")
	}
	if nextPolicy.SocketPath != policy.SocketPath || nextPolicy.ClientUID != policy.ClientUID || nextPolicy.ClientGID != policy.ClientGID || nextPolicy.ClientPath != nativeVersionPath(journal.Version, "helox-linux-amd64") {
		t.Fatal("policy identity or version path changed incorrectly")
	}
	if nextHost.DockerPath != host.DockerPath || nextHost.DockerEndpoint != host.DockerEndpoint || nextHost.ObserverHelperPath != nativeVersionPath(journal.Version, "haa_gvisor_observer-linux-amd64") {
		t.Fatal("runtime tool identity changed incorrectly")
	}
	want, err := nativePolicyUnit(journal.Version)
	if err != nil || !bytes.Equal(derived[2], want) {
		t.Fatal("new unit does not select exact helper")
	}
	for _, name := range []string{"socket", "uid", "client", "observer", "docker", "endpoint", "unknown", "duplicate", "unit"} {
		t.Run(name, func(t *testing.T) {
			value := runtimeConfigFixture(t)
			switch name {
			case "socket":
				value.Configs[0].Body = bytes.Replace(value.Configs[0].Body, []byte(policySocketPath), []byte("/run/other/service.sock"), 1)
			case "uid":
				value.Configs[0].Body = bytes.Replace(value.Configs[0].Body, []byte(`"client_uid":1000`), []byte(`"client_uid":0`), 1)
			case "client":
				value.Configs[0].Body = bytes.Replace(value.Configs[0].Body, []byte("helox-linux-amd64"), []byte("foreign"), 1)
			case "observer":
				value.Configs[1].Body = bytes.Replace(value.Configs[1].Body, []byte("haa_gvisor_observer"), []byte("foreign"), 1)
			case "docker":
				value.Configs[1].Body = bytes.Replace(value.Configs[1].Body, []byte("/usr/local/bin/docker"), []byte("/tmp/docker"), 1)
			case "endpoint":
				value.Configs[1].Body = bytes.Replace(value.Configs[1].Body, []byte("unix:///run/local-docker.sock"), []byte("tcp://remote:2375"), 1)
			case "unknown":
				value.Configs[0].Body = bytes.Replace(value.Configs[0].Body, []byte(`"socket_path":`), []byte(`"argv":"other","socket_path":`), 1)
			case "duplicate":
				value.Configs[0].Body = bytes.Replace(value.Configs[0].Body, []byte(`"client_uid":1000`), []byte(`"client_uid":1000,"client_uid":1000`), 1)
			case "unit":
				value.Configs[2].Body = append(value.Configs[2].Body, []byte("\n[Service]\nExecStart=/tmp/foreign\n")...)
			}
			for index := range value.Configs {
				value.Configs[index].SHA256 = bodySHA256(value.Configs[index].Body)
			}
			if _, err := derivedRuntimeConfigs(value); err == nil {
				t.Fatal("foreign managed config was adopted")
			}
		})
	}
}
