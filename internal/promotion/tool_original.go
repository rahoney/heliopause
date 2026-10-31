package promotion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// This is a held managed or independently authenticated native original. A raw
// receipt never grants ownership. Original code remains guarded Host DATA and
// is not transported or executed; current approval/publication stay separate.
type toolOriginalState struct {
	input     artifactpypi.UVToolUpgradeInput
	pipx      *artifactpypi.PipxToolUpgradeInput
	metadata  pypiInstalledSnapshot
	links     []toolPublishedLink
	selected  artifactpypi.ToolReceipt
	committed bool
	native    *uvToolNativeOriginalProof
}

func beginToolOriginalTarget(ctx context.Context, install domain.InstallContext, request artifactpypi.UpgradeRequirement, binRoot string, trustedRoots []string) (*toolFreshTarget, error) {
	return beginToolTarget(ctx, install, request, binRoot, trustedRoots, true)
}

func captureToolOriginal(ctx context.Context, g *toolFreshTarget, receiptPath string) (*toolOriginalState, error) {
	return captureToolOriginalWithProjection(ctx, g, receiptPath, nil)
}

func readToolOriginalReceipt(receiptPath string) (artifactpypi.ToolReceipt, pypiFileSnapshot, error) {
	return readToolOriginalReceiptForClient(receiptPath, toolClientUV)
}

func readToolOriginalReceiptForClient(receiptPath string, client toolClient) (artifactpypi.ToolReceipt, pypiFileSnapshot, error) {
	before, err := snapshotBoundedPromotionFile(receiptPath, 4<<20)
	if err != nil || before.Mode.Perm()&0022 != 0 {
		return artifactpypi.ToolReceipt{}, pypiFileSnapshot{}, errors.New("original tool requires a regular private native receipt")
	}
	body, err := readBoundedPromotionFile(receiptPath, 4<<20)
	if err != nil || !before.matches(receiptPath) {
		return artifactpypi.ToolReceipt{}, pypiFileSnapshot{}, errors.New("original tool receipt changed")
	}
	receipt, err := client.parseReceipt(body)
	if err != nil {
		return artifactpypi.ToolReceipt{}, pypiFileSnapshot{}, err
	}
	return receipt, before, nil
}

