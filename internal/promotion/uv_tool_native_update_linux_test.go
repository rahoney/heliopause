package promotion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func nativeToolUpdatePreparationFixture(t *testing.T, f nativeToolOriginalFixture, primary error) (*UVToolUpdatePreparation, *uvToolUpgradeFailure) {
	t.Helper()
	selector := &uvToolUpgradeFailure{uvToolSelectionFailure: uvToolSelectionFailure{err: primary}}
	p, err := NewUVToolUpdatePreparation(&PyPIPromotion{stagingRoot: f.staging, runner: unavailableDockerRunner{}, goos: "linux", goarch: "amd64"}, selector, f.bin, []string{f.intake, f.staging})
	if err != nil {
		t.Fatal(err)
	}
	configured, err := p.WithNativeEnrollment(f.ports, f.ports, f.intake)
	if err != nil {
		t.Fatal(err)
	}
	if p.shared.native != nil || configured.shared == p.shared {
		t.Fatal("native enrollment mutated the original preparation")
	}
	return configured, selector
}

func TestUVToolNativeUpdateRequiresExplicitAdoptionAndCurrentSelection(t *testing.T) {
	f := makeNativeToolOriginalFixture(t, "valid")
	primary := errors.New("current native selection stopped")
	p, selector := nativeToolUpdatePreparationFixture(t, f, primary)
	if session, err := p.BeginInstallUpdate(context.Background(), artifactpypi.PublicPyPIProfile().Source(), f.request.Requirement(), f.install, false); err == nil || session != nil {
		if session != nil {
			_ = session.Close()
		}
		t.Fatal("native original adopted without opt-in")
	}
	if len(f.ports.resolves) != 0 || selector.upgrades != 0 {
		t.Fatal("refused original launched selection")
	}
	session, err := p.BeginInstallUpdate(context.Background(), artifactpypi.PublicPyPIProfile().Source(), f.request.Requirement(), f.install, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	})
	held := session.(*uvToolInstallSession)
	if held.guard.original.native == nil || len(held.originalWheels) != 1 {
		t.Fatal("native proof/original WHEEL control missing")
	}
	controls, err := uvToolNativeOriginalControls(context.Background(), held.guard)
	if err != nil || len(controls) != 2 || controls[0].Name() != uvToolOriginalNativeReceiptControl || !bytes.Equal(controls[0].Body(), f.raw) || controls[1].Name() != "pypi-update-native-ownership.json" {
		t.Fatal("native original current controls differ", err)
	}
	lock, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
	target, err := uvToolTargetControl(held.guard, lock)
	var targetData struct{ Intent string }
	if err != nil || json.Unmarshal(target.Body(), &targetData) != nil || targetData.Intent != "native-adoption-upgrade" {
		t.Fatal("native target intent was conflated", err)
	}
	resolution, err := session.ResolveDependencies(context.Background(), session.Reference(), f.install)
	if !errors.Is(err, primary) || len(resolution.Graph().Nodes()) != 0 || selector.upgrades != 1 || selector.calls != 0 {
		t.Fatal("current failure/dispatch changed", err)
	}
	if _, err := session.ResolveDependencies(context.Background(), session.Reference(), f.install); err == nil || selector.upgrades != 1 {
		t.Fatal("native update repeated")
	}
	if _, err := os.Lstat(filepath.Join(f.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("current failure wrote ownership", err)
	}
}

func TestUVToolNativeOriginalCurrentControlsRejectSubstitution(t *testing.T) {
	f := makeNativeToolOriginalFixture(t, "valid")
	g, err := f.begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := g.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, mode := range []string{"missing-source", "replaced-source", "replaced-raw"} {
		t.Run(mode, func(t *testing.T) {
			proof := g.original.native
			raw, source := proof.rawReceipt, proof.ownership
			switch mode {
			case "missing-source":
				proof.ownership = domain.ProjectControlFile{}
			case "replaced-source":
				proof.ownership, _ = domain.NewProjectControlFile(source.Name(), []byte("{}"), true)
			case "replaced-raw":
				proof.rawReceipt, _ = domain.NewProjectControlFile(raw.Name(), append(raw.Body(), '\n'), true)
			}
			_, err := uvToolNativeOriginalControls(context.Background(), g)
			proof.rawReceipt, proof.ownership = raw, source
			if err == nil {
				t.Fatal("unbound original controls accepted")
			}
		})
	}
}

