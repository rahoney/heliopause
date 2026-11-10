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

type projectSourceMember struct {
	name string
	info os.FileInfo
	body []byte
	hash [32]byte
}

// projectSourceSnapshot retains anchored directory handles and exact source
// members until selection/observation is reconciled with the original project.
// It does not grant registry provenance or publication approval to local bytes.
type projectSourceSnapshot struct {
	project  string
	rootInfo os.FileInfo
	roots    []*os.Root
	members  []projectSourceMember
	limits   projectBuildInputLimits
}

// CargoProjectSource exposes the existing data-only capture to the guarded
// promotion consumer. It carries no registry or execution authority.
type CargoProjectSource = projectSourceSnapshot

type cargoProjectSource = projectSourceSnapshot

// GoProjectSource is the same anchored data witness with Go-specific limits.
type GoProjectSource = projectSourceSnapshot

func CaptureCargoProjectSource(ctx context.Context, project string) (*CargoProjectSource, error) {
	return captureCargoProject(ctx, project)
}

func (s *projectSourceSnapshot) Files() map[string][]byte         { return s.files() }
func (s *projectSourceSnapshot) Verify(ctx context.Context) error { return s.verify(ctx) }
func (s *projectSourceSnapshot) Archive() ([]byte, error)         { return s.archive() }

// BuildArchive preserves source bytes while using readonly transport metadata.
func (s *projectSourceSnapshot) BuildArchive() ([]byte, error) { return s.archiveWithFileMode(0o400) }
func (s *projectSourceSnapshot) Close() error                  { return s.close() }

// VerifyPublishedControls permits only the two exact selected root controls to
// change during publication. Every other source member retains its original
// inode, mode and content witness; hidden HAA records are not source authority.
func (s *projectSourceSnapshot) VerifyPublishedControls(ctx context.Context, controls map[string][]byte) (resultErr error) {
	if s == nil || len(s.roots) == 0 || len(controls) != 2 {
		return errors.New("cargo publication source witness is unavailable")
	}
	current, err := captureProjectSource(ctx, s.project, s.limits)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, current.close()) }()
	if !os.SameFile(s.rootInfo, current.rootInfo) || s.rootInfo.Mode() != current.rootInfo.Mode() {
		return errors.New("cargo publication source root changed")
	}
	expected := map[string]projectSourceMember{}
	for _, member := range s.members {
		if member.name != s.limits.controls[0] && member.name != s.limits.controls[1] {
			expected[member.name] = member
		}
	}
	seen := 0
	for _, member := range current.members {
		if member.name == s.limits.controls[0] || member.name == s.limits.controls[1] {
			body, exists := controls[member.name]
			if !exists || !bytes.Equal(body, member.body) {
				return errors.New("cargo published controls differ from selection")
			}
			seen++
			continue
		}
		before, exists := expected[member.name]
		if !exists || !os.SameFile(before.info, member.info) || before.info.Mode() != member.info.Mode() || before.hash != member.hash {
			return errors.New("cargo source changed during publication")
		}
		delete(expected, member.name)
	}
	if seen != 2 || len(expected) != 0 {
		return errors.New("cargo publication source coverage changed")
	}
	return nil
}

func captureCargoProject(ctx context.Context, project string) (*projectSourceSnapshot, error) {
	return captureProjectSource(ctx, project, cargoBuildInputLimits("project"))
}

func CaptureGoProjectSource(ctx context.Context, project string) (*GoProjectSource, error) {
	return captureProjectSource(ctx, project, goBuildInputLimits("project"))
}

