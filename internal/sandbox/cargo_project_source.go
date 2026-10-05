package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode/utf8"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
)

type cargoSourceMember struct {
	name string
	info os.FileInfo
	body []byte
	hash [32]byte
}

// cargoProjectSource retains anchored directory handles and exact source
// members until selection/observation is reconciled with the original project.
// It does not grant registry provenance or publication approval to local bytes.
type cargoProjectSource struct {
	project  string
	rootInfo os.FileInfo
	roots    []*os.Root
	members  []cargoSourceMember
}

// CargoProjectSource exposes the existing data-only capture to the guarded
// promotion consumer. It carries no registry or execution authority.
type CargoProjectSource = cargoProjectSource

func CaptureCargoProjectSource(ctx context.Context, project string) (*CargoProjectSource, error) {
	return captureCargoProject(ctx, project)
}

func (s *cargoProjectSource) Files() map[string][]byte         { return s.files() }
func (s *cargoProjectSource) Verify(ctx context.Context) error { return s.verify(ctx) }
func (s *cargoProjectSource) Archive() ([]byte, error)         { return s.archive() }
func (s *cargoProjectSource) Close() error                     { return s.close() }

func captureCargoProject(ctx context.Context, project string) (result *cargoProjectSource, resultErr error) {
	if ctx == nil || !filepath.IsAbs(project) || filepath.Clean(project) != project || project == "/" {
		return nil, errors.New("cargo source project is invalid")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Lstat(project)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("cargo source root is not an exact directory")
	}
	root, err := os.OpenRoot(project)
	if err != nil {
		return nil, errors.New("open anchored cargo source")
	}
	snapshot := &cargoProjectSource{project: project, rootInfo: info, roots: []*os.Root{root}}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, snapshot.close())
			result = nil
		}
	}()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("cargo source root identity changed")
	}
	count, files, total := 0, 0, int64(0)
	var walk func(*os.Root, string) error
	walk = func(directory *os.Root, prefix string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		file, err := directory.Open(".")
		if err != nil {
			return errors.New("open cargo source inventory")
		}
		entries, readErr := file.ReadDir(2*artifactcargo.MaxCrateFiles + 1)
		closeErr := file.Close()
		if (readErr != nil && !errors.Is(readErr, io.EOF)) || closeErr != nil {
			return errors.New("read cargo source inventory")
		}
		for _, entry := range entries {
			count++
			if count > 2*artifactcargo.MaxCrateFiles {
				return errors.New("cargo source inventory exceeds entry bound")
			}
			if prefix == "" && entry.Name() == ".cargo" {
				return errors.New("project Cargo configuration is unsupported")
			}
			// As in the existing project source boundary, hidden host configuration
			// and HAA records are not input. Cargo output is private, not copied
			// from the user's existing target tree.
			if strings.HasPrefix(entry.Name(), ".") || (prefix == "" && entry.Name() == "target") {
				continue
			}
			name := path.Join(prefix, entry.Name())
			if len(name) > 4096 || !utf8.ValidString(name) || name != path.Clean(name) || strings.ContainsAny(name, "\\:\x00\r\n\t") {
				return errors.New("cargo source member is noncanonical")
			}
			info, err := directory.Lstat(entry.Name())
			if err != nil {
				return errors.New("cargo source member identity unavailable")
			}
			if info.IsDir() {
				child, err := directory.OpenRoot(entry.Name())
				if err != nil {
					return errors.New("open anchored cargo source directory")
				}
				snapshot.roots = append(snapshot.roots, child)
				opened, err := child.Stat(".")
				if err != nil || !os.SameFile(info, opened) {
					return errors.New("cargo source directory identity changed")
				}
				snapshot.members = append(snapshot.members, cargoSourceMember{name: name, info: info})
				if err := walk(child, name); err != nil {
					return err
				}
				current, err := directory.Lstat(entry.Name())
				if err != nil || !os.SameFile(info, current) || current.Mode() != info.Mode() {
					return errors.New("cargo source directory changed during capture")
				}
				continue
			}
			files++
			if (name == "Cargo.toml" || name == "Cargo.lock") && info.Size() > artifactcargo.MaxProjectControlBytes {
				return errors.New("cargo project control exceeds bound")
			}
			if files > artifactcargo.MaxCrateFiles || !info.Mode().IsRegular() || !goBuildSingleLink(info) || info.Size() < 0 || info.Size() > artifactcargo.MaxCrateFileBytes || info.Size() > artifactcargo.MaxCrateExpandedBytes-total {
				return errors.New("cargo source requires bounded single-link regular files")
			}
			file, err := directory.OpenFile(entry.Name(), os.O_RDONLY|syscall.O_NONBLOCK, 0)
			if err != nil {
				return errors.New("open cargo source file")
			}
			opened, statErr := file.Stat()
			if statErr != nil || !opened.Mode().IsRegular() || !goBuildSingleLink(opened) || !os.SameFile(info, opened) {
				return errors.Join(errors.New("cargo source file identity changed"), file.Close())
			}
			body, readErr := io.ReadAll(io.LimitReader(file, info.Size()+1))
			after, statErr := file.Stat()
			closeErr := file.Close()
			current, currentErr := directory.Lstat(entry.Name())
			if readErr != nil || statErr != nil || closeErr != nil || currentErr != nil || !current.Mode().IsRegular() || !os.SameFile(info, current) || !goBuildSingleLink(current) || int64(len(body)) != info.Size() || after.Mode() != info.Mode() || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
				return errors.New("cargo source changed during capture")
			}
			total += int64(len(body))
			snapshot.members = append(snapshot.members, cargoSourceMember{name: name, info: info, body: body, hash: sha256.Sum256(body)})
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return nil, err
	}
	currentRoot, err := os.Lstat(project)
	if err != nil || !os.SameFile(info, currentRoot) || info.Mode() != currentRoot.Mode() {
		return nil, errors.New("cargo source root changed during capture")
	}
	sort.Slice(snapshot.members, func(i, j int) bool { return snapshot.members[i].name < snapshot.members[j].name })
	bodies := snapshot.files()
	if _, err := artifactcargo.SelectedLocalManifests(bodies); err != nil {
		return nil, err
	}
	if lock, present := bodies["Cargo.lock"]; present {
		if err := artifactcargo.ValidateProjectLock(lock); err != nil {
			return nil, err
		}
	}
	return snapshot, nil
}