func TestUVToolExplicitAdoptionReusesManagedGuardAfterEnrollment(t *testing.T) {
	install, bin, staging := managedToolOriginalFixture(t)
	intake := filepath.Join(filepath.Dir(install.Target().String()), "native-source-intake")
	if err := os.Mkdir(intake, 0700); err != nil {
		t.Fatal(err)
	}
	ports := &originalProofSetPorts{root: intake, entries: map[string]originalProofArchive{}}
	primary := errors.New("current managed selection stopped")
	selector := &uvToolUpgradeFailure{uvToolSelectionFailure: uvToolSelectionFailure{err: primary}}
	p, err := NewUVToolUpdatePreparation(&PyPIPromotion{stagingRoot: staging, runner: unavailableDockerRunner{}, goos: "linux", goarch: "amd64"}, selector, bin, []string{intake, staging})
	if err != nil {
		t.Fatal(err)
	}
	p, err = p.WithNativeEnrollment(ports, ports, intake)
	if err != nil {
		t.Fatal(err)
	}
	session, err := p.BeginInstallUpdate(context.Background(), artifactpypi.PublicPyPIProfile().Source(), "haa-tool-fixture", install, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	})
	held := session.(*uvToolInstallSession)
	if held.guard.original.native != nil || len(ports.resolves) != 0 {
		t.Fatal("managed target re-enrolled native source")
	}
	controls, err := uvToolNativeOriginalControls(context.Background(), held.guard)
	if err != nil || len(controls) != 0 {
		t.Fatal("managed path invented native controls", err)
	}
	_, err = session.ResolveDependencies(context.Background(), session.Reference(), install)
	if !errors.Is(err, primary) || selector.upgrades != 1 {
		t.Fatal("managed current inspection path changed", err)
	}
}

func TestUVToolNativeAdoptionDoesNotBypassCorruptManagedOwnership(t *testing.T) {
	f := makeNativeToolOriginalFixture(t, "valid")
	if err := os.Mkdir(filepath.Dir(filepath.Join(f.root, pypiVenvMetadata)), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, pypiVenvMetadata), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	p, selector := nativeToolUpdatePreparationFixture(t, f, errors.New("unexpected current selection"))
	session, err := p.BeginInstallUpdate(context.Background(), artifactpypi.PublicPyPIProfile().Source(), f.request.Requirement(), f.install, true)
	if err == nil || session != nil || len(f.ports.resolves) != 0 || selector.upgrades != 0 {
		if session != nil {
			_ = session.Close()
		}
		t.Fatal("corrupt managed state fell back to native enrollment", err)
	}
	if data, err := os.ReadFile(filepath.Join(f.root, pypiVenvMetadata)); err != nil || string(data) != "{}" {
		t.Fatal("corrupt original metadata changed", err)
	}
}

func nativeToolPublicationFixture(t *testing.T, noop bool) (*toolFreshTarget, []pypiDestination, domain.ProjectControlFile, string) {
	t.Helper()
	f := makeNativeToolOriginalFixture(t, "valid")
	f.request, _ = artifactpypi.ParseUpgradeRequirement("haa-tool-fixture")
	g, err := f.begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := g.Close(); err != nil {
			t.Error(err)
		}
	})
	return toolOriginalPublicationMaterial(t, g, noop)
}

func TestNativeUVToolOriginalUsesExistingCompletePublisherWithoutPriorLedger(t *testing.T) {
	for _, noop := range []bool{false, true} {
		t.Run(map[bool]string{false: "changed", true: "same-version"}[noop], func(t *testing.T) {
			g, destinations, receipt, work := nativeToolPublicationFixture(t, noop)
			if g.tx.metadata != nil {
				t.Fatal("native original unexpectedly has a prior ledger")
			}
			if err := g.commitTool(context.Background(), destinations, receipt, work); err != nil {
				t.Fatal(err)
			}
			if !g.original.committed {
				t.Fatal("native source never committed")
			}
			if _, err := os.Stat(filepath.Join(g.tx.plan.root, pypiVenvMetadata)); err != nil {
				t.Fatal("current publication omitted its new ledger", err)
			}
			body, err := os.ReadFile(filepath.Join(g.tx.plan.root, uvToolHostReceipt))
			if err != nil || !bytes.Equal(body, receipt.Body()) {
				t.Fatal("current receipt not published", err)
			}
			if !noop {
				if _, err := os.Lstat(filepath.Join(g.binRoot, "haa-alt")); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("retired native command remains", err)
				}
			}
		})
	}
}