// Only bounded data copying and identity retention are shared. Each source
// profile retains its own control grammar, ignored namespaces and budgets.
func captureProjectSource(ctx context.Context, project string, limits projectBuildInputLimits) (result *projectSourceSnapshot, resultErr error) {
	if ctx == nil || !filepath.IsAbs(project) || filepath.Clean(project) != project || project == "/" || (limits.variant != "cargo-source" && limits.variant != "go-source") {
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
	snapshot := &projectSourceSnapshot{project: project, rootInfo: info, roots: []*os.Root{root}, limits: limits}
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
		entries, readErr := file.ReadDir(2*limits.files + 1)
		closeErr := file.Close()
		if (readErr != nil && !errors.Is(readErr, io.EOF)) || closeErr != nil {
			return errors.New("read cargo source inventory")
		}
		for _, entry := range entries {
			count++
			if count > 2*limits.files {
				return errors.New("cargo source inventory exceeds entry bound")
			}
			if limits.variant == "cargo-source" && prefix == "" && entry.Name() == ".cargo" {
				return errors.New("project Cargo configuration is unsupported")
			}
			// As in the existing project source boundary, hidden host configuration
			// and HAA records are not input. Cargo output is private, not copied
			// from the user's existing target tree.
			if strings.HasPrefix(entry.Name(), ".") || (limits.variant == "cargo-source" && prefix == "" && entry.Name() == "target") || (limits.variant == "go-source" && entry.Name() == "vendor") {
				continue
			}
			name := path.Join(prefix, entry.Name())
			if len(name) > limits.maxPath || !utf8.ValidString(name) || name != path.Clean(name) || strings.ContainsAny(name, "\\:\x00\r\n\t") {
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
				snapshot.members = append(snapshot.members, projectSourceMember{name: name, info: info})
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
			if (name == limits.controls[0] || name == limits.controls[1]) && info.Size() > limits.control {
				return errors.New("cargo project control exceeds bound")
			}
			if files > limits.files || !info.Mode().IsRegular() || !goBuildSingleLink(info) || info.Size() < 0 || info.Size() > limits.file || info.Size() > limits.payload-total {
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
			snapshot.members = append(snapshot.members, projectSourceMember{name: name, info: info, body: body, hash: sha256.Sum256(body)})
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
	if limits.variant == "cargo-source" {
		if _, err := artifactcargo.SelectedLocalManifests(bodies); err != nil {
			return nil, err
		}
		if lock, present := bodies["Cargo.lock"]; present {
			if err := artifactcargo.ValidateProjectLock(lock); err != nil {
				return nil, err
			}
		}
	} else if err := validateGoProjectSource(bodies); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (s *projectSourceSnapshot) files() map[string][]byte {
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
func (s *projectSourceSnapshot) privateWorkspace(ctx context.Context) (workspace string, cleanup func() error, resultErr error) {
	if err := s.verify(ctx); err != nil {
		return "", nil, err
	}
	workspace, err := os.MkdirTemp("", "haa-"+s.limits.variant+"-")
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

func (s *projectSourceSnapshot) verify(ctx context.Context) (resultErr error) {
	if s == nil || len(s.roots) == 0 {
		return errors.New("cargo source snapshot is closed")
	}
	current, err := captureProjectSource(ctx, s.project, s.limits)
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

func (s *projectSourceSnapshot) archive() ([]byte, error) { return s.archiveWithFileMode(0o600) }

func (s *projectSourceSnapshot) archiveWithFileMode(fileMode int64) ([]byte, error) {
	if s == nil || len(s.roots) == 0 {
		return nil, errors.New("cargo source snapshot is closed")
	}
	var body bytes.Buffer
	w := tar.NewWriter(&body)
	for _, member := range s.members {
		h := &tar.Header{Name: member.name, Mode: fileMode, Size: int64(len(member.body)), Typeflag: tar.TypeReg}
		if s.limits.variant == "go-source" {
			h.Uid, h.Gid = 1000, 1000
			if member.name != s.limits.controls[0] && member.name != s.limits.controls[1] {
				h.Mode = 0o400
			}
		}
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
	if err := w.Close(); err != nil || int64(body.Len()) > s.limits.payload+int64(2*s.limits.files+2)*512 {
		return nil, errors.New("cargo source archive exceeds bound")
	}
	return body.Bytes(), nil
}

func (s *projectSourceSnapshot) close() error {
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
