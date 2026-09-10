// Package promotion implements trusted storage boundaries used after Policy ALLOW.
package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	manifestFilename = "manifest.json"
	sbomFilename     = "sbom.cdx.json"
)

// LocalStaging atomically persists verified artifacts and their records under
// a trusted root that is separate from controlled intake and Evidence.
type LocalStaging struct {
	intakeRoot   string
	evidenceRoot string
	stagingRoot  string
}

// NewLocalStaging requires three absolute, non-overlapping storage roots.
func NewLocalStaging(intakeRoot, evidenceRoot, stagingRoot string) (*LocalStaging, error) {
	roots := []string{filepath.Clean(intakeRoot), filepath.Clean(evidenceRoot), filepath.Clean(stagingRoot)}
	for _, root := range roots {
		if !filepath.IsAbs(root) {
			return nil, errors.New("intake, Evidence, and staging roots must be absolute")
		}
	}
	if !separateRoots(roots) {
		return nil, errors.New("intake, Evidence, and staging roots must not overlap")
	}
	return &LocalStaging{intakeRoot: roots[0], evidenceRoot: roots[1], stagingRoot: roots[2]}, nil
}

// Stage re-hashes every intake source, writes a private temporary tree, syncs
// it, and atomically publishes it only after all required records are present.
func (s *LocalStaging) Stage(ctx context.Context, bundle domain.VerifiedBundle) (staged domain.StagedSet, resultErr error) {
	if ctx == nil {
		return domain.StagedSet{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return domain.StagedSet{}, err
	}
	if s == nil || !bundle.Valid() {
		return domain.StagedSet{}, errors.New("staging requires a configured adapter and valid verified bundle")
	}
	if err := verifyDocuments(bundle); err != nil {
		return domain.StagedSet{}, err
	}
	if err := ensureTrustedRoot(s.stagingRoot); err != nil {
		return domain.StagedSet{}, err
	}
	if err := artifactpypi.CheckTemporaryDisk(s.stagingRoot, artifactpypi.ResourcePolicyFromContext(ctx)); err != nil {
		return domain.StagedSet{}, err
	}
	final := filepath.Join(s.stagingRoot, bundle.ManifestID().String())
	if filepath.Dir(final) != s.stagingRoot {
		return domain.StagedSet{}, errors.New("staging destination escapes root")
	}
	if _, err := os.Lstat(final); !errors.Is(err, os.ErrNotExist) {
		return domain.StagedSet{}, errors.New("staging destination already exists or cannot be verified")
	}
	temporary, err := os.MkdirTemp(s.stagingRoot, ".stage-")
	if err != nil {
		return domain.StagedSet{}, fmt.Errorf("create staging temporary directory: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			if err := os.RemoveAll(temporary); err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("remove incomplete staging: %w", err))
			}
		}
	}()
	if err := os.Chmod(temporary, 0o700); err != nil {
		return domain.StagedSet{}, fmt.Errorf("protect staging temporary directory: %w", err)
	}
	artifacts := filepath.Join(temporary, "artifacts")
	if err := os.Mkdir(artifacts, 0o700); err != nil {
		return domain.StagedSet{}, fmt.Errorf("create staged artifact directory: %w", err)
	}
	written := make(map[string]bool)
	for _, inspection := range bundle.Set().Inspected().Inspections() {
		if err := ctx.Err(); err != nil {
			return domain.StagedSet{}, err
		}
		if err := s.stageArtifact(artifacts, inspection, written); err != nil {
			return domain.StagedSet{}, err
		}
	}
	if err := writeRecord(temporary, manifestFilename, bundle.ManifestDocument()); err != nil {
		return domain.StagedSet{}, err
	}
	if err := writeRecord(temporary, sbomFilename, bundle.SBOMDocument()); err != nil {
		return domain.StagedSet{}, err
	}
	if err := sealFiles(temporary); err != nil {
		return domain.StagedSet{}, err
	}
	if err := syncDirectory(artifacts); err != nil {
		return domain.StagedSet{}, err
	}
	if err := syncDirectory(temporary); err != nil {
		return domain.StagedSet{}, err
	}
	if err := renameNoReplace(temporary, final); err != nil {
		return domain.StagedSet{}, fmt.Errorf("atomically finalize staged set: %w", err)
	}
	cleanup = false
	if err := syncDirectory(s.stagingRoot); err != nil {
		cleanupErr := os.RemoveAll(final)
		if cleanupErr != nil {
			return domain.StagedSet{}, errors.Join(err, fmt.Errorf("remove uncertain staged set: %w", cleanupErr))
		}
		return domain.StagedSet{}, err
	}
	return domain.NewStagedSet(bundle.ManifestID(), "staging:"+bundle.ManifestID().String())
}

