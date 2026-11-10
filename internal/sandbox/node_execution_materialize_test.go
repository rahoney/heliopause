package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type nodeCleanupRunner struct {
	id, name, owner, image      string
	queryErr, removeErr         error
	missing, ambiguous, remains bool
	removed                     bool
}

func (r *nodeCleanupRunner) Output(_ context.Context, _ string, args ...string) ([]byte, error) {
	switch args[0] {
	case "ps":
		if r.queryErr != nil {
			return nil, r.queryErr
		}
		if r.missing || r.removed && !r.remains {
			return nil, nil
		}
		if r.ambiguous {
			return []byte(r.id + "\n" + strings.Repeat("b", 64)), nil
		}
		return []byte(r.id + "\n"), nil
	case "inspect":
		return json.Marshal([]any{map[string]any{"Id": r.id, "Name": "/" + r.name, "Config": map[string]any{"Image": r.image, "Labels": map[string]string{nodeMaterializationOwnerLabel: r.owner}}}})
	case "rm":
		r.removed = true
		return nil, r.removeErr
	default:
		return nil, errors.New("unexpected fixture command")
	}
}

func TestNodeMaterializationCleanupRequiresExactOwnedResource(t *testing.T) {
	t.Parallel()
	primary := errors.New("fixture provider failure")
	for _, failure := range []string{"", "absent", "query", "ambiguous", "image", "owner", "name", "identity", "remove", "still present"} {
		t.Run(failure, func(t *testing.T) {
			r := &nodeCleanupRunner{id: strings.Repeat("a", 64), name: "heliopause-fixture", owner: "fixture", image: nodeImageReference}
			expected := r.id
			switch failure {
			case "absent":
				r.missing = true
			case "query":
				r.queryErr = primary
			case "ambiguous":
				r.ambiguous = true
			case "image":
				r.image = "foreign-image"
			case "owner":
				r.owner = "foreign-owner"
			case "name":
				r.name = "foreign-name"
			case "identity":
				expected = strings.Repeat("b", 64)
			case "remove":
				r.removeErr = primary
			case "still present":
				r.remains = true
			}
			err := cleanupNodeMaterialization(context.Background(), r, "heliopause-fixture", "fixture", expected)
			wantSuccess := failure == "" || failure == "absent"
			if (err == nil) != wantSuccess {
				t.Fatalf("cleanup failure=%s error=%v", failure, err)
			}
			if failure == "query" || failure == "remove" {
				if !errors.Is(err, primary) {
					t.Fatal("original provider cause lost")
				}
			}
			mayRemove := failure == "" || failure == "remove" || failure == "still present"
			if r.removed != mayRemove {
				t.Fatal("uncertain or foreign resource was removed")
			}
		})
	}
}

func TestNodeMaterializationPayloadHasOnlyFrozenControllerData(t *testing.T) {
	t.Parallel()
	subject := sandboxRequest(t).Artifact()
	body := bytes.Repeat([]byte{'a'}, int(subject.SizeBytes()))
	digest, err := pnpmPayloadSHA256(body)
	if err != nil {
		t.Fatal(err)
	}
	subject, err = domain.NewAcquiredArtifact(subject.Identity(), digest, subject.ContentHandle(), uint64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	archive := pnpmInspectionArchive{subject, "artifacts/" + subject.Digest().String() + ".tgz", body}
	manifest, err := domain.NewProjectControlFile("package.json", []byte(`{"private":true}`), true)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := domain.NewProjectControlFile("package-lock.json", []byte(`{"lockfileVersion":3}`), true)
	if err != nil {
		t.Fatal(err)
	}
	controls := []domain.ProjectControlFile{manifest, lock}
	payload, err := nodeMaterializationPayload(context.Background(), []pnpmInspectionArchive{archive, archive}, controls)
	if err != nil {
		t.Fatal(err)
	}
	reader := tar.NewReader(bytes.NewReader(payload))
	files := map[string][]byte{}
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Uid != 1000 || header.Gid != 1000 {
			t.Fatal("payload owner changed")
		}
		if header.Typeflag == tar.TypeReg {
			body, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			if _, duplicate := files[header.Name]; duplicate {
				t.Fatal("duplicate frozen archive")
			}
			files[header.Name] = body
		}
	}
	if len(files) != 3 || !bytes.Equal(files["project/package.json"], manifest.Body()) || !bytes.Equal(files["project/.heliopause/"+archive.name], body) {
		t.Fatal("frozen payload differed")
	}
	for _, bad := range [][]domain.ProjectControlFile{{manifest}, {manifest, manifest}, {manifest, domain.ProjectControlFile{}}} {
		if _, err := nodeMaterializationPayload(context.Background(), []pnpmInspectionArchive{archive}, bad); err == nil {
			t.Fatal("incomplete/duplicate controls accepted")
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("fixture canceled")
	cancel(cause)
	if _, err := nodeMaterializationPayload(ctx, []pnpmInspectionArchive{archive}, controls); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal("payload cancellation lost", err)
	}
}

func TestNodeMaterializationWorkspaceRetainsUncertainIdentityAndFixedBounds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	owner, source := "node-fixture", strings.Repeat("a", 64)
	name, err := closureVolumeName(owner + "-node-materialization")
	if err != nil {
		t.Fatal(err)
	}
	v := closureVolume{name: name, transaction: owner + "-node-materialization", manifestID: source, createdAt: "fixed", mountpoint: "/owned/node", capacity: nodeExecutionWorkspaceLimit, projectBuildDestination: nodeExecutionRoot}
	r := &pipxVolumeRunner{volume: v}
	actual, err := createNodeMaterializationVolume(ctx, r, owner, source)
	if err != nil || actual.destination() != nodeExecutionRoot || actual.projectBuildExecutable || actual.tmpfsOptions() != "size=134217728,nosuid,nodev,uid=1000,gid=1000,mode=0700,noexec" {
		t.Fatal("fixed DATA volume refused", err)
	}
	for _, failure := range []string{"volume create", "volume inspect"} {
		r.fail, r.removed = failure, false
		uncertain, err := createNodeMaterializationVolume(ctx, r, owner, source)
		if err == nil || uncertain.name != name || r.removed {
			t.Fatal("uncertain volume identity lost or removed", failure, err)
		}
		r.fail = "volume inspect"
		if err := closeWritableOutputVolume(ctx, r, &uncertain, nodeExecutionRoot); err == nil || r.removed {
			t.Fatal("unknown volume cleanup trusted", failure)
		}
	}
	r.fail = ""
	for _, capacity := range []int64{1, nodeExecutionWorkspaceLimit - 1, nodeExecutionWorkspaceLimit + 1} {
		if _, err := createInputVolume(ctx, r, owner, source, capacity, false, projectBuildVolumeConfig{nodeExecutionRoot, false}); err == nil {
			t.Fatal("workspace capacity override")
		}
	}
	r.volume.manifestID = strings.Repeat("b", 64)
	r.removed = false
	if err := closeWritableOutputVolume(ctx, r, &actual, nodeExecutionRoot); err == nil || r.removed {
		t.Fatal("foreign volume removed")
	}
	r.volume = v
	if err := closeWritableOutputVolume(ctx, r, &actual, nodeExecutionRoot); err != nil || !r.removed {
		t.Fatal("owned unattached volume not cleaned", err)
	}
}

