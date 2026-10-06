package terraformprovider

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const registryHeader = "HAA-TERRAFORM-REGISTRY-1\n"
const bundleHeader = "HAA-TERRAFORM-PROVIDER-1\n"

var registryLimits = []int{64 << 10, 2 << 20, 1 << 20}
var bundleLimits = []int{64 << 10, 2 << 20, 1 << 20, 1 << 20, 16 << 10, MaxProviderArchiveBytes}

// Bundle contains one immutable acquired subject. Envelope, archive SHA-256,
// extracted h1 and executable SHA-256 remain distinct identities.
type Bundle struct{ parts [][]byte }

func (b Bundle) Registry() RegistrySnapshot {
	return RegistrySnapshot{bytes.Clone(b.parts[0]), bytes.Clone(b.parts[1]), bytes.Clone(b.parts[2])}
}
func (b Bundle) Checksums() []byte     { return bytes.Clone(b.parts[3]) }
func (b Bundle) Signature() []byte     { return bytes.Clone(b.parts[4]) }
func (b Bundle) Archive() []byte       { return bytes.Clone(b.parts[5]) }
func (b Bundle) ArchiveDigest() string { return sha256Hex(b.parts[5]) }
func (b Bundle) RegistryDigest() string {
	body, _ := encodeParts(registryHeader, b.parts[:3], registryLimits)
	return sha256Hex(body)
}
func (b Bundle) ZipReader() (*zip.Reader, error) {
	return zip.NewReader(bytes.NewReader(b.parts[5]), int64(len(b.parts[5])))
}

func registryParts(s RegistrySnapshot) [][]byte { return [][]byte{s.Discovery, s.Versions, s.Package} }

func encodeParts(header string, parts [][]byte, limits []int) ([]byte, error) {
	if len(parts) != len(limits) {
		return nil, errors.New("terraform envelope field count is invalid")
	}
	size := len(header) + 8*len(parts)
	for index, part := range parts {
		if len(part) == 0 || len(part) > limits[index] {
			return nil, errors.New("terraform envelope field exceeds bounds")
		}
		size += len(part)
	}
	body := make([]byte, size)
	copy(body, header)
	offset := len(header)
	for _, part := range parts {
		binary.BigEndian.PutUint64(body[offset:], uint64(len(part)))
		offset += 8
		copy(body[offset:], part)
		offset += len(part)
	}
	return body, nil
}

func decodeParts(body []byte, header string, limits []int) ([][]byte, error) {
	if !bytes.HasPrefix(body, []byte(header)) {
		return nil, errors.New("terraform envelope header is invalid")
	}
	offset := len(header)
	parts := make([][]byte, 0, len(limits))
	for _, limit := range limits {
		if len(body)-offset < 8 {
			return nil, errors.New("terraform envelope is truncated")
		}
		size := binary.BigEndian.Uint64(body[offset:])
		offset += 8
		if size == 0 || size > uint64(limit) || size > uint64(len(body)-offset) {
			return nil, errors.New("terraform envelope field length is invalid")
		}
		parts = append(parts, body[offset:offset+int(size)])
		offset += int(size)
	}
	if offset != len(body) {
		return nil, errors.New("terraform envelope has trailing bytes")
	}
	return parts, nil
}

func openPrivateRoot(directory string) (*os.Root, error) {
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || directory == "/" {
		return nil, errors.New("terraform private intake path is invalid")
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("terraform intake directory is not private")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, errors.New("open terraform private intake")
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		_ = root.Close()
		return nil, errors.New("terraform intake root changed")
	}
	return root, nil
}

func ReadIntake(directory string, artifact domain.AcquiredArtifact) (Bundle, error) {
	if _, err := IdentityReference(artifact.Identity()); err != nil {
		return Bundle{}, err
	}
	parts := strings.Split(artifact.ContentHandle(), ":")
	if len(parts) != 3 || parts[0] != "intake" || parts[2] != artifact.Identity().Variant() {
		return Bundle{}, errors.New("terraform intake handle differs from subject")
	}
	run, err := domain.ParseRunID(parts[1])
	if err != nil {
		return Bundle{}, errors.New("terraform intake Run is invalid")
	}
	root, err := openPrivateRoot(directory)
	if err != nil {
		return Bundle{}, err
	}
	defer root.Close()
	beforeDir, err := root.Lstat(run.String())
	if err != nil || !beforeDir.IsDir() || beforeDir.Mode().Perm() != 0o700 || beforeDir.Mode()&os.ModeSymlink != 0 {
		return Bundle{}, errors.New("terraform Run intake directory is invalid")
	}
	runRoot, err := root.OpenRoot(run.String())
	if err != nil {
		return Bundle{}, err
	}
	defer runRoot.Close()
	openedDir, err := runRoot.Stat(".")
	if err != nil || !os.SameFile(beforeDir, openedDir) {
		return Bundle{}, errors.New("terraform Run intake directory changed")
	}
	before, err := runRoot.Lstat("provider.bundle")
	maximum := len(bundleHeader) + 8*len(bundleLimits)
	for _, limit := range bundleLimits {
		maximum += limit
	}
	if err != nil || !validPrivateFile(before) || before.Size() <= 0 || before.Size() > int64(maximum) || uint64(before.Size()) != artifact.SizeBytes() {
		return Bundle{}, errors.New("terraform intake file is invalid")
	}
	file, err := runRoot.OpenFile("provider.bundle", os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return Bundle{}, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !validPrivateFile(opened) || !os.SameFile(before, opened) {
		return Bundle{}, errors.New("terraform intake file changed")
	}
	body, err := io.ReadAll(io.LimitReader(file, before.Size()+1))
	if err != nil || int64(len(body)) != before.Size() {
		return Bundle{}, errors.New("terraform intake file read is incomplete")
	}
	after, err := runRoot.Lstat("provider.bundle")
	if err != nil || !validPrivateFile(after) || !os.SameFile(before, after) || after.Size() != before.Size() {
		return Bundle{}, errors.New("terraform intake file changed")
	}
	afterDir, err := root.Lstat(run.String())
	if err != nil || !os.SameFile(beforeDir, afterDir) {
		return Bundle{}, errors.New("terraform Run intake directory changed")
	}
	if sha256Hex(body) != artifact.Digest().String() {
		return Bundle{}, errors.New("terraform acquired envelope digest changed")
	}
	decoded, err := decodeParts(body, bundleHeader, bundleLimits)
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{decoded}, nil
}

func validPrivateFile(info os.FileInfo) bool {
	if info == nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Nlink == 1
}