func captureToolOriginalWithProjection(ctx context.Context, g *toolFreshTarget, receiptPath string, projection *artifactpypi.ToolReceipt) (*toolOriginalState, error) {
	receipt, before, err := readToolOriginalReceiptForClient(receiptPath, g.client)
	if err != nil {
		return nil, err
	}
	if projection != nil {
		if g.client == toolClientPipx {
			if g.original == nil || g.original.native == nil || g.original.native.ownershipSHA == "" {
				return nil, errors.New("pipx original projection requires complete independent source proof")
			}
			if err := g.original.native.verify(ctx, g); err != nil {
				return nil, err
			}
		}
		expected, err := g.client.projectReceipt(receipt, g.tx.plan.root, g.binRoot)
		if err != nil || !bytes.Equal(expected.Body(), projection.Body()) {
			return nil, errors.New("native tool original projection changed its raw receipt semantics")
		}
		receipt = *projection
	}
	if err := receipt.ValidateLayout(g.tx.plan.root, g.binRoot, filepath.Join(g.tx.plan.root, "bin/python")); err != nil {
		return nil, err
	}
	g.tx.before[receiptPath] = &before
	metadata, err := capturePyPIInstalledSnapshot(g.tx)
	if err != nil {
		return nil, err
	}
	managed, err := managedPyPIInventoryFromMetadata(g.tx, metadata.metadataFiles())
	if err != nil || len(managed) == 0 || len(managed) != len(metadata.files) {
		return nil, errors.New("original tool requires complete existing HAA ownership; native adoption is unavailable")
	}
	state := &toolOriginalState{metadata: metadata}
	if g.client == toolClientPipx {
		input, err := artifactpypi.NewPipxToolUpgradeInput(receipt, metadata.metadataFiles(), sandbox.PinnedPythonRuntime().PythonVersion)
		if err != nil {
			return nil, err
		}
		if !input.Request().SameConstraints(g.requirement) {
			return nil, errors.New("pipx upgrade requires the held original bare project")
		}
		state.pipx = &input
	} else {
		input, err := artifactpypi.NewUVToolUpgradeInput(receipt, g.requirement, metadata.metadataFiles(), sandbox.PinnedPythonRuntime().PythonVersion)
		if err != nil {
			return nil, err
		}
		state.input = input
	}
	root := receipt.Requirement().Project()
	var main pypiDestination
	ok := false
	for _, file := range metadata.metadataFiles() {
		name, version, valid := installedDistInfo(file.Directory)
		if valid && name == root && version == managed[root] {
			main, ok = g.tx.current.Files["site/"+file.Directory+"/METADATA"]
		}
	}
	if !ok || len(main.owners()) != 1 || main.Distribution != root || main.Version != managed[root] {
		return nil, errors.New("original tool main metadata is not singly owned")
	}
	commands := map[string]bool{}
	for _, command := range g.client.exposedCommands(receipt, g.binRoot) {
		if command.Name == toolBinTransactionLock {
			return nil, errors.New("original tool command collides with its guard")
		}
		d, ok := g.tx.current.Files["scripts/"+command.Name]
		file := g.tx.before[filepath.Join(g.tx.plan.root, "bin", command.Name)]
		if !ok || len(d.owners()) != 1 || d.Distribution != root || d.Version != main.Version || d.ArtifactDigest != main.ArtifactDigest || file == nil || file.Mode.Perm()&0111 == 0 {
			return nil, errors.New("original exposed command lacks its main distribution ownership")
		}
		info, err := g.bin.root.Lstat(command.Name)
		if err != nil || !singleToolSymlink(info) {
			return nil, errors.New("original exposed command is absent, aliased or not a symlink")
		}
		link := toolPublishedLink{Name: command.Name, Path: command.Path, Target: filepath.Join(g.tx.plan.root, "bin", command.Name), info: info}
		if !toolLinkMatches(g.bin.root, command.Name, link) {
			return nil, errors.New("original command link differs from held layout")
		}
		state.links = append(state.links, link)
		commands[command.Name] = true
	}
	for _, d := range g.tx.current.Files {
		if d.Scheme == "scripts" && d.Distribution == root && !commands[d.Relative] {
			return nil, errors.New("original main command is missing from its receipt")
		}
	}
	if err := state.verify(ctx, g); err != nil {
		return nil, err
	}
	return state, nil
}

func singleToolSymlink(info os.FileInfo) bool {
	if info == nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	value := reflect.ValueOf(info.Sys())
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return false
	}
	nlink := value.Elem().FieldByName("Nlink")
	return nlink.IsValid() && nlink.CanUint() && nlink.Uint() == 1
}

func (s *toolOriginalState) verifyLinks(g *toolFreshTarget) error {
	if s == nil || len(s.links) == 0 || g == nil || g.bin.root == nil {
		return errors.New("original tool command snapshot is unavailable")
	}
	file, err := os.Open(g.binRoot)
	if err != nil {
		return err
	}
	entries, readErr := file.ReadDir(100001)
	closeErr := file.Close()
	if errors.Join(readErr, closeErr) != nil || len(entries) > 100000 {
		return errors.New("original command directory exceeds bounds or is unavailable")
	}
	names := map[string]string{}
	for _, link := range s.links {
		names[strings.ToLower(link.Name)] = link.Name
	}
	for _, entry := range entries {
		if wanted, ok := names[strings.ToLower(entry.Name())]; ok && wanted != entry.Name() {
			return errors.New("original command has a case-colliding destination")
		}
	}
	for _, link := range s.links {
		info, err := g.bin.root.Lstat(link.Name)
		if err != nil || !singleToolSymlink(info) || !toolLinkMatches(g.bin.root, link.Name, link) {
			return errors.New("original command link identity or target changed")
		}
	}
	return nil
}

