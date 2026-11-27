package sandbox

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestNodeExecutionInputReadonlyVariantPreservesWriterBoundary(t *testing.T) {
	t.Parallel()
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	volume := closureVolume{name: "input-fixture", createdAt: "fixed", mountpoint: "/owned/input", capacity: nodeExecutionWorkspaceLimit, projectBuildDestination: nodeExecutionRoot}
	writer, err := nodeExecutionInputCreateArguments(session, volume, false)
	oldWriter, oldErr := nodeMaterializationCreateArguments(session, volume)
	if err != nil || oldErr != nil || !slices.Equal(writer, oldWriter) {
		t.Fatal("old writer boundary changed", err, oldErr)
	}
	reader, err := nodeExecutionInputCreateArguments(session, volume, true)
	if err != nil {
		t.Fatal(err)
	}
	writerMount := slices.Index(writer, "--mount") + 1
	readerMount := slices.Index(reader, "--mount") + 1
	if reader[readerMount] != writer[writerMount]+",readonly" || !strings.HasSuffix(volume.tmpfsOptions(), ",noexec") {
		t.Fatal("readonly or noexec mount absent")
	}
	reader[readerMount] = writer[writerMount]
	if !slices.Equal(reader, writer) {
		t.Fatal("reader changed fixed runtime limits or environment")
	}
	base, ok := nodeMaterializationExpectedTopology()
	ro, roOK := nodeExecutionInputExpectedTopology(true)
	if !ok || !roOK || len(base) != len(ro) || !ro[len(ro)-1].ReadOnly {
		t.Fatal("readonly expected topology absent")
	}
	ro[len(ro)-1].ReadOnly = false
	if !slices.Equal(base, ro) {
		t.Fatal("ordinary topology changed")
	}
}

func TestNodeExecutionInputsRejectUnavailableContextBeforeIO(t *testing.T) {
	t.Parallel()
	for _, backend := range []*Backend{nil, {}} {
		if err := backend.ValidateNodeExecutionInputs(context.Background(), domain.ExecutionContext{}, artifactnpm.NodeExecutionLayout{}); err == nil {
			t.Fatal("unavailable current input accepted")
		}
	}
}

func TestNodeExecutionInputsPreserveCallerCancellationAtPreflight(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("caller canceled readonly input preparation")
	cancel(cause)
	var backend *Backend
	err := backend.ValidateNodeExecutionInputs(ctx, domain.ExecutionContext{}, artifactnpm.NodeExecutionLayout{})
	if !errors.Is(err, errNodeMaterialization) || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal("preflight lost validation error or original cancellation", err)
	}
}