func (s *LocalStaging) stageArtifact(directory string, inspection domain.DependencyInspection, written map[string]bool) error {
	artifact := inspection.Artifact()
	if artifact.SizeBytes() > math.MaxInt64-1 {
		return errors.New("staged artifact size exceeds supported bound")
	}
	runID, err := intakeRunID(artifact.ContentHandle())
	if err != nil || runID != inspection.RunID().String() {
		return errors.New("intake handle does not match inspection Run")
	}
	sourceName, stagedName, err := stagedArtifactNames(artifact)
	if err != nil {
		return err
	}
	source := filepath.Join(s.intakeRoot, runID, sourceName)
	if err := rejectSymlinkPath(source); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open controlled intake artifact: %w", err)
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil || !info.Mode().IsRegular() || uint64(info.Size()) != artifact.SizeBytes() {
		return errors.New("controlled intake artifact is not the expected regular file")
	}
	digest := sha256.New()
	var output *os.File
	if !written[artifact.Digest().String()] {
		output, err = os.OpenFile(filepath.Join(directory, stagedName), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return fmt.Errorf("create staged artifact: %w", err)
		}
	}
	writer := io.Writer(digest)
	if output != nil {
		writer = io.MultiWriter(digest, output)
	}
	copied, copyErr := io.Copy(writer, io.LimitReader(input, int64(artifact.SizeBytes())+1))
	if output != nil {
		if copyErr == nil {
			copyErr = output.Sync()
		}
		if closeErr := output.Close(); copyErr == nil {
			copyErr = closeErr
		}
	}
	if copyErr != nil || copied != int64(artifact.SizeBytes()) || hex.EncodeToString(digest.Sum(nil)) != artifact.Digest().String() {
		return errors.New("controlled intake artifact changed before staging")
	}
	written[artifact.Digest().String()] = true
	return nil
}

func verifyDocuments(bundle domain.VerifiedBundle) error {
	manifest := bundle.ManifestDocument()
	sbom := bundle.SBOMDocument()
	var manifestObject map[string]any
	decoder := json.NewDecoder(bytes.NewReader(manifest))
	decoder.UseNumber()
	if !json.Valid(manifest) || decoder.Decode(&manifestObject) != nil {
		return errors.New("verified Manifest document is invalid")
	}
	id, ok := manifestObject["manifest_id"].(string)
	if !ok || id != bundle.ManifestID().String() {
		return errors.New("verified Manifest identity does not match bundle")
	}
	delete(manifestObject, "manifest_id")
	canonical, err := json.Marshal(manifestObject)
	if err != nil {
		return errors.New("canonicalize verified Manifest")
	}
	sum := sha256.Sum256(canonical)
	if hex.EncodeToString(sum[:]) != id {
		return errors.New("verified Manifest content does not match identity")
	}
	var bom struct {
		BOMFormat   string `json:"bomFormat"`
		SpecVersion string `json:"specVersion"`
		Metadata    struct {
			Properties []struct{ Name, Value string } `json:"properties"`
		} `json:"metadata"`
	}
	if !json.Valid(sbom) || json.Unmarshal(sbom, &bom) != nil || bom.BOMFormat != "CycloneDX" || bom.SpecVersion != "1.7" {
		return errors.New("verified SBOM is not CycloneDX 1.7")
	}
	for _, property := range bom.Metadata.Properties {
		if property.Name == "heliopause:manifest-id" && property.Value == id {
			return nil
		}
	}
	return errors.New("verified SBOM does not bind the Manifest identity")
}