func TestNativeUVToolPublicationFaultRestoresRawReceiptLinksAndLedgerAbsence(t *testing.T) {
	for _, phase := range []string{"after-publish:tool-control", "metadata", "after-tool-link:haa-new", "reconcile"} {
		t.Run(phase, func(t *testing.T) {
			g, destinations, receipt, work := nativeToolPublicationFixture(t, false)
			primary := errors.New("native original publication stopped")
			g.tx.fault = func(at string) error {
				if at == phase {
					return primary
				}
				return nil
			}
			if err := g.commitTool(context.Background(), destinations, receipt, work); !errors.Is(err, primary) {
				t.Fatal("primary native fault lost", err)
			}
			for path, before := range g.tx.before {
				if before != nil && !before.matches(path) {
					t.Fatal("native original identity/bytes/mode not restored", path)
				}
			}
			for _, link := range g.original.links {
				if !toolLinkMatches(g.bin.root, link.Name, link) {
					t.Fatal("native original link not restored", link.Name)
				}
			}
			if _, err := os.Lstat(filepath.Join(g.tx.plan.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("rollback left a new ledger on native original", err)
			}
			// Rollback has already verified the pre-state and removed its private
			// directories. Release uses the persistent-root contract, rather than
			// reusing that completed transaction's mutation inventory.
			if err := g.Close(); err != nil {
				t.Fatal("native original guard did not release", err)
			}
			for _, path := range []string{g.tx.backup, g.tx.tool.private, g.tx.tool.original.path, filepath.Join(g.binRoot, toolBinTransactionLock), filepath.Join(g.tx.plan.root, ".heliopause-pypi-transaction.lock")} {
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("successful native rollback retained owned state", path, err)
				}
			}
		})
	}
}

func TestNativeUVToolPublicationCancellationAndLateCleanupRetainOriginal(t *testing.T) {
	for _, mode := range []string{"cancel", "late-private-cleanup"} {
		t.Run(mode, func(t *testing.T) {
			g, destinations, receipt, work := nativeToolPublicationFixture(t, false)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var foreign string
			g.tx.fault = func(at string) error {
				if mode == "cancel" && at == "after-tool-link:haa-new" {
					cancel()
				}
				if mode == "late-private-cleanup" && at == "reconcile" {
					foreign = filepath.Join(g.tx.tool.private, "foreign-user-data")
					if err := os.WriteFile(foreign, []byte("keep foreign data"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				return nil
			}
			err := g.commitTool(ctx, destinations, receipt, work)
			if err == nil || (mode == "cancel" && !errors.Is(err, context.Canceled)) {
				t.Fatal("native publication lost cancellation/cleanup failure", err)
			}
			for path, before := range g.tx.before {
				if before != nil && !before.matches(path) {
					t.Fatal("native original file not restored", path)
				}
			}
			for _, link := range g.original.links {
				if !toolLinkMatches(g.bin.root, link.Name, link) {
					t.Fatal("native original link not restored", link.Name)
				}
			}
			if _, err := os.Lstat(filepath.Join(g.tx.plan.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed native publication left a ledger", err)
			}
			if mode == "cancel" {
				if err := g.Close(); err != nil {
					t.Fatal(err)
				}
			} else {
				if data, err := os.ReadFile(foreign); err != nil || string(data) != "keep foreign data" || !g.tx.recovery {
					t.Fatal("late cleanup removed foreign data or recovery state", err)
				}
				_ = g.Close()
				for _, path := range []string{g.tx.backup, filepath.Join(g.binRoot, toolBinTransactionLock), filepath.Join(g.tx.plan.root, ".heliopause-pypi-transaction.lock")} {
					if _, err := os.Lstat(path); err != nil {
						t.Fatal("uncertain native cleanup discarded recovery state", path, err)
					}
				}
			}
		})
	}
}
