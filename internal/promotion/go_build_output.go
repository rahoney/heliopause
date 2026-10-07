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
	"reflect"
	"syscall"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	projectbuild "github.com/rahoney/heliopause/internal/artifact/projectbuild"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func (g *approvedGoProjectGuard) FreezeBuildInputs(ctx context.Context, run domain.RunID, selector string) (domain.ProjectBuildInputs, error) {
	if g == nil || g.buildInputs != nil || run.String() == "" || artifactgo.ValidateBuildPackage(selector) != nil {
		return domain.ProjectBuildInputs{}, errors.New("go build inputs are unavailable or already frozen")
	}
	snapshot, _, err := g.OpenBuildInputs(ctx)
	if err != nil {
		return domain.ProjectBuildInputs{}, err
	}
	source, err := g.SnapshotBuildSource(ctx)
	if err != nil {
		return domain.ProjectBuildInputs{}, err
	}
	cache, err := g.SnapshotBuildCache(ctx)
	if err != nil {
		return domain.ProjectBuildInputs{}, err
	}
	inputs, err := domain.NewProjectBuildInputs(snapshot, source, cache, run, selector)
	if err != nil {
		return domain.ProjectBuildInputs{}, err
	}
	g.buildInputs = &inputs
	return inputs, nil
}

type goBuildEvidenceRecord struct {
	ID     string `json:"id"`
	Handle string `json:"handle"`
	SHA256 string `json:"sha256"`
}
type goBuildOutputReceipt struct {
	Schema        int                          `json:"schema"`
	Run           string                       `json:"run"`
	Source        string                       `json:"source_sha256"`
	Cache         string                       `json:"cache_sha256"`
	Graph         string                       `json:"graph_sha256"`
	Output        string                       `json:"output_sha256"`
	Recipe        string                       `json:"recipe_sha256"`
	Policy        string                       `json:"policy"`
	PolicyVersion uint64                       `json:"policy_version"`
	Files         []artifactgo.BuildOutputFile `json:"files"`
	Evidence      []goBuildEvidenceRecord      `json:"evidence"`
}

func (g *approvedGoProjectGuard) verifyBuildApproval(ctx context.Context, build domain.ApprovedProjectBuild) ([]goBuildEvidenceRecord, error) {
	if g == nil || g.buildInputs == nil || g.buildSource == nil || g.buildCache == nil || !build.Valid() || !reflect.DeepEqual(build.Report().Inputs(), *g.buildInputs) || build.Report().Inputs().Source() != g.buildSource.artifact || build.Report().Inputs().Cache() != *g.buildCache {
		return nil, errors.New("go build approval differs from guarded inputs")
	}
	if _, err := domain.NewApprovedProjectBuild(build.Report(), build.Verification(), build.Result()); err != nil {
		return nil, err
	}
	if err := g.VerifyBuildSource(ctx); err != nil {
		return nil, err
	}
	snapshot, _, err := g.OpenBuildInputs(ctx)
	if err != nil || !reflect.DeepEqual(snapshot, build.Report().Inputs().Snapshot()) {
		return nil, errors.New("go build retained approval changed before publication")
	}
	expected := map[domain.EvidenceID]domain.Evidence{}
	for _, fact := range append(build.Verification().Evidence(), build.Report().Inspection().Evidence()...) {
		expected[fact.ID()] = fact
	}
	bindingID, _ := domain.NewEvidenceID("project-build-output-binding")
	if fact, ok := expected[bindingID]; !ok || fact.Kind() != "go-build-output-binding" || fact.Summary() != domain.BuildBindingSummary(*g.buildInputs, build.Report().Output(), build.Report().Binding()) {
		return nil, errors.New("go build output lacks exact recorded binding")
	}
	refs := build.Result().Evidence()
	if len(expected) != len(refs) {
		return nil, errors.New("go build Evidence coverage differs")
	}
	records := make([]goBuildEvidenceRecord, 0, len(refs))
	seen := map[domain.EvidenceID]bool{}
	for _, ref := range refs {
		fact, ok := expected[ref.ID()]
		if !ok || seen[ref.ID()] {
			return nil, errors.New("go build Evidence reference is substituted")
		}
		actual, digest, err := g.owner.cache.evidence.ReadReference(ctx, g.buildInputs.RunID(), ref, g.buildSource.artifact.Identity(), g.buildSource.artifact.Digest())
		if err != nil || actual != fact || digest.String() == "" {
			return nil, errors.New("go build required Evidence is unavailable or changed")
		}
		records = append(records, goBuildEvidenceRecord{ref.ID().String(), ref.Handle(), digest.String()})
		seen[ref.ID()] = true
	}
	return records, nil
}

