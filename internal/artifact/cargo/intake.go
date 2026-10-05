package cargo

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	MaxCrateFiles         = 10000
	MaxCrateFileBytes     = 64 << 20
	MaxCrateExpandedBytes = 200 << 20
)

func openPrivateIntake(directory string) (*os.Root, error) {
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("cargo intake directory is not private")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, errors.New("open private cargo intake")
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		_ = root.Close()
		return nil, errors.New("cargo intake directory identity changed")
	}
	return root, nil
}

// ReadIntake resolves only the typed Run handle and rechecks exact observed
// bytes. Artifact-controlled paths are never used as host filenames.
func ReadIntake(directory string, artifact domain.AcquiredArtifact) ([]byte, error) {
	parts := strings.Split(artifact.ContentHandle(), ":")
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || directory == "/" || len(parts) != 3 || parts[0] != "intake" || parts[2] != "crate" || artifact.Identity().Source() != Source() || artifact.Identity().Variant() != "crate" || artifact.SizeBytes() == 0 || artifact.SizeBytes() > MaxCrateBytes {
		return nil, errors.New("cargo intake subject is invalid")
	}
	if _, err := ParseReference(artifact.Identity().Name() + "@" + artifact.Identity().Version()); err != nil {
		return nil, err
	}
	run, err := domain.ParseRunID(parts[1])
	if err != nil {
		return nil, errors.New("cargo intake Run is invalid")
	}
	root, err := openPrivateIntake(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	dirInfo, err := root.Lstat(run.String())
	if err != nil || !dirInfo.IsDir() || dirInfo.Mode().Perm() != 0o700 || dirInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("cargo intake Run directory is invalid")
	}
	runRoot, err := root.OpenRoot(run.String())
	if err != nil {
		return nil, errors.New("open cargo intake Run")
	}
	defer runRoot.Close()
	opened, err := runRoot.Stat(".")
	if err != nil || !os.SameFile(dirInfo, opened) {
		return nil, errors.New("cargo intake Run identity changed")
	}
	before, err := runRoot.Lstat("package.crate")
	if err != nil || !before.Mode().IsRegular() || before.Mode().Perm()&0o077 != 0 || uint64(before.Size()) != artifact.SizeBytes() {
		return nil, errors.New("cargo intake file is invalid")
	}
	file, err := runRoot.OpenFile("package.crate", os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errors.New("open cargo intake file")
	}
	defer file.Close()
	opened, err = file.Stat()
	if err != nil || !os.SameFile(before, opened) || !opened.Mode().IsRegular() {
		return nil, errors.New("cargo intake file identity changed")
	}
	body, err := io.ReadAll(io.LimitReader(file, MaxCrateBytes+1))
	after, statErr := file.Stat()
	current, currentErr := runRoot.Lstat("package.crate")
	if err != nil || statErr != nil || currentErr != nil || !current.Mode().IsRegular() || !os.SameFile(before, current) || !os.SameFile(before, after) || after.Size() != before.Size() || after.ModTime() != before.ModTime() || uint64(len(body)) != artifact.SizeBytes() {
		return nil, errors.New("cargo intake content changed during read")
	}
	digest := sha256.Sum256(body)
	if hex.EncodeToString(digest[:]) != artifact.Digest().String() {
		return nil, errors.New("cargo intake content differs from observed digest")
	}
	return body, nil
}

// ArchiveFiles parses data only. It neither extracts to a host path nor runs
// Rust, build scripts or macros. Callers must retain identity/digest approval.
func ArchiveFiles(body []byte, identity domain.ResolvedArtifactIdentity) (map[string][]byte, error) {
	if len(body) == 0 || len(body) > MaxCrateBytes || identity.Source() != Source() || identity.Variant() != "crate" {
		return nil, errors.New("cargo archive subject is invalid")
	}
	if _, err := ParseReference(identity.Name() + "@" + identity.Version()); err != nil {
		return nil, err
	}
	compressed := bytes.NewReader(body)
	reader, err := gzip.NewReader(compressed)
	if err != nil {
		return nil, errors.New("cargo archive compression is invalid")
	}
	defer reader.Close()
	reader.Multistream(false)
	bounded := &io.LimitedReader{R: reader, N: MaxCrateExpandedBytes + MaxCrateFiles*1024 + 1}
	archive := tar.NewReader(bounded)
	prefix := identity.Name() + "-" + identity.Version() + "/"
	files := map[string][]byte{}
	seen, folded := map[string]bool{}, map[string]string{}
	count, total := 0, int64(0)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		count++
		if err != nil || count > MaxCrateFiles || header.Size < 0 || header.Size > MaxCrateFileBytes || header.Size > MaxCrateExpandedBytes-total || header.Mode&^int64(0o777) != 0 || len(header.PAXRecords) != 0 || header.Linkname != "" {
			return nil, errors.New("cargo archive entry is invalid or excessive")
		}
		name, ok := strings.CutPrefix(header.Name, prefix)
		if header.Typeflag == tar.TypeDir {
			name = strings.TrimSuffix(name, "/")
		}
		if !ok || name == "" || name == "." || len(name) > 4096 || !utf8.ValidString(name) || name != path.Clean(name) || strings.HasPrefix(name, "/") || name == ".." || strings.HasPrefix(name, "../") || strings.ContainsAny(name, "\\\x00\r\n\t") || seen[name] || (header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir) || (header.Typeflag == tar.TypeDir && header.Size != 0) {
			return nil, errors.New("cargo archive path or type is invalid")
		}
		fold := strings.ToLower(name)
		if previous, exists := folded[fold]; exists && previous != name {
			return nil, errors.New("cargo archive contains case collision")
		}
		seen[name], folded[fold] = true, name
		if header.Typeflag == tar.TypeDir {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(archive, header.Size+1))
		if err != nil || int64(len(data)) != header.Size {
			return nil, errors.New("cargo archive file is incomplete")
		}
		files[name] = data
		total += header.Size
	}
	// Consume padding and verify gzip CRC/size; a second member or nonzero tail
	// cannot hide bytes outside the inspected archive.
	tail, err := io.ReadAll(io.LimitReader(bounded, 10241))
	if err != nil || len(tail) > 10240 || bounded.N <= 0 || compressed.Len() != 0 || len(bytes.Trim(tail, "\x00")) != 0 {
		return nil, errors.New("cargo archive has invalid trailer or exceeded expansion bound")
	}
	for name := range seen {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if _, exists := files[parent]; exists {
				return nil, errors.New("cargo archive contains file/ancestor collision")
			}
			if previous, exists := folded[strings.ToLower(parent)]; exists && previous != parent {
				return nil, errors.New("cargo archive contains ancestor case collision")
			}
			folded[strings.ToLower(parent)] = parent
		}
	}
	manifest := files["Cargo.toml"]
	if len(manifest) == 0 || len(manifest) > MaxProjectControlBytes {
		return nil, errors.New("cargo archive lacks bounded normalized manifest")
	}
	var document struct {
		Package struct {
			Name    string `toml:"name"`
			Version string `toml:"version"`
		} `toml:"package"`
	}
	if toml.Unmarshal(manifest, &document) != nil || document.Package.Name != identity.Name() || document.Package.Version != identity.Version() {
		return nil, errors.New("cargo archive manifest identity differs")
	}
	return files, nil
}
