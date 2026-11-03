package sandbox

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type responderControllerModel struct {
	terminate chan struct{}
	failure   error
}

func (*responderControllerModel) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected DATA command")
}
func (m *responderControllerModel) StartContainerInputOutput(ctx context.Context, input io.Reader, output io.Writer, id string) error {
	if !exactObservationContainerID(id) {
		return errors.New("invalid init")
	}
	for _, part := range []string{"RE", "ADY", "\n"} {
		if _, err := io.WriteString(output, part); err != nil {
			return err
		}
	}
	command, err := io.ReadAll(input)
	if err != nil {
		return err
	}
	if string(command) != "STOP\n" {
		return errors.New("controller stop missing")
	}
	if _, err := io.WriteString(output, "STOPPED\n"); err != nil {
		return err
	}
	select {
	case <-m.terminate:
		return m.failure
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestResponderControllerRequiresStopAndIndependentTermination(t *testing.T) {
	failure := errors.New("original init failure")
	for _, variant := range []string{"complete", "unconfirmed", "failed"} {
		t.Run(variant, func(t *testing.T) {
			model := &responderControllerModel{terminate: make(chan struct{})}
			if variant == "failed" {
				model.failure = failure
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			responder, err := startResponderInit(ctx, model, strings.Repeat("a", 64))
			if err != nil {
				t.Fatal(err)
			}
			select {
			case <-responder.done:
				t.Fatal("readiness became completion")
			default:
			}
			if err = responder.stop(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case <-responder.done:
				t.Fatal("STOP acknowledgement became termination")
			default:
			}
			close(model.terminate)
			err = responder.waitAfterTermination(ctx, variant != "unconfirmed")
			switch variant {
			case "complete":
				if err != nil {
					t.Fatal(err)
				}
			case "unconfirmed":
				if err == nil {
					t.Fatal("unconfirmed termination accepted")
				}
			case "failed":
				if !errors.Is(err, failure) {
					t.Fatal("original controller failure lost")
				}
			}
		})
	}
}

func TestResponderControllerRejectsForgedAndUnboundedFrames(t *testing.T) {
	for _, body := range []string{"STOPPED\n", "READY\nSTOPPED\n", "READY\nREADY\n", "READY\nx", strings.Repeat("x", 64<<10)} {
		r := &inspectionResponder{ready: make(chan struct{}), stopped: make(chan struct{})}
		if _, err := r.Write([]byte(body)); err == nil {
			t.Fatal("unsolicited/malformed init frame accepted")
		}
		if r.position > len("READY\nSTOPPED\n") {
			t.Fatal("controller output grew beyond fixed frame")
		}
	}
}

func TestResponderControllerCancellationClosesPipe(t *testing.T) {
	model := &responderControllerModel{terminate: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	r, err := startResponderInit(ctx, model, strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	cleanup, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	// Closing the controller writer unblocks the peer even before STOP.
	_ = r.input.CloseWithError(context.Canceled)
	err = r.waitAfterTermination(cleanup, false)
	if err == nil {
		t.Fatal("cancellation supplied completion")
	}
}

func TestResponderSourceMetadataRefusesForeignCodeAndPins(t *testing.T) {
	for _, body := range [][]byte{nil, []byte("not an ELF"), make([]byte, responderSupplyCapacity)} {
		if validResponderImage(body) {
			t.Fatal("foreign or oversized helper accepted")
		}
	}
	// Project-owned HAA supply does not weaken any third-party expected pin.
	for _, kind := range []nativeRuntimeToolKind{nativeRuntimeUV, nativeRuntimePNPM, nativeRuntimeBun, nativeRuntimePipx} {
		spec, ok := nativeRuntimeToolIdentity(kind)
		if !ok || spec.size <= 0 || len(spec.sha256) != 64 {
			t.Fatal("third-party immutable identity lost")
		}
	}
}

func TestResponderBootstrapPreservesArtifactDemotion(t *testing.T) {
	command := responderContainerCommand()
	if !strings.Contains(command, "; GOMAXPROCS=1 exec ") {
		t.Fatal("init runtime CPU discovery was not bounded independently")
	}
	if !strings.Contains(command, "demote() { exec "+boundarySetprivPath+" "+boundaryDemotionArguments) ||
		!strings.HasSuffix(command, "--reuid=1001 --regid=1001 --clear-groups --inh-caps=-all --ambient-caps=-all --bounding-set=-all --no-new-privs -- "+responderGuestPath+" serve") {
		t.Fatal("responder changed Artifact demotion")
	}
	original := boundaryReadinessArguments(strings.Repeat("a", 64))
	current := boundaryReadinessArguments(strings.Repeat("a", 64), true)
	first, last, _ := strings.Cut(current[len(current)-1], "\nalready_demoted()")
	if !strings.Contains(first, "= 1001") || strings.Contains(first, "= 1000") || !strings.Contains(last, "= 1000") || original[len(original)-1] != boundaryReadinessScript {
		t.Fatal("init/Artifact credential validation crossed roles")
	}
}

func TestResponderControllerTrailingFrameFaultCannotBecomeCompletion(t *testing.T) {
	input, control := io.Pipe()
	defer input.Close()
	r := &inspectionResponder{input: control, cancel: func() {}, ready: make(chan struct{}), stopped: make(chan struct{}), done: make(chan struct{}), stopRequested: true}
	if _, err := io.WriteString(r, "READY\nSTOPPED\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(r, "x"); err == nil {
		t.Fatal("trailing frame accepted")
	}
	close(r.done)
	if r.waitAfterTermination(context.Background(), true) == nil {
		t.Fatal("parser failure became completion after resource termination")
	}
}
