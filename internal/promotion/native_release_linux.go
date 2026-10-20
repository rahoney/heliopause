//go:build linux

package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/releaseactivation"
	"github.com/rahoney/heliopause/internal/releaseinstall"
)

// NativeReleasePromotion connects the existing complete ALLOW bundle/staging
// boundary to the old installed controller's fixed installation client.
type NativeReleasePromotion struct {
	mu          sync.Mutex
	stagingRoot string
	proof       func(domain.AcquiredArtifact) (releaseinstall.ReleaseManifest, [][]byte, error)
	activate    func(context.Context, []byte, [][]byte, func(string) (io.ReadCloser, error)) error
	payload     *nativeStagePayload
	closed      bool
	closeErr    error
}

func NewNativeReleasePromotion(stagingRoot string, lease *releaseactivation.RuntimeLease, proof func(domain.AcquiredArtifact) (releaseinstall.ReleaseManifest, [][]byte, error), drain func() error) (*NativeReleasePromotion, error) {
	if !filepath.IsAbs(stagingRoot) || filepath.Clean(stagingRoot) != stagingRoot || lease == nil || proof == nil || drain == nil {
		return nil, releaseactivation.ErrUnmanaged
	}
	return &NativeReleasePromotion{stagingRoot: stagingRoot, proof: proof, activate: func(ctx context.Context, body []byte, proofs [][]byte, open func(string) (io.ReadCloser, error)) error {
		return releaseactivation.Activate(ctx, lease, body, proofs, open, drain)
	}}, nil
}

