package gomodule

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/mod/module"
	"golang.org/x/mod/sumdb/dirhash"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	MaxModBytes            = 4 << 20
	MaxZipBytes            = 64 << 20
	MaxModuleFiles         = 10000
	MaxModuleFileBytes     = 64 << 20
	MaxModuleExpandedBytes = 200 << 20
	bundleHeader           = "HAA-GO-MODULE-1\n"
	bundleRequiredHeader   = "HAA-GO-MODULE-2\n"
)

// Bundle binds the exact proxy .mod and .zip bytes in one acquired subject.
// The length-delimited envelope is transport only; it grants no trust.
type Bundle struct {
	mod, zip []byte
	required []RequiredModControl
}

func (b Bundle) Mod() []byte { return bytes.Clone(b.mod) }
func (b Bundle) Zip() []byte { return bytes.Clone(b.zip) }
func (b Bundle) RequiredMods() []RequiredModControl {
	var result []RequiredModControl
	for _, control := range b.required {
		result = append(result, RequiredModControl{control.Version, bytes.Clone(control.Body)})
	}
	return result
}

type bundleControls struct {
	Mod      []byte               `json:"mod"`
	Required []RequiredModControl `json:"required_mods"`
}

func (b Bundle) ZipReader() (*zip.Reader, error) {
	return zip.NewReader(bytes.NewReader(b.zip), int64(len(b.zip)))
}

func encodeBundle(mod, archive []byte) ([]byte, error) {
	return encodeBundleWithRequiredMods(mod, archive, nil)
}

func encodeBundleWithRequiredMods(mod, archive []byte, required []RequiredModControl) ([]byte, error) {
	if len(mod) == 0 || len(mod) > MaxModBytes || len(archive) == 0 || len(archive) > MaxZipBytes {
		return nil, errors.New("go module intake exceeds content bounds")
	}
	control, header := mod, bundleHeader
	if len(required) > 0 {
		if len(required) > MaxRequiredModControls {
			return nil, errors.New("required Go control count exceeds bound")
		}
		remaining := MaxModBytes - len(mod)
		for _, control := range required {
			if len(control.Body) == 0 || len(control.Body) > remaining {
				return nil, errors.New("required Go controls exceed combined bound")
			}
			remaining -= len(control.Body)
		}
		var err error
		control, err = json.Marshal(bundleControls{mod, required})
		if err != nil || len(control) > MaxModBytes {
			return nil, errors.New("combined Go control envelope exceeds bound")
		}
		header = bundleRequiredHeader
	}
	body := make([]byte, len(header)+8+len(control)+len(archive))
	copy(body, header)
	binary.BigEndian.PutUint64(body[len(header):], uint64(len(control)))
	copy(body[len(header)+8:], control)
	copy(body[len(header)+8+len(control):], archive)
	return body, nil
}