func (s *toolOriginalState) verify(ctx context.Context, g *toolFreshTarget) error {
	if s == nil {
		return errors.New("original tool state is unavailable")
	}
	if err := s.metadata.verify(ctx, g.tx); err != nil {
		return err
	}
	if s.native != nil {
		if err := s.native.verify(ctx, g); err != nil {
			return err
		}
	}
	return s.verifyLinks(g)
}

func (g *toolFreshTarget) freezeOriginalCommands(ctx context.Context, receipt artifactpypi.ToolReceipt) error {
	if g.original == nil || len(g.commands) != 0 || g.original.selected.Valid() || !receipt.Requirement().SameConstraints(g.original.selectionRequirement(g.requirement)) {
		return errors.New("original tool command selection is repeated or changed its original requirement")
	}
	if err := receipt.ValidateLayout(g.tx.plan.root, g.binRoot, filepath.Join(g.tx.plan.root, "bin/python")); err != nil {
		return err
	}
	if g.client == toolClientUV {
		if _, err := artifactpypi.NewUVToolUpgradeInput(receipt, g.requirement, g.original.metadata.metadataFiles(), sandbox.PinnedPythonRuntime().PythonVersion); err != nil {
			return err
		}
	} else if g.original.pipx == nil || receipt.Client() != "pipx" {
		return errors.New("pipx selected receipt has no held original")
	}
	old := map[string]toolPublishedLink{}
	for _, link := range g.original.links {
		old[link.Name] = link
	}
	directory, err := os.Open(g.binRoot)
	if err != nil {
		return err
	}
	entries, readErr := directory.ReadDir(100001)
	closeErr := directory.Close()
	if errors.Join(readErr, closeErr) != nil || len(entries) > 100000 {
		return errors.New("tool command directory is unavailable or exceeds bounds")
	}
	for _, command := range g.client.exposedCommands(receipt, g.binRoot) {
		if command.Name == toolBinTransactionLock {
			return errors.New("new tool command collides with controller guard")
		}
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), command.Name) {
				if _, ok := old[command.Name]; !ok || entry.Name() != command.Name {
					return errors.New("new tool command collides with an existing command")
				}
			}
		}
		if _, ok := old[command.Name]; !ok {
			if _, err := os.Lstat(command.Path); !errors.Is(err, os.ErrNotExist) {
				return errors.New("new exposed command destination is occupied")
			}
			g.tx.before[command.Path] = nil
		}
		internal := filepath.Join(g.tx.plan.root, "bin", command.Name)
		if _, ok := g.tx.current.Files["scripts/"+command.Name]; !ok {
			if _, err := os.Lstat(internal); !errors.Is(err, os.ErrNotExist) {
				return errors.New("new tool script would replace unowned data")
			}
			g.tx.before[internal] = nil
		}
	}
	g.commands = g.client.exposedCommands(receipt, g.binRoot)
	g.original.selected = receipt
	return g.verify(ctx)
}

// Check the complete frozen new command set before mutation and at final
// reconciliation. Unknown case aliases are preserved and force rollback.
func (g *toolFreshTarget) verifyCommandCaseCollisions() error {
	if len(g.commands) == 0 {
		return nil
	}
	file, err := g.bin.root.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := file.ReadDir(100001)
	closeErr := file.Close()
	if errors.Join(readErr, closeErr) != nil || len(entries) > 100000 {
		return errors.New("tool command namespace is unavailable or exceeds bounds")
	}
	wanted := map[string]string{}
	for _, command := range g.commands {
		wanted[strings.ToLower(command.Name)] = command.Name
	}
	for _, entry := range entries {
		if name, ok := wanted[strings.ToLower(entry.Name())]; ok && name != entry.Name() {
			return errors.New("tool command namespace acquired a foreign case alias")
		}
	}
	return nil
}
