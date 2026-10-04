package gomodule

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/mod/module"
	"golang.org/x/mod/sumdb/dirhash"
	modzip "golang.org/x/mod/zip"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// ModuleCachePaths implements the canonical Go module-cache spelling. Paths
// are relative to an infrastructure-owned private materialization root.
func ModuleCachePaths(identity domain.ResolvedArtifactIdentity) (extracted, download string, err error) {
	if identity.Source() != Source() || identity.Variant() != "module" || module.Check(identity.Name(), identity.Version()) != nil {
		return "", "", errors.New("go cache module identity is invalid")
	}
	p, err := module.EscapePath(identity.Name())
	if err != nil {
		return "", "", err
	}
	v, err := module.EscapeVersion(identity.Version())
	if err != nil {
		return "", "", err
	}
	return p + "@" + v, "cache/download/" + p + "/@v/" + v, nil
}

// MaterializeModule is used only after Policy approval by the staging owner.
// It has no network or trust decision capability and never copies a resolver
// cache. The official ZIP validator/unzipper enforces Go's collision grammar.
func MaterializeModule(root string, bundle Bundle, identity domain.ResolvedArtifactIdentity, integrity string) error {
	extracted, download, err := ModuleCachePaths(identity)
	if err != nil {
		return err
	}
	wantZip, wantMod, err := ParseIntegrity(integrity)
	if err != nil {
		return err
	}
	zipSum, modSum, err := HashBundle(bundle, identity)
	if err != nil || zipSum != wantZip || modSum != wantMod {
		return errors.New("go module changed before cache materialization")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
		return errors.New("go module cache root is invalid")
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("go module cache root is not an exact directory")
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, download)), 0o755); err != nil {
		return errors.New("create private Go download tree")
	}
	info, err := json.Marshal(struct{ Version string }{identity.Version()})
	if err != nil {
		return err
	}
	for suffix, body := range map[string][]byte{".mod": bundle.Mod(), ".zip": bundle.Zip(), ".ziphash": []byte(wantZip), ".info": append(info, '\n')} {
		f, err := os.OpenFile(filepath.Join(root, download+suffix), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return errors.New("create exact Go cache member")
		}
		_, writeErr := f.Write(body)
		if writeErr == nil {
			writeErr = f.Sync()
		}
		closeErr := f.Close()
		if writeErr != nil || closeErr != nil {
			return errors.New("persist exact Go cache member")
		}
	}
	if err := modzip.Unzip(filepath.Join(root, extracted), module.Version{Path: identity.Name(), Version: identity.Version()}, filepath.Join(root, download+".zip")); err != nil {
		return errors.New("go module archive violates canonical ZIP rules")
	}
	return CheckMaterializedModule(root, bundle, identity, integrity)
}

// CheckMaterializedModule checks the payload and complete extracted source,
// rather than treating .ziphash or a cached checksum declaration as authority.
func CheckMaterializedModule(root string, bundle Bundle, identity domain.ResolvedArtifactIdentity, integrity string) error {
	extracted, download, err := ModuleCachePaths(identity)
	if err != nil {
		return err
	}
	wantZip, wantMod, err := ParseIntegrity(integrity)
	if err != nil {
		return err
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer r.Close()
	for suffix, want := range map[string][]byte{".mod": bundle.Mod(), ".zip": bundle.Zip(), ".ziphash": []byte(wantZip)} {
		got, err := readCacheMember(r, download+suffix, int64(len(want)))
		if err != nil || !bytes.Equal(got, want) {
			return errors.New("go cache payload changed")
		}
	}
	var count int
	var total int64
	dir := filepath.Join(root, extracted)
	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.Type()&os.ModeSymlink != 0 {
			return errors.New("go extracted cache contains an invalid path")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		count++
		total += info.Size()
		if !info.Mode().IsRegular() || count > MaxModuleFiles || info.Size() > MaxModuleFileBytes || total > MaxModuleExpandedBytes {
			return errors.New("go extracted cache exceeds bounds")
		}
		return nil
	})
	if err != nil {
		return err
	}
	got, err := dirhash.HashDir(dir, identity.Name()+"@"+identity.Version(), dirhash.Hash1)
	if err != nil || got != wantZip {
		return errors.New("go extracted source differs from authenticated archive")
	}
	mod, err := readCacheMember(r, download+".mod", MaxModBytes)
	if err != nil {
		return err
	}
	modSum, err := dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(mod)), nil })
	if err != nil || modSum != wantMod {
		return errors.New("go cache module definition differs from authenticated record")
	}
	return nil
}

func readCacheMember(root *os.Root, name string, limit int64) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > limit {
		return nil, errors.New("go cache member is not bounded regular content")
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return nil, errors.New("go cache member identity changed")
	}
	body, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(body)) != info.Size() {
		return nil, errors.New("go cache member read is incomplete")
	}
	return body, nil
}
