package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestInspectionEnvironmentIsSyntheticFreshAndRoleSeparated(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "host-secret-must-not-appear")
	t.Setenv("HOME", "/host/private-home-must-not-appear")
	first, _ := domain.NewSandboxSessionID()
	second, _ := domain.NewSandboxSessionID()
	minimal, err := newInspectionEnvironmentProfile(minimalEnvironment, first)
	if err != nil {
		t.Fatal(err)
	}
	mock, err := newInspectionEnvironmentProfile(developmentEnvironment, first)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := newInspectionEnvironmentProfile(developmentEnvironment, second)
	if err != nil || fresh.identity == mock.identity || mock.identity == minimal.identity {
		t.Fatal("independent sessions/profiles share identity", err)
	}
	if len(minimal.fixtures) != 1 || !minimal.fixtures[0].canary {
		t.Fatal("minimal environment has unexpected mock configuration")
	}
	for _, profile := range []inspectionEnvironmentProfile{minimal, mock, fresh} {
		payload, err := profile.payload()
		if err != nil {
			t.Fatal(err)
		}
		joined := strings.Join(profile.environment, "\x00") + string(payload)
		for _, forbidden := range []string{"host-secret-must-not-appear", "/host/private-home-must-not-appear", "LD_PRELOAD=", "HTTP_PROXY=", "HTTPS_PROXY=", "SSL_CERT_FILE="} {
			if strings.Contains(joined, forbidden) {
				t.Fatal("profile imports Host or executable/network authority")
			}
		}
		for _, file := range profile.fixtures {
			if file.canary != (file.name == ".haa-honeytoken") {
				t.Fatal("normal configuration was classified as canary")
			}
		}
		root := environmentTarFixture(t, nil, "")
		state, err := scanEnvironmentState(context.Background(), bytes.NewReader(append(root[:512], payload...)), environmentStateLimits{entries: 64, bytes: 32 << 10, archiveBytes: 64 << 10})
		if err != nil || profile.verifyPlacement(state) != nil {
			t.Fatal("controller payload not verifiable", err)
		}
		if profile.kind == developmentEnvironment {
			if !strings.Contains(joined, "HOME=/tmp/home") || !strings.Contains(joined, "project/.env") || !strings.Contains(joined, "home/.aws/credentials") {
				t.Fatal("mock development cues missing")
			}
		}
		file := state.files[".haa-honeytoken"]
		file.mode = 0600
		state.files[".haa-honeytoken"] = file
		if profile.verifyPlacement(state) == nil {
			t.Fatal("wrong placement mode accepted")
		}
	}
	if _, err := newInspectionEnvironmentProfile("artifact-selected", first); err == nil {
		t.Fatal("unknown profile admitted")
	}
	if _, err := newInspectionEnvironmentProfile(minimalEnvironment, domain.SandboxSessionID{}); err == nil {
		t.Fatal("missing fresh session admitted")
	}
}

func TestInspectionEnvironmentPlacementRejectsMissingChangedOrForeignFiles(t *testing.T) {
	session, _ := domain.NewSandboxSessionID()
	profile, err := newInspectionEnvironmentProfile(developmentEnvironment, session)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := profile.payload()
	if err != nil {
		t.Fatal(err)
	}
	root := environmentTarFixture(t, nil, "")
	state, err := scanEnvironmentState(context.Background(), bytes.NewReader(append(root[:512], payload...)), environmentStateLimits{entries: 64, bytes: 32 << 10, archiveBytes: 64 << 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(environmentFileState) environmentFileState{
		func(f environmentFileState) environmentFileState { f.uid = 0; return f },
		func(f environmentFileState) environmentFileState { f.gid = 0; return f },
		func(f environmentFileState) environmentFileState { f.size++; return f },
		func(f environmentFileState) environmentFileState { f.digest = strings.Repeat("0", 64); return f },
		func(f environmentFileState) environmentFileState { f.kind = tar.TypeSymlink; return f },
	} {
		original := state.files["project/.env"]
		state.files["project/.env"] = mutate(original)
		if profile.verifyPlacement(state) == nil {
			t.Fatal("changed placement accepted")
		}
		state.files["project/.env"] = original
	}
	delete(state.files, "project/.env")
	if profile.verifyPlacement(state) == nil {
		t.Fatal("missing placement accepted")
	}
}