func (s *cargoProjectSource) files() map[string][]byte {
	result := map[string][]byte{}
	for _, m := range s.members {
		if m.info.Mode().IsRegular() {
			result[m.name] = bytes.Clone(m.body)
		}
	}
	return result
}

// privateWorkspace copies only the anchored data inventory. Cargo is never
// invoked while this code has access to the caller's original directory.
func (s *cargoProjectSource) privateWorkspace(ctx context.Context) (workspace string, cleanup func() error, resultErr error) {
	if err := s.verify(ctx); err != nil {
		return "", nil, err
	}
	workspace, err := os.MkdirTemp("", "haa-cargo-project-")
	if err != nil {
		return "", nil, errors.New("create private Cargo project")
	}
	cleanup = func() error {
		if err := os.RemoveAll(workspace); err != nil {
			return errors.New("private Cargo project cleanup failed")
		}
		return nil
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, cleanup())
			workspace = ""
			cleanup = nil
		}
	}()
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return workspace, cleanup, errors.New("open private Cargo project")
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	for _, member := range s.members {
		if err := ctx.Err(); err != nil {
			return workspace, cleanup, err
		}
		if member.info.IsDir() {
			if err := root.Mkdir(member.name, 0o700); err != nil {
				return workspace, cleanup, errors.New("copy private Cargo directory")
			}
			continue
		}
		file, err := root.OpenFile(member.name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return workspace, cleanup, errors.New("copy private Cargo file")
		}
		_, writeErr := file.Write(member.body)
		if errors.Join(writeErr, file.Sync(), file.Close()) != nil {
			return workspace, cleanup, errors.New("write private Cargo data")
		}
	}
	if err := s.verify(ctx); err != nil {
		return workspace, cleanup, err
	}
	return workspace, cleanup, nil
}

func (s *cargoProjectSource) verify(ctx context.Context) (resultErr error) {
	if s == nil || len(s.roots) == 0 {
		return errors.New("cargo source snapshot is closed")
	}
	current, err := captureCargoProject(ctx, s.project)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, current.close()) }()
	if !os.SameFile(s.rootInfo, current.rootInfo) || s.rootInfo.Mode() != current.rootInfo.Mode() || len(s.members) != len(current.members) {
		return errors.New("cargo source root or inventory changed")
	}
	for index, member := range s.members {
		other := current.members[index]
		if member.name != other.name || !os.SameFile(member.info, other.info) || member.info.Mode() != other.info.Mode() || member.hash != other.hash {
			return errors.New("cargo source member identity or bytes changed")
		}
	}
	return nil
}

func (s *cargoProjectSource) archive() ([]byte, error) {
	if s == nil || len(s.roots) == 0 {
		return nil, errors.New("cargo source snapshot is closed")
	}
	var body bytes.Buffer
	w := tar.NewWriter(&body)
	for _, member := range s.members {
		h := &tar.Header{Name: member.name, Mode: 0o600, Size: int64(len(member.body)), Typeflag: tar.TypeReg}
		if member.info.IsDir() {
			h.Typeflag = tar.TypeDir
			h.Mode = 0o700
			h.Size = 0
		}
		if err := w.WriteHeader(h); err != nil {
			return nil, errors.New("write bounded cargo source header")
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := w.Write(member.body); err != nil {
				return nil, errors.New("write bounded cargo source data")
			}
		}
	}
	if err := w.Close(); err != nil || body.Len() > artifactcargo.MaxCrateExpandedBytes+(2*artifactcargo.MaxCrateFiles+2)*512 {
		return nil, errors.New("cargo source archive exceeds bound")
	}
	return body.Bytes(), nil
}

func (s *cargoProjectSource) close() error {
	if s == nil {
		return nil
	}
	var result error
	for index := len(s.roots) - 1; index >= 0; index-- {
		result = errors.Join(result, s.roots[index].Close())
	}
	s.roots = nil
	if result != nil {
		return errors.New("cargo source handle cleanup failed")
	}
	return result
}
