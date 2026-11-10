package promotion

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type uvToolSelectionFailure struct {
	calls  int
	before func()
	err    error
}

func (r *uvToolSelectionFailure) ResolveToolInstall(context.Context, artifactpypi.UpgradeRequirement) (sandbox.UVToolSelection, error) {
	r.calls++
	if r.before != nil {
		r.before()
	}
	return sandbox.UVToolSelection{}, r.err
}

func TestUVToolPreparationKeepsBothGuardsAcrossNativeFailure(t *testing.T) {
	install, requirement, bin, storage := toolTargetFixture(t)
	primary := errors.New("native tool stopped")
	resolver := &uvToolSelectionFailure{err: primary}
	promoter := &PyPIPromotion{stagingRoot: storage, runner: unavailableDockerRunner{}, goos: "linux", goarch: "amd64"}
	preparation, err := NewUVToolInstallPreparation(promoter, resolver, bin, []string{storage})
	if err != nil {
		t.Fatal(err)
	}
	reference, err := domain.NewArtifactReference(artifactpypi.PublicPyPIProfile().Source(), requirement.Requirement())
	if err != nil {
		t.Fatal(err)
	}
	session, err := preparation.BeginInstall(context.Background(), reference, install)
	if err != nil {
		t.Fatal(err)
	}
	resolver.before = func() {
		for _, path := range []string{filepath.Join(bin, toolBinTransactionLock), filepath.Join(install.Target().String(), ".heliopause-pypi-transaction.lock")} {
			if _, err := os.Lstat(path); err != nil {
				t.Fatal("native selection escaped target guard")
			}
		}
		if other, err := preparation.BeginInstall(context.Background(), reference, install); err == nil {
			_ = other.Close()
			t.Fatal("concurrent request entered native selection")
		}
	}
	result, err := session.ResolveDependencies(context.Background(), reference, install)
	if !errors.Is(err, primary) || len(result.Graph().Nodes()) != 0 || resolver.calls != 1 {
		t.Fatal("native failure lost or produced selection")
	}
	if _, err := session.ResolveDependencies(context.Background(), reference, install); err == nil || resolver.calls != 1 {
		t.Fatal("repeated native attempt accepted")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestUVToolPreparationRejectsInvalidNativeResultAndLateOriginalData(t *testing.T) {
	for _, late := range []bool{false, true} {
		install, requirement, bin, storage := toolTargetFixture(t)
		resolver := &uvToolSelectionFailure{}
		if late {
			resolver.before = func() {
				if err := os.WriteFile(filepath.Join(install.Target().String(), "lib/python3.14/site-packages/foreign.pth"), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
		preparation, err := NewUVToolInstallPreparation(&PyPIPromotion{stagingRoot: storage, runner: unavailableDockerRunner{}, goos: "linux", goarch: "amd64"}, resolver, bin, []string{storage})
		if err != nil {
			t.Fatal(err)
		}
		reference, err := domain.NewArtifactReference(artifactpypi.PublicPyPIProfile().Source(), requirement.Requirement())
		if err != nil {
			t.Fatal(err)
		}
		session, err := preparation.BeginInstall(context.Background(), reference, install)
		if err != nil {
			t.Fatal(err)
		}
		result, err := session.ResolveDependencies(context.Background(), reference, install)
		if err == nil || len(result.Graph().Nodes()) != 0 || resolver.calls != 1 {
			t.Fatal("invalid native selection or changed original accepted")
		}
		if err := session.Close(); err != nil {
			t.Fatal(err)
		}
		if late {
			body, err := os.ReadFile(filepath.Join(install.Target().String(), "lib/python3.14/site-packages/foreign.pth"))
			if err != nil || string(body) != "keep" {
				t.Fatal("late foreign original removed")
			}
		}
	}
}

func TestUVToolTargetControlSerializesRealFingerprintAndRootBindings(t *testing.T) {
	install, requirement, bin, storage := toolTargetFixture(t)
	g, err := beginToolFreshTarget(context.Background(), install, requirement, bin, []string{storage})
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.NewSHA256Digest(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	control, err := uvToolTargetControl(g, digest)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Version                               int
		NativeLock, ConfigSHA256, Interpreter string
		Target, Exposed                       struct {
			Path          string
			Device, Inode uint64
			Mode          uint32
		}
		InterpreterMembers []struct {
			Path, Link, Digest string
			Device, Inode      uint64
			Mode               uint32
			Size               int64
		}
	}
	if err := json.Unmarshal(control.Body(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Version != 1 || wire.NativeLock != digest.String() || wire.ConfigSHA256 != g.tx.config.Digest || wire.Interpreter != filepath.Join(install.Target().String(), "bin/python") || wire.Target.Path != install.Target().String() || wire.Exposed.Path != bin || wire.Target.Inode == 0 || wire.Exposed.Inode == 0 || len(wire.InterpreterMembers) != 2 || len(wire.InterpreterMembers[1].Digest) != 64 || wire.InterpreterMembers[0].Link != "python3.14" {
		t.Fatal("private digest or identity was serialized opaquely")
	}
	again, err := uvToolTargetControl(g, digest)
	if err != nil || again.Digest() != control.Digest() {
		t.Fatal("unchanged target binding is unstable")
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
}
