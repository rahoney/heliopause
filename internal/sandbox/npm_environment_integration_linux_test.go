//go:build linux

package sandbox

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

func TestLinuxGVisorLifecycleIntegration(t *testing.T) {
	if os.Getenv("HELOX_GVISOR_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor runtime")
	}
	runLinuxGVisorLifecycleIntegration(t, `{"name":"tiny","version":"1.2.3"}`)
}

func TestLinuxGVisorArtifactEnvironmentIsolationIntegration(t *testing.T) {
	if os.Getenv("HELOX_GVISOR_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor runtime")
	}
	for key, value := range map[string]string{
		"GITHUB_TOKEN":          "fake-github-token",
		"NPM_TOKEN":             "fake-npm-token",
		"PYPI_TOKEN":            "fake-pypi-token",
		"AWS_SECRET_ACCESS_KEY": "fake-aws-secret",
		"OPENAI_API_KEY":        "fake-openai-key",
		"CI_SECRET":             "fake-ci-secret",
		"ARBITRARY_TOKEN":       "fake-arbitrary-token",
		"SSH_AUTH_SOCK":         "/tmp/fake-ssh-agent.sock",
	} {
		t.Setenv(key, value)
	}
	runLinuxGVisorLifecycleIntegration(t, `{"name":"tiny","version":"1.2.3","scripts":{"preinstall":"test \"$GITHUB_TOKEN\" != fake-github-token && test \"$NPM_TOKEN\" != fake-npm-token && test -z \"$PYPI_TOKEN\" && test \"$AWS_SECRET_ACCESS_KEY\" != fake-aws-secret && test -z \"$OPENAI_API_KEY\" && test -z \"$CI_SECRET\" && test -z \"$ARBITRARY_TOKEN\" && test -z \"$SSH_AUTH_SOCK\""}}`)
}

func runLinuxGVisorLifecycleIntegration(t *testing.T, body string) domain.SandboxResult {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "tarball.tgz")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	writer := tar.NewWriter(gzipWriter)
	if err := writer.WriteHeader(&tar.Header{Name: "package/package.json", Size: int64(len(body)), Mode: 0o600}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	runner := integrationRunner{t: t}
	introducer, err := NewDockerArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewBackend(runner, introducer, supervisor.Observer(), integrationCapabilityProbe(runner))
	if err != nil {
		t.Fatal(err)
	}
	resources, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	backend.resources = integrationObservationResources{client: resources}
	result, err := backend.Execute(context.Background(), integrationRequest(t, path))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status() != domain.SandboxCompleted {
		code, _ := result.LimitationCode()
		t.Fatalf("Sandbox result = %q/%q observer_reason=%s", result.Status(), code, integrationObserverFaultReason(supervisor))
	}
	return result
}

func integrationRequest(t *testing.T, path string) domain.SandboxRequest {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:tarball", uint64(len(payload)))
	if err != nil {
		t.Fatal(err)
	}
	request, err := domain.NewSandboxRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func TestLinuxGVisorFreshEnvironmentIntegration(t *testing.T) {
	if os.Getenv("HELOX_GVISOR_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and protected resource service")
	}
	result := runLinuxGVisorLifecycleIntegration(t, `{"name":"tiny","version":"1.2.3","scripts":{"preinstall":"test ! -e /tmp/haa-environment-dirty; case \"$GITHUB_TOKEN\" in \"\") test ! -e /tmp/.aws/credentials;; haa-invalid-*) test -r /tmp/.aws/credentials;; *) exit 51;; esac; : > /tmp/haa-environment-dirty"}}`)
	counts := map[string]uint64{}
	for _, fact := range result.Observations() {
		counts[fact.Subject()] += fact.Count()
	}
	for _, kind := range []inspectionEnvironment{minimalEnvironment, developmentEnvironment} {
		if counts["environment-"+string(kind)+"-completed"] != 1 {
			t.Fatal("required fresh environment did not complete")
		}
	}
	t.Log("fresh environments=2; inherited Host secrets=absent; synthetic cues=isolated; previous-run marker=absent; completion authority=external")
}
