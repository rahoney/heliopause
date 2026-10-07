package promotion

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"syscall"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const maxGoBuildSourceArchiveBytes = artifactgo.MaxModuleExpandedBytes + (2*artifactgo.MaxModuleFiles+2)*512

type goBuildSourceMember struct {
	name   string
	info   os.FileInfo
	digest [32]byte
}

type goBuildSourcePlan struct {
	artifact domain.AcquiredArtifact
	members  []goBuildSourceMember
}

// SnapshotBuildSource copies bounded project data under the live original
// guard. Hidden metadata/credential namespaces are never read or introduced.
// Public dependencies remain in the separately approved cache, not this source
// snapshot. The source digest proves byte identity, not registry provenance or
// an ALLOW. Build execution and its Policy decision are still required.
func (g *approvedGoProjectGuard) SnapshotBuildSource(ctx context.Context) (result domain.AcquiredArtifact, resultErr error) {
	if g == nil || g.buildSource != nil {
		return domain.AcquiredArtifact{}, errors.New("go build source snapshot is unavailable or already frozen")
	}
	if _, _, err := g.OpenBuildInputs(ctx); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	body, members, err := g.collectBuildSource(ctx)
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if err := ensureTrustedRoot(g.owner.cache.intakeRoot); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	run, err := domain.NewRunID()
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	parentInfo, err := os.Lstat(g.owner.cache.intakeRoot)
	if err != nil {
		return domain.AcquiredArtifact{}, errors.New("go build intake root unavailable")
	}
	parent, err := os.OpenRoot(g.owner.cache.intakeRoot)
	if err != nil {
		return domain.AcquiredArtifact{}, errors.New("open Go build intake root")
	}
	var owned *os.Root
	var ownedInfo os.FileInfo
	created := false
	defer func() {
		if owned != nil {
			resultErr = errors.Join(resultErr, owned.Close())
		}
		if resultErr != nil && created {
			current, err := parent.Lstat(run.String())
			if err != nil || ownedInfo == nil || !current.IsDir() || !os.SameFile(current, ownedInfo) {
				resultErr = errors.Join(resultErr, errors.New("go build intake cleanup identity unavailable"))
			} else {
				resultErr = errors.Join(resultErr, parent.RemoveAll(run.String()))
			}
		}
		resultErr = errors.Join(resultErr, parent.Close())
		if resultErr != nil {
			g.buildSource = nil
			result = domain.AcquiredArtifact{}
		}
	}()
	openedParent, err := parent.Stat(".")
	if err != nil || !os.SameFile(parentInfo, openedParent) {
		return domain.AcquiredArtifact{}, errors.New("go build intake root identity changed")
	}
	if err := parent.Mkdir(run.String(), 0o700); err != nil {
		return domain.AcquiredArtifact{}, errors.New("create Go build source intake")
	}
	created = true
	ownedInfo, err = parent.Lstat(run.String())
	if err != nil || !ownedInfo.IsDir() {
		return domain.AcquiredArtifact{}, errors.New("go build source intake identity unavailable")
	}
	owned, err = parent.OpenRoot(run.String())
	if err != nil {
		return domain.AcquiredArtifact{}, errors.New("open Go build source intake")
	}
	opened, err := owned.Stat(".")
	if err != nil || !os.SameFile(ownedInfo, opened) {
		return domain.AcquiredArtifact{}, errors.New("go build source intake identity changed")
	}
	file, err := owned.OpenFile("go-source.tar", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return domain.AcquiredArtifact{}, errors.New("create Go build source archive")
	}
	_, writeErr := file.Write(body)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return domain.AcquiredArtifact{}, errors.New("persist Go build source archive")
	}
	hash := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	source, err := domain.NewSourceID("project-local")
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	project := sha256.Sum256([]byte(g.context.Target().String()))
	identity, err := domain.NewResolvedArtifactIdentity(source, "project-"+hex.EncodeToString(project[:]), "snapshot", "go-source")
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":go-source", uint64(len(body)), "sha256:"+digest.String())
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	g.buildSource = &goBuildSourcePlan{artifact: artifact, members: members}
	if err := g.VerifyBuildSource(ctx); err != nil {
		g.buildSource = nil
		return domain.AcquiredArtifact{}, err
	}
	return artifact, nil
}