// ReadIntake uses only a typed Run handle under the configured intake root.
// Reads are bounded, confined and digest-checked before exposing immutable bytes.
func ReadIntake(root string, artifact domain.AcquiredArtifact) (Bundle, error) {
	parts := strings.Split(artifact.ContentHandle(), ":")
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" || len(parts) != 3 || parts[0] != "intake" || parts[2] != "module" || artifact.Identity().Source() != Source() || artifact.Identity().Variant() != "module" {
		return Bundle{}, errors.New("go module intake binding is invalid")
	}
	if _, err := ParseReference(artifact.Identity().Name() + "@" + artifact.Identity().Version()); err != nil {
		return Bundle{}, err
	}
	run, err := domain.ParseRunID(parts[1])
	if err != nil {
		return Bundle{}, errors.New("go module intake Run is invalid")
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return Bundle{}, errors.New("go module intake root is invalid")
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return Bundle{}, err
	}
	defer r.Close()
	opened, err := r.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return Bundle{}, errors.New("go module intake root changed")
	}
	dir, err := r.Lstat(run.String())
	if err != nil || !dir.IsDir() || dir.Mode()&os.ModeSymlink != 0 {
		return Bundle{}, errors.New("go module intake directory is invalid")
	}
	name := run.String() + "/module.bundle"
	before, err := r.Lstat(name)
	if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || uint64(before.Size()) != artifact.SizeBytes() || before.Size() > MaxModBytes+MaxZipBytes+int64(len(bundleHeader))+8 {
		return Bundle{}, errors.New("go module intake file is invalid")
	}
	f, err := r.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return Bundle{}, err
	}
	defer f.Close()
	opened, err = f.Stat()
	if err != nil || !os.SameFile(before, opened) || !opened.Mode().IsRegular() {
		return Bundle{}, errors.New("go module intake file changed")
	}
	body, err := io.ReadAll(io.LimitReader(f, before.Size()+1))
	if err != nil || int64(len(body)) != before.Size() {
		return Bundle{}, errors.New("go module intake read is incomplete")
	}
	after, err := r.Lstat(name)
	if err != nil || !os.SameFile(before, after) {
		return Bundle{}, errors.New("go module intake file changed")
	}
	hash := sha256.Sum256(body)
	if hex.EncodeToString(hash[:]) != artifact.Digest().String() {
		return Bundle{}, errors.New("go module intake content changed")
	}
	if len(body) < len(bundleHeader)+8 {
		return Bundle{}, errors.New("go module intake envelope is invalid")
	}
	header := string(body[:len(bundleHeader)])
	if header != bundleHeader && header != bundleRequiredHeader {
		return Bundle{}, errors.New("go module intake envelope version is invalid")
	}
	n := binary.BigEndian.Uint64(body[len(bundleHeader) : len(bundleHeader)+8])
	start := uint64(len(bundleHeader) + 8)
	if n == 0 || n > MaxModBytes || start+n >= uint64(len(body)) || uint64(len(body))-start-n > MaxZipBytes {
		return Bundle{}, errors.New("go module intake envelope bounds are invalid")
	}
	bundle := Bundle{mod: body[start : start+n], zip: body[start+n:]}
	if header == bundleRequiredHeader {
		var controls bundleControls
		decoder := json.NewDecoder(bytes.NewReader(bundle.mod))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&controls) != nil {
			return Bundle{}, errors.New("required Go control envelope is invalid")
		}
		canonical, err := json.Marshal(controls)
		if err != nil || !bytes.Equal(canonical, bundle.mod) || len(controls.Mod) == 0 || len(controls.Required) == 0 || len(controls.Required) > MaxRequiredModControls {
			return Bundle{}, errors.New("required Go control envelope is not canonical")
		}
		previous := ""
		for _, control := range controls.Required {
			if !validModuleVersion(artifact.Identity().Name(), control.Version) || control.Version <= previous || len(control.Body) == 0 || len(control.Body) > MaxModBytes {
				return Bundle{}, errors.New("required Go envelope member is invalid")
			}
			previous = control.Version
		}
		bundle.mod, bundle.required = controls.Mod, controls.Required
	}
	return bundle, nil
}

// ParseIntegrity accepts only the normalized pair frozen by the resolver.
func ParseIntegrity(value string) (archive, mod string, err error) {
	archive, mod, _, err = parseModuleIntegrity(value)
	return archive, mod, err
}

// HashBundle uses Go's h1 algorithm only after bounded ZIP validation. The
// byte digest of the envelope remains the acquired content identity.
func HashBundle(bundle Bundle, identity domain.ResolvedArtifactIdentity) (string, string, error) {
	reader, err := bundle.ZipReader()
	if err != nil {
		return "", "", err
	}
	files, err := CheckedFiles(reader, identity)
	if err != nil {
		return "", "", err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	archiveSum, err := dirhash.Hash1(names, func(name string) (io.ReadCloser, error) { return files[name].Open() })
	if err != nil {
		return "", "", err
	}
	modSum, err := dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(bundle.Mod())), nil })
	return archiveSum, modSum, err
}

// CheckedFiles bounds the input before hashing or static parsing and rejects
// aliases, duplicate paths, links and noncanonical module prefixes.
func CheckedFiles(reader *zip.Reader, identity domain.ResolvedArtifactIdentity) (map[string]*zip.File, error) {
	if identity.Source() != Source() || identity.Variant() != "module" || module.Check(identity.Name(), identity.Version()) != nil || len(reader.File) == 0 || len(reader.File) > MaxModuleFiles {
		return nil, errors.New("go module ZIP identity or entry count is invalid")
	}
	prefix := identity.Name() + "@" + identity.Version() + "/"
	files := map[string]*zip.File{}
	folds := map[string]string{}
	var total uint64
	for _, f := range reader.File {
		rel, ok := strings.CutPrefix(f.Name, prefix)
		if !ok || rel == "" || rel != path.Clean(rel) || strings.ContainsAny(rel, "\\\r\n") || module.CheckFilePath(rel) != nil || !f.Mode().IsRegular() || f.UncompressedSize64 > MaxModuleFileBytes || f.UncompressedSize64 > MaxModuleExpandedBytes-total {
			return nil, errors.New("go module ZIP entry is invalid or exceeds bounds")
		}
		if _, exists := files[f.Name]; exists {
			return nil, errors.New("go module ZIP contains duplicate entry")
		}
		fold := strings.ToLower(rel)
		if previous, exists := folds[fold]; exists && previous != rel {
			return nil, errors.New("go module ZIP contains case collision")
		}
		files[f.Name] = f
		folds[fold] = rel
		total += f.UncompressedSize64
	}
	return files, nil
}
