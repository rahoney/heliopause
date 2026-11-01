package npm

import (
	"context"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestNPMProjectRootLifecycleUsesNativeInstallKeys(t *testing.T) {
	phases, err := NPMProjectRootLifecycleScripts([]byte(`{"private":true,"scripts":{"test":"unused","preinstall":"a","install":"b","postinstall":"c","prepublish":"d","preprepare":"e","prepare":"f","postprepare":"g"}}`))
	if err != nil || strings.Join(phases, ",") != "preinstall,install,postinstall,prepublish,preprepare,prepare,postprepare" {
		t.Fatal("root install lifecycle lost a native phase", phases, err)
	}
	if phases, err := NPMProjectRootLifecycleScripts([]byte(`{"scripts":{"postinstall":"\u00a0"}}`)); err != nil || len(phases) != 1 {
		t.Fatal("nonempty Unicode script was mistaken for no action", err)
	}
	if phases, err := NPMProjectRootLifecycleScripts([]byte(`{"scripts":{"dependencies":"node hook.js"}}`)); err != nil || len(phases) != 1 || phases[0] != "dependencies" {
		t.Fatal("conditional dependency-change lifecycle was omitted", err)
	}
	if phases, err := NPMProjectRootLifecycleScripts([]byte(`{"scripts":{"test":"unused","install":""}}`)); err != nil || len(phases) != 0 {
		t.Fatal("non-install/empty script became a lifecycle action", err)
	}
	for _, body := range []string{`{"scripts":{"install":true}}`, `{"scripts":{"install":"a","install":"b"}}`, `{"scripts":[]}`, `[]`} {
		if _, err := NPMProjectRootLifecycleScripts([]byte(body)); err == nil {
			t.Fatal("malformed root scripts accepted")
		}
	}
}

func TestNPMProjectRootArtifactRetainsLocalIdentityAndExactFrozenSubject(t *testing.T) {
	local, _ := domain.NewSourceID("project-local")
	identity, _ := domain.NewResolvedArtifactIdentity(local, "project-fixture", "snapshot", "npm-source")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	transport, _ := domain.NewRunID()
	source, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+transport.String()+":npm-source", 1024, "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := NewNPMProjectRootArtifact(source)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := domain.NewArtifactReference(local, identity.Name())
	resolved, err := artifact.Resolve(context.Background(), reference)
	if err != nil || resolved.Identity() != identity {
		t.Fatal("local source identity changed", err)
	}
	run, _ := domain.NewRunID()
	acquired, err := artifact.Acquire(context.Background(), run, resolved)
	if err != nil || acquired != source {
		t.Fatal("inspection substituted frozen local source", err)
	}
	public, _ := domain.NewSourceID("npm")
	foreign, _ := domain.NewArtifactReference(public, identity.Name())
	if _, err := artifact.Resolve(context.Background(), foreign); err == nil {
		t.Fatal("local byte binding claimed public registry provenance")
	}
	if _, err := artifact.Acquire(context.Background(), domain.RunID{}, resolved); err == nil {
		t.Fatal("root acquisition lacked current inspection run")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := artifact.Resolve(ctx, reference); err == nil {
		t.Fatal("cancelled local request accepted")
	}
	if _, err := NewNPMProjectRootArtifact(domain.AcquiredArtifact{}); err == nil {
		t.Fatal("zero source subject accepted")
	}
}