func intakeRunID(handle string) (string, error) {
	parts := strings.Split(handle, ":")
	if len(parts) != 3 || parts[0] != "intake" || (!validIntakeVariant(parts[2]) && parts[2] != "github-release") {
		return "", errors.New("invalid controlled intake handle")
	}
	runID, err := domain.ParseRunID(parts[1])
	if err != nil {
		return "", errors.New("invalid controlled intake handle")
	}
	return runID.String(), nil
}

func stagedArtifactNames(artifact domain.AcquiredArtifact) (string, string, error) {
	variant := artifact.Identity().Variant()
	if artifact.Identity().Source().String() == "github-release" && variant != "" {
		return "asset", artifact.Digest().String() + ".asset", nil
	}
	if !validIntakeVariant(variant) {
		return "", "", errors.New("unsupported controlled intake artifact variant")
	}
	if artifact.Identity().Source().String() == "npm" && variant == "tarball" {
		return "tarball.tgz", artifact.Digest().String() + ".tgz", nil
	}
	if _, supported := artifactpypi.ProfileForSource(artifact.Identity().Source()); supported {
		switch variant {
		case "wheel":
			return "wheel.whl", artifact.Digest().String() + ".whl", nil
		case "derived-wheel":
			return "derived.whl", artifact.Digest().String() + ".whl", nil
		case "sdist":
			return "sdist.tar.gz", artifact.Digest().String() + ".tar.gz", nil
		}
	}
	if artifact.Identity().Source().String() == "github-release" && variant != "" {
		return "asset", artifact.Digest().String() + ".asset", nil
	}
	return "", "", errors.New("controlled intake source and variant are incompatible")
}

func validIntakeVariant(value string) bool {
	return value == "tarball" || value == "wheel" || value == "derived-wheel" || value == "sdist"
}

func separateRoots(roots []string) bool {
	for i := range roots {
		for j := range roots {
			if i == j {
				continue
			}
			relative, err := filepath.Rel(roots[i], roots[j])
			if err != nil || relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
				return false
			}
		}
	}
	return true
}

func ensureTrustedRoot(root string) error {
	if err := rejectSymlinkPathAllowMissing(root); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("create staging root: %w", err)
	}
	if err := rejectSymlinkPath(root); err != nil {
		return err
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() {
		return errors.New("staging root is not a trusted directory")
	}
	return os.Chmod(root, 0o700)
}

func rejectSymlinkPath(path string) error             { return walkPath(path, false) }
func rejectSymlinkPathAllowMissing(path string) error { return walkPath(path, true) }

func walkPath(path string, allowMissing bool) error {
	volume := filepath.VolumeName(path)
	root := volume + string(filepath.Separator)
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("trusted storage path is invalid")
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) && allowMissing {
			return nil
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("trusted storage path contains a symbolic link or unavailable component")
		}
	}
	return nil
}

func writeRecord(directory, name string, document []byte) error {
	file, err := os.OpenFile(filepath.Join(directory, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create staged record: %w", err)
	}
	if _, err = file.Write(document); err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("persist staged record: %w", err)
	}
	return nil
}

func sealFiles(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("staging temporary tree contains a symbolic link")
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("staging temporary tree contains a non-regular file")
		}
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open completed staged file: %w", err)
		}
		if err = os.Chmod(path, 0o400); err == nil {
			err = file.Sync()
		}
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return fmt.Errorf("seal completed staged file: %w", err)
		}
		return nil
	})
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open staging directory for sync: %w", err)
	}
	err = directory.Sync()
	closeErr := directory.Close()
	if err != nil {
		return fmt.Errorf("sync staging directory: %w", err)
	}
	return closeErr
}