func TestNodeMaterializationWorkspaceTopologyAndCreatePreserveOrdinaryNPM(t *testing.T) {
	t.Parallel()
	base, ok := observerExpectedTopology("npm-lifecycle")
	actual, actualOK := nodeMaterializationExpectedTopology()
	if !ok || !actualOK || len(actual) != len(base)+1 || !slices.Equal(actual[:len(base)], base) || actual[len(base)] != (observerMountExpectation{nodeExecutionRoot, "workspace", "/tmp", "9p", false, false, false, false}) {
		t.Fatal("scoped DATA topology changed ordinary mounts")
	}
	actual[0].ReadOnly = false
	again, _ := observerExpectedTopology("npm-lifecycle")
	if !slices.Equal(again, base) {
		t.Fatal("ordinary topology mutated")
	}
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	v := closureVolume{name: "owned-node", createdAt: "fixed", mountpoint: "/owned/node", capacity: nodeExecutionWorkspaceLimit, projectBuildDestination: nodeExecutionRoot}
	args, err := nodeMaterializationCreateArguments(session, v)
	if err != nil || !slices.Contains(args, "type=volume,source=owned-node,target="+nodeExecutionRoot+",volume-nocopy") || !slices.Contains(args, "none") || !slices.Contains(args, "--read-only") || !slices.Contains(args, nodeImageReference) {
		t.Fatal("fixed create boundary missing", err)
	}
	mount := slices.Index(args, "--mount")
	withoutMount := append(slices.Clone(args[:mount]), args[mount+2:]...)
	if !slices.Equal(withoutMount, createArguments(session)) {
		t.Fatal("ordinary Node runtime limits or environment changed")
	}
	for _, bad := range []closureVolume{{}, {projectBuildDestination: pnpmProjectRoot}, {projectBuildDestination: nodeExecutionRoot, projectBuildExecutable: true}, {projectBuildDestination: nodeExecutionRoot, capacity: 1}} {
		if _, err := nodeMaterializationCreateArguments(session, bad); err == nil {
			t.Fatal("foreign/exec/unattested workspace accepted")
		}
	}
}

func TestNodeMaterializationUsesOnlyFixedPinnedControllerCommand(t *testing.T) {
	t.Parallel()
	args := nodeMaterializationArguments(strings.Repeat("a", 64))
	joined := strings.Join(args, " ")
	for _, required := range []string{"/usr/local/bin/node -- /usr/local/lib/node_modules/npm/bin/npm-cli.js", "--prefix " + nodeExecutionProject + " ci", "--offline", "--ignore-scripts=true", "--bin-links=false", "--workspaces=false", "--userconfig=/tmp/haa-user.npmrc", "--globalconfig=/tmp/haa-global.npmrc"} {
		if !strings.Contains(joined, required) {
			t.Fatal("fixed controller argument absent", required)
		}
	}
	if !slices.Equal(args[len(args)-3:], []string{"/bin/sh", "-ceu", nodeMaterializationCommand}) || !strings.HasSuffix(nodeMaterializationCommand, "\n:") || strings.Contains(nodeMaterializationCommand, "exec ") || strings.Contains(nodeMaterializationCommand, "$@") {
		t.Fatal("fixed waiting controller or closed command changed")
	}
	var missing *Backend
	if result, err := missing.MaterializeNodeExecution(context.Background(), domain.ExecutionContext{}, nil, nil); err == nil || result.Valid() {
		t.Fatal("unavailable backend accepted execution")
	}
}