func (p *NativeReleasePromotion) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (result domain.PromotedInstall, resultErr error) {
	if p == nil {
		return result, releaseactivation.ErrUntrusted
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.payload != nil {
		return result, releaseactivation.ErrTransition
	}
	if ctx == nil || p == nil || p.proof == nil || p.activate == nil || !install.Valid() || install.Mode() != domain.InstallGitHubAsset || install.Target().String() != releaseinstall.SystemInstallRoot || !bundle.Valid() || staged.ManifestID() != bundle.ManifestID() || staged.Handle() != "staging:"+bundle.ManifestID().String() || verifyDocuments(bundle) != nil {
		return result, releaseactivation.ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	var contextRecord struct {
		Target struct {
			Path        string `json:"path"`
			RequiresNew bool   `json:"requires_new"`
		} `json:"target"`
	}
	if json.Unmarshal(bundle.ManifestDocument(), &contextRecord) != nil || contextRecord.Target.Path != install.Target().String() || contextRecord.Target.RequiresNew {
		return result, releaseactivation.ErrUntrusted
	}
	inspections := bundle.Set().Inspected().Inspections()
	if len(inspections) != 7 {
		return result, releaseactivation.ErrUntrusted
	}
	byName := map[string]domain.AcquiredArtifact{}
	for _, inspection := range inspections {
		artifact := inspection.Artifact()
		id := artifact.Identity()
		if id.Source().String() != "github-release" || id.Name() != "rahoney-heliopause" || inspection.PolicyDecision().Decision() != domain.DecisionAllow || id.Variant() == "" || byName[id.Variant()].Identity().Name() != "" {
			return result, releaseactivation.ErrUntrusted
		}
		byName[id.Variant()] = artifact
	}
	primary, ok := byName["helox-release-manifest.json"]
	if !ok {
		return result, releaseactivation.ErrUntrusted
	}
	manifest, proofs, err := p.proof(primary)
	if err != nil {
		return result, err
	}
	native, err := manifest.NativeUpdateFiles()
	if err != nil || manifest.SHA256() != primary.Digest().String() {
		return result, releaseactivation.ErrUntrusted
	}
	selected := append([]releaseinstall.ReleaseFile(nil), native...)
	for _, file := range manifest.Files() {
		if file.Role == "release-metadata" {
			selected = append(selected, file)
		}
	}
	if len(selected) != 6 {
		return result, releaseactivation.ErrUntrusted
	}
	for _, file := range selected {
		artifact, ok := byName[file.Name]
		if !ok || artifact.Identity().Version() != manifest.Version() || artifact.Digest().String() != file.SHA256 || artifact.SizeBytes() != uint64(file.Size) {
			return result, releaseactivation.ErrUntrusted
		}
	}
	if primary.Identity().Version() != manifest.Version() || primary.SizeBytes() > 1<<20 {
		return result, releaseactivation.ErrUntrusted
	}
	rootPath := filepath.Join(p.stagingRoot, bundle.ManifestID().String())
	if filepath.Dir(rootPath) != p.stagingRoot || rejectSymlinkPath(rootPath) != nil {
		return result, releaseactivation.ErrUntrusted
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return result, err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	if err := nativeStageDirectory(root, "."); err != nil {
		return result, err
	}
	for name, expected := range map[string][]byte{manifestFilename: bundle.ManifestDocument(), sbomFilename: bundle.SBOMDocument()} {
		file, err := openNativeStageFile(root, name, int64(len(expected)))
		if err != nil {
			return result, err
		}
		actual, readErr := io.ReadAll(io.LimitReader(file, int64(len(expected))+1))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil || !bytes.Equal(actual, expected) {
			return result, errors.Join(releaseactivation.ErrUntrusted, readErr, closeErr)
		}
	}
	if err := nativeStageDirectory(root, "artifacts"); err != nil {
		return result, err
	}
	artifacts, err := root.OpenRoot("artifacts")
	if err != nil {
		return result, err
	}
	defer func() { resultErr = errors.Join(resultErr, artifacts.Close()) }()
	held := map[string]*os.File{}
	identities := map[string]os.FileInfo{}
	defer func() {
		for _, file := range held {
			resultErr = errors.Join(resultErr, file.Close())
		}
	}()
	for name, artifact := range byName {
		_, stagedName, err := stagedArtifactNames(artifact)
		if err != nil {
			return result, err
		}
		if artifact.SizeBytes() == 0 || artifact.SizeBytes() > 512<<20 {
			return result, releaseactivation.ErrUntrusted
		}
		file, err := openNativeStageFile(artifacts, stagedName, int64(artifact.SizeBytes()))
		if err != nil {
			return result, err
		}
		held[name] = file
		identity, err := file.Stat()
		if err != nil {
			return result, err
		}
		identities[stagedName] = identity
		hash := sha256.New()
		count, err := io.Copy(hash, &nativeContextReader{ctx, io.LimitReader(file, int64(artifact.SizeBytes())+1)})
		if err != nil || count != int64(artifact.SizeBytes()) || hex.EncodeToString(hash.Sum(nil)) != artifact.Digest().String() {
			return result, errors.Join(releaseactivation.ErrUntrusted, err)
		}
	}
	body, err := io.ReadAll(io.NewSectionReader(held["helox-release-manifest.json"], 0, int64(primary.SizeBytes())))
	if err != nil {
		return result, err
	}
	checked, err := releaseinstall.ParseReleaseManifest(body)
	if err != nil || checked.SHA256() != manifest.SHA256() {
		return result, errors.Join(releaseactivation.ErrUntrusted, err)
	}
	rootInfo, rootErr := root.Stat(".")
	artifactsInfo, artifactsErr := artifacts.Stat(".")
	currentArtifacts, currentErr := root.Lstat("artifacts")
	if rootErr != nil || artifactsErr != nil || currentErr != nil || !os.SameFile(artifactsInfo, currentArtifacts) {
		return result, errors.Join(releaseactivation.ErrCleanup, rootErr, artifactsErr, currentErr)
	}
	p.payload = &nativeStagePayload{rootPath, rootInfo, artifactsInfo, identities}
	ports.ReportProgress(ctx, ports.Progress{Stage: "authorize"})
	if err := p.activate(ctx, body, proofs, func(name string) (io.ReadCloser, error) {
		file, ok := held[name]
		if !ok || name == "helox-release-manifest.json" {
			return nil, releaseactivation.ErrUntrusted
		}
		return io.NopCloser(io.NewSectionReader(file, 0, int64(byName[name].SizeBytes()))), nil
	}); err != nil {
		return result, err
	}
	return domain.NewPromotedInstall(bundle.ManifestID(), install.Target())
}

func nativeStageDirectory(root *os.Root, name string) error {
	info, err := root.Lstat(name)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0700 {
		return releaseactivation.ErrUntrusted
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != uint32(os.Geteuid()) {
		return releaseactivation.ErrUntrusted
	}
	return nil
}

func openNativeStageFile(root *os.Root, name string, size int64) (*os.File, error) {
	before, err := root.Lstat(name)
	if err != nil || !before.Mode().IsRegular() || before.Mode().Perm() != 0400 || before.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 || before.Size() != size {
		return nil, releaseactivation.ErrUntrusted
	}
	owner, ok := before.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != uint32(os.Geteuid()) || owner.Nlink != 1 {
		return nil, releaseactivation.ErrUntrusted
	}
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) {
		return nil, errors.Join(releaseactivation.ErrUntrusted, err, file.Close())
	}
	return file, nil
}

type nativeContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *nativeContextReader) Read(body []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(body)
}