// VerifyBuildSource checks the complete copied source inventory, contents,
// original file identities and controls before any output can be published.
func (g *approvedGoProjectGuard) VerifyBuildSource(ctx context.Context) error {
	if g == nil || g.buildSource == nil {
		return errors.New("go build source was not frozen")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	body, members, err := g.collectBuildSource(ctx)
	if err != nil {
		return err
	}
	if len(members) != len(g.buildSource.members) {
		return errors.New("go build source inventory changed")
	}
	for i, member := range members {
		original := g.buildSource.members[i]
		if member.name != original.name || !os.SameFile(member.info, original.info) || member.info.Mode() != original.info.Mode() || member.digest != original.digest {
			return errors.New("go build source identity or content changed")
		}
	}
	hash := sha256.Sum256(body)
	if hex.EncodeToString(hash[:]) != g.buildSource.artifact.Digest().String() {
		return errors.New("go build source snapshot changed")
	}
	return g.VerifyUnchanged(ctx)
}

// Each directory is opened once and kept anchored while copying its direct
// members. Resolving a full relative path again could follow a replaced parent
// into a hidden namespace even when os.Root prevents an escape outside the root.
func (g *approvedGoProjectGuard) collectBuildSource(ctx context.Context) (result []byte, resultMembers []goBuildSourceMember, resultErr error) {
	if err := g.VerifyUnchanged(ctx); err != nil {
		return nil, nil, err
	}
	root, err := g.guard.root.OpenRoot(".")
	if err != nil {
		return nil, nil, errors.New("open Go build source root")
	}
	roots := []*os.Root{root}
	defer func() {
		for i := len(roots) - 1; i >= 0; i-- {
			resultErr = errors.Join(resultErr, roots[i].Close())
		}
		if resultErr != nil {
			result, resultMembers = nil, nil
		}
	}()
	openedRoot, err := root.Stat(".")
	if err != nil || !os.SameFile(openedRoot, g.guard.rootInfo) {
		return nil, nil, errors.New("go build source root identity changed")
	}
	type pendingMember struct {
		name   string
		leaf   string
		parent *os.Root
		info   os.FileInfo
	}
	var pending []pendingMember
	visited, fileCount := 0, 0
	var walk func(*os.Root, string) error
	walk = func(directory *os.Root, prefix string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		file, err := directory.Open(".")
		if err != nil {
			return errors.New("open Go build source inventory")
		}
		entries, readErr := file.ReadDir(2*artifactgo.MaxModuleFiles + 1)
		closeErr := file.Close()
		if readErr != nil && !errors.Is(readErr, io.EOF) || closeErr != nil {
			return errors.New("read Go build source inventory")
		}
		for _, entry := range entries {
			visited++
			if visited > 2*artifactgo.MaxModuleFiles {
				return errors.New("go build source exceeds bounded entry limits")
			}
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			name := path.Join(prefix, entry.Name())
			if len(name) > 1024 || path.Clean(name) != name || strings.ContainsAny(name, "\\\x00:\r\n") {
				return errors.New("go build source member is noncanonical")
			}
			info, err := directory.Lstat(entry.Name())
			if err != nil {
				return errors.New("go build source identity unavailable")
			}
			if info.IsDir() {
				child, err := directory.OpenRoot(entry.Name())
				if err != nil {
					return errors.New("open anchored Go source directory")
				}
				roots = append(roots, child)
				opened, err := child.Stat(".")
				if err != nil || !os.SameFile(info, opened) {
					return errors.New("go source directory identity changed")
				}
				pending = append(pending, pendingMember{name, entry.Name(), directory, info})
				if err := walk(child, name); err != nil {
					return err
				}
			} else if !info.Mode().IsRegular() || !pypiSingleLink(info) {
				return errors.New("go build source requires single-link regular files")
			} else {
				fileCount++
				if fileCount > artifactgo.MaxModuleFiles {
					return errors.New("go build source exceeds bounded file limits")
				}
				pending = append(pending, pendingMember{name, entry.Name(), directory, info})
			}
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return nil, nil, err
	}
	sort.Slice(pending, func(i, j int) bool { return pending[i].name < pending[j].name })
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	var members []goBuildSourceMember
	var total int64
	controls := map[string]domain.ProjectControlFile{}
	for _, control := range g.plan.controls {
		controls[control.Name()] = control
	}
	for _, member := range pending {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		info, err := member.parent.Lstat(member.leaf)
		if err != nil || !os.SameFile(info, member.info) || info.Mode() != member.info.Mode() {
			return nil, nil, errors.New("go source member identity changed before copy")
		}
		if info.IsDir() {
			if err := writer.WriteHeader(&tar.Header{Name: member.name, Mode: 0o700, Typeflag: tar.TypeDir}); err != nil {
				return nil, nil, errors.New("write Go source directory header")
			}
			members = append(members, goBuildSourceMember{name: member.name, info: info})
			continue
		}
		if !info.Mode().IsRegular() || !pypiSingleLink(info) || info.Size() < 0 || info.Size() > artifactgo.MaxModuleFileBytes || total+info.Size() > artifactgo.MaxModuleExpandedBytes {
			return nil, nil, errors.New("go build source exceeds bounded content or identity limits")
		}
		file, err := member.parent.OpenFile(member.leaf, os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return nil, nil, errors.New("open Go build source member")
		}
		opened, statErr := file.Stat()
		if statErr != nil || !opened.Mode().IsRegular() || !pypiSingleLink(opened) || !os.SameFile(info, opened) {
			closeErr := file.Close()
			return nil, nil, errors.Join(errors.New("go build source identity changed"), closeErr)
		}
		body, readErr := io.ReadAll(io.LimitReader(file, info.Size()+1))
		after, afterErr := file.Stat()
		closeErr := file.Close()
		if readErr != nil || afterErr != nil || closeErr != nil || int64(len(body)) != info.Size() || after.Size() != info.Size() || after.Mode() != info.Mode() || !after.ModTime().Equal(info.ModTime()) {
			return nil, nil, errors.New("go build source changed during copy")
		}
		total += int64(len(body))
		if control, exists := controls[member.name]; exists {
			if !control.Present() || !bytes.Equal(body, control.Body()) {
				return nil, nil, errors.New("go build source controls differ from guard")
			}
			delete(controls, member.name)
		}
		if err := writer.WriteHeader(&tar.Header{Name: member.name, Mode: 0o400, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			return nil, nil, errors.New("write Go build source header")
		}
		if _, err := writer.Write(body); err != nil {
			return nil, nil, errors.New("write Go build source data")
		}
		if archive.Len() > maxGoBuildSourceArchiveBytes {
			return nil, nil, errors.New("go build source archive exceeds bound")
		}
		members = append(members, goBuildSourceMember{member.name, info, sha256.Sum256(body)})
	}
	if len(controls) != 0 {
		return nil, nil, errors.New("go build source control inventory is incomplete")
	}
	if err := writer.Close(); err != nil || archive.Len() > maxGoBuildSourceArchiveBytes {
		return nil, nil, errors.New("go build source archive exceeds bound")
	}
	return archive.Bytes(), members, nil
}
