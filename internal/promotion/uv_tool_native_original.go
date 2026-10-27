package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

const uvToolOriginalNativeReceiptControl = "uv-tool-original-native-receipt.toml"

// Raw claims and independent source proof remain separate from current approval.
// No original program or interpreter is transported or executed.
type uvToolNativeOriginalProof struct {
	rawReceipt  domain.ProjectControlFile
	ownership   domain.ProjectControlFile
	receipt     pypiFileSnapshot
	interpreter []toolInterpreterMember
	scripts     *uvNativeToolScriptProof
}

func beginToolNativeOriginalTarget(ctx context.Context, install domain.InstallContext, request artifactpypi.UpgradeRequirement, binRoot string, trustedRoots []string, native *pypiNativeEnrollment) (*toolFreshTarget, error) {
	if native == nil || native.client != nativeUVClient || native.resolver == nil || native.intake == nil {
		return nil, errors.New("native UV tool requires existing public original enrollment capabilities")
	}
	bound := false
	for _, root := range trustedRoots {
		bound = bound || root == native.intakeRoot
	}
	if !bound {
		return nil, errors.New("native UV tool original intake is outside held storage boundaries")
	}
	return beginToolTargetWithNative(ctx, install, request, binRoot, trustedRoots, true, native)
}

func (p *uvToolNativeOriginalProof) verify(ctx context.Context, g *toolFreshTarget) error {
	if p == nil || g == nil || p.scripts == nil || p.scripts.tx != g.tx ||
		!p.rawReceipt.Present() || p.rawReceipt.Name() != uvToolOriginalNativeReceiptControl {
		return errors.New("native UV tool original proof is unavailable or changed its target")
	}
	if !p.receipt.matches(filepath.Join(g.tx.plan.root, uvToolHostReceipt)) {
		return errors.New("native UV tool raw receipt changed")
	}
	if err := verifyToolInterpreterSnapshot(p.interpreter); err != nil {
		return err
	}
	return p.scripts.verify(ctx)
}

func captureUVToolNativeOriginal(ctx context.Context, g *toolFreshTarget, receiptPath string, install domain.InstallContext, native *pypiNativeEnrollment) (*toolOriginalState, error) {
	if len(g.tx.current.Files) != 0 || g.tx.metadata != nil || native == nil || native.client != nativeUVClient {
		return nil, errors.New("native UV tool enrollment requires an entirely unmanaged original")
	}
	raw, before, err := readToolOriginalReceipt(receiptPath)
	if err != nil {
		return nil, err
	}
	if err := raw.ValidateLayout(g.tx.plan.root, g.binRoot, raw.InterpreterPath()); err != nil {
		return nil, err
	}
	g.tx.before[receiptPath] = &before
	metadata, err := capturePyPIInstalledSnapshot(g.tx)
	if err != nil {
		return nil, err
	}
	projection, err := raw.WithUVLayout(g.tx.plan.root, g.binRoot)
	if err != nil {
		return nil, err
	}
	// Preserve M1's existing fixed public wheel-only recipe gate. Raw defaults
	// or overrides are not rewritten to masquerade as that supported profile.
	input, err := artifactpypi.NewUVToolUpgradeInput(projection, g.requirement, metadata.metadataFiles(), sandbox.PinnedPythonRuntime().PythonVersion)
	if err != nil {
		return nil, err
	}
	scripts, err := newUVNativeToolScriptProof(ctx, g.tx)
	if err != nil {
		return nil, err
	}
	rawInterpreter, err := snapshotToolInterpreter(raw.InterpreterPath())
	if err != nil {
		return nil, err
	}
	a, b := scripts.interpreter[len(scripts.interpreter)-1], rawInterpreter[len(rawInterpreter)-1]
	if a.file == nil || b.file == nil || !os.SameFile(a.info, b.info) ||
		a.file.Digest != b.file.Digest || a.file.Size != b.file.Size || a.file.Mode != b.file.Mode {
		return nil, errors.New("native UV tool receipt and target do not share the held original interpreter")
	}
	rawControl, err := domain.NewProjectControlFile(uvToolOriginalNativeReceiptControl, raw.Body(), true)
	if err != nil {
		return nil, err
	}
	proof := &uvToolNativeOriginalProof{rawReceipt: rawControl, receipt: before, interpreter: rawInterpreter, scripts: scripts}
	candidate := &toolOriginalState{input: input, metadata: metadata, native: proof}
	for _, command := range raw.Commands() {
		if command.Name == toolBinTransactionLock {
			return nil, errors.New("native original command collides with its guard")
		}
		info, err := g.bin.root.Lstat(command.Name)
		link := toolPublishedLink{Name: command.Name, Path: command.Path, Target: filepath.Join(g.tx.plan.root, "bin", command.Name), info: info}
		if err != nil || !singleToolSymlink(info) || !toolLinkMatches(g.bin.root, command.Name, link) {
			return nil, errors.New("native original exposed command lacks its exact held link")
		}
		candidate.links = append(candidate.links, link)
	}
	// Only constructor-local unapproved state is visible during authentication.
	// The constructor returns no guard until every original has been proved and
	// the same M2 main-owner and command census below succeeds.
	g.original = candidate
	if err := g.verify(ctx); err != nil {
		return nil, err
	}
	lookup := *native
	lookup.toolScripts = scripts
	proof.ownership, err = prepareUVCompleteOwnership(ctx, g.tx, metadata, install, &lookup, true, g.verify)
	if err != nil {
		return nil, err
	}
	if !proof.ownership.Present() {
		return nil, errors.New("native UV tool complete original source proof is absent")
	}
	state, err := captureToolOriginalWithProjection(ctx, g, receiptPath, &projection)
	if err != nil {
		return nil, err
	}
	state.native = proof
	if err := state.verify(ctx, g); err != nil {
		return nil, err
	}
	return state, nil
}
