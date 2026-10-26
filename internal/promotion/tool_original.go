package promotion

import (
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

// This is a held managed original, never receipt-only native adoption. The
// original code remains on the Host as inert guarded data and is not transported
// or executed. Current new-set inspection and complete publication are separate.
type toolOriginalState struct {
	input    artifactpypi.UVToolUpgradeInput
	metadata pypiInstalledSnapshot
	links    []toolPublishedLink
}

func beginToolOriginalTarget(ctx context.Context, install domain.InstallContext, request artifactpypi.UpgradeRequirement, binRoot string, trustedRoots []string) (*toolFreshTarget, error) {
	return beginToolTarget(ctx, install, request, binRoot, trustedRoots, true)
}

func captureToolOriginal(ctx context.Context, g *toolFreshTarget, receiptPath string) (*toolOriginalState, error) {
	before, err := snapshotBoundedPromotionFile(receiptPath, 4<<20)
	if err != nil || before.Mode.Perm()&0022 != 0 {
		return nil, errors.New("original tool requires a regular private native receipt")
	}
	body, err := readBoundedPromotionFile(receiptPath, 4<<20)
	if err != nil || !before.matches(receiptPath) {
		return nil, errors.New("original tool receipt changed")
	}
	receipt, err := artifactpypi.ParseUVToolReceipt(body)
	if err != nil {
		return nil, err
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
	input, err := artifactpypi.NewUVToolUpgradeInput(receipt, g.requirement, metadata.metadataFiles(), sandbox.PinnedPythonRuntime().PythonVersion)
	if err != nil {
		return nil, err
	}
	state := &toolOriginalState{input: input, metadata: metadata}
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
	for _, command := range receipt.Commands() {
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
	return s.verifyLinks(g)
}