type goBuildParentBinding struct {
	parent *os.Root
	name   string
	info   os.FileInfo
}

func verifyGoBuildParents(bindings []goBuildParentBinding) error {
	for _, binding := range bindings {
		current, err := binding.parent.Lstat(binding.name)
		if err != nil || !current.IsDir() || current.Mode() != binding.info.Mode() || !os.SameFile(current, binding.info) {
			return errors.New("go build publication parent changed")
		}
	}
	return nil
}

func openGoBuildParent(parent *os.Root, name string) (*os.Root, goBuildParentBinding, error) {
	before, err := parent.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		if err = parent.Mkdir(name, 0o700); err != nil {
			return nil, goBuildParentBinding{}, errors.New("create go build publication parent")
		}
		before, err = parent.Lstat(name)
	}
	if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 || before.Mode().Perm()&0o022 != 0 {
		return nil, goBuildParentBinding{}, errors.New("go build publication parent is untrusted")
	}
	root, err := parent.OpenRoot(name)
	if err != nil {
		return nil, goBuildParentBinding{}, errors.New("open go build publication parent")
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		_ = root.Close()
		return nil, goBuildParentBinding{}, errors.New("go build publication parent identity changed")
	}
	return root, goBuildParentBinding{parent, name, before}, nil
}

// PublishBuild consumes exact completed ALLOW and actual recorded Evidence.
// No resulting program is executed here. Existing outputs are never replaced.
func (g *approvedGoProjectGuard) PublishBuild(ctx context.Context, build domain.ApprovedProjectBuild) (domain.PublishedProjectBuild, error) {
	if g == nil || g.buildInputs == nil {
		return domain.PublishedProjectBuild{}, errors.New("go build inputs were not frozen")
	}
	return publishProjectBuildOutput(ctx, g.guard.root, g.owner.cache.intakeRoot, *g.buildInputs, build, g.verifyBuildApproval, g.owner.buildCheck)
}

func publishProjectBuildOutput(ctx context.Context, project *os.Root, intake string, inputs domain.ProjectBuildInputs, build domain.ApprovedProjectBuild, verify func(context.Context, domain.ApprovedProjectBuild) ([]goBuildEvidenceRecord, error), check func(string) error) (result domain.PublishedProjectBuild, resultErr error) {
	records, err := verify(ctx, build)
	if err != nil {
		return result, err
	}
	metadata, first, err := openGoBuildParent(project, ".heliopause")
	if err != nil {
		return result, err
	}
	defer func() {
		if metadata.Close() != nil {
			resultErr = errors.Join(resultErr, errors.New("close project build publication parent"))
			result = domain.PublishedProjectBuild{}
		}
	}()
	outputs, second, err := openGoBuildParent(metadata, "builds")
	if err != nil {
		return result, err
	}
	defer func() {
		if outputs.Close() != nil {
			resultErr = errors.Join(resultErr, errors.New("close project build output parent"))
			result = domain.PublishedProjectBuild{}
		}
	}()
	parents := []goBuildParentBinding{first, second}
	name := ".haa-" + inputs.Kind() + "-output-" + inputs.RunID().String()
	if err := outputs.Mkdir(name, 0o700); err != nil {
		return result, errors.New("create private project build output")
	}
	ownedInfo, err := outputs.Lstat(name)
	if err != nil || !ownedInfo.IsDir() {
		return result, errors.New("project build output ownership unavailable")
	}
	var stage *os.Root
	defer func() {
		if stage != nil && stage.Close() != nil {
			resultErr = errors.Join(resultErr, errors.New("close private project build output"))
		}
		if resultErr != nil {
			current, err := outputs.Lstat(name)
			if err != nil || !current.IsDir() || !os.SameFile(current, ownedInfo) {
				resultErr = errors.Join(resultErr, errors.New("project build output rollback identity unavailable"))
			} else if outputs.RemoveAll(name) != nil || syncGoTransactionRoot(outputs) != nil {
				resultErr = errors.Join(resultErr, errors.New("project build output rollback incomplete"))
			}
			result = domain.PublishedProjectBuild{}
		}
	}()
	stage, err = outputs.OpenRoot(name)
	if err != nil {
		return result, errors.New("open private project build output")
	}
	files, err := projectbuild.ReadBuildOutput(ctx, intake, build.Report().Output(), inputs.Kind()+"-output", func(name string, size int64, input io.Reader) error {
		file, err := stage.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o500)
		if err != nil {
			return errors.New("create staged project build file")
		}
		n, writeErr := io.Copy(file, input)
		syncErr := file.Sync()
		closeErr := file.Close()
		if n != size || errors.Join(writeErr, syncErr, closeErr) != nil {
			return errors.New("persist staged project build file")
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	decision, _ := build.Result().PolicyDecision()
	doc := goBuildOutputReceipt{1, inputs.RunID().String(), inputs.Source().Digest().String(), inputs.Cache().Digest().String(), inputs.Snapshot().GraphDigest().String(), build.Report().Output().Digest().String(), build.Report().Binding().ConfigDigest().String(), decision.PolicyID(), decision.Version(), files, records}
	body, err := json.Marshal(doc)
	if err != nil || len(body) > 1<<20 {
		return result, errors.New("project build output receipt is invalid")
	}
	file, err := stage.OpenFile(".haa-build.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return result, errors.New("create project build output receipt")
	}
	_, writeErr := file.Write(body)
	syncErr := file.Sync()
	closeErr := file.Close()
	if errors.Join(writeErr, syncErr, closeErr) != nil || syncGoTransactionRoot(stage) != nil {
		return result, errors.New("persist project build output receipt")
	}
	if err := check("BEFORE_PUBLISH"); err != nil {
		return result, err
	}
	if verifyGoBuildParents(parents) != nil || verifyGoBuildOutputTree(stage, ownedInfo, files, body) != nil {
		return result, errors.New("project build output changed before publication")
	}
	if _, err := verify(ctx, build); err != nil {
		return result, err
	}
	if err := renameRootNoReplace(outputs, name, inputs.RunID().String()); err != nil {
		return result, errors.New("project build output destination exists or publication failed")
	}
	name = inputs.RunID().String()
	if err := check("AFTER_PUBLISH"); err != nil {
		return result, err
	}
	current, err := outputs.Lstat(name)
	if err != nil || !os.SameFile(current, ownedInfo) || verifyGoBuildParents(parents) != nil || syncGoTransactionRoot(outputs) != nil || verifyGoBuildOutputTree(stage, ownedInfo, files, body) != nil {
		return result, errors.New("project build output publication could not be confirmed")
	}
	if _, err := verify(ctx, build); err != nil {
		return result, err
	}
	return domain.NewPublishedProjectBuild(build)
}

func verifyGoBuildOutputTree(root *os.Root, expected os.FileInfo, files []artifactgo.BuildOutputFile, receipt []byte) error {
	info, err := root.Stat(".")
	if err != nil || !os.SameFile(info, expected) || info.Mode() != os.ModeDir|0o700 {
		return errors.New("go build output tree identity changed")
	}
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := directory.ReadDir(artifactgo.MaxBuildOutputFiles + 2)
	closeErr := directory.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) || closeErr != nil || len(entries) != len(files)+1 {
		return errors.New("go build output tree inventory changed")
	}
	for _, item := range append(files, artifactgo.BuildOutputFile{Name: ".haa-build.json", Size: int64(len(receipt))}) {
		mode := os.FileMode(0o500)
		if item.Name == ".haa-build.json" {
			mode = 0o400
		}
		before, err := root.Lstat(item.Name)
		if err != nil || before.Mode() != mode || before.Size() != item.Size || !pypiSingleLink(before) {
			return errors.New("go build output file changed")
		}
		file, err := root.OpenFile(item.Name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return err
		}
		opened, statErr := file.Stat()
		hash := sha256.New()
		var content bytes.Buffer
		writer := io.Writer(hash)
		if item.Name == ".haa-build.json" {
			writer = &content
		}
		n, readErr := io.Copy(writer, io.LimitReader(file, item.Size+1))
		closeErr := file.Close()
		after, afterErr := root.Lstat(item.Name)
		if statErr != nil || !os.SameFile(before, opened) || readErr != nil || closeErr != nil || n != item.Size || afterErr != nil || !os.SameFile(before, after) || after.Mode() != mode || !pypiSingleLink(after) || (item.Name == ".haa-build.json" && !bytes.Equal(content.Bytes(), receipt)) || (item.Name != ".haa-build.json" && hex.EncodeToString(hash.Sum(nil)) != item.SHA256) {
			return errors.New("go build output consumed bytes changed")
		}
	}
	return nil
}
