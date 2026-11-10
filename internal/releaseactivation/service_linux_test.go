//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func serviceStateFixture() string {
	return "Id=" + policyUnitName + "\nLoadState=loaded\nFragmentPath=" + policyUnitPath + "\nDropInPaths=\nNeedDaemonReload=no\nUser=root\nGroup=root\nActiveState=active\nSubState=running\nMainPID=123\nControlPID=0\nControlGroup=" + policyControlGroup + "\nResult=success\n"
}

type serviceToolFixture struct {
	body       string
	failure    error
	discardErr error
	calls      []string
	closeErr   error
}

func (f *serviceToolFixture) RunOutput(_ context.Context, writer io.Writer, name string, args ...string) error {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	if f.failure != nil {
		return f.failure
	}
	_, err := io.WriteString(writer, f.body)
	return err
}
func (f *serviceToolFixture) RunDiscard(_ context.Context, name string, args ...string) error {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	if f.discardErr != nil {
		return f.discardErr
	}
	return f.failure
}
func (f *serviceToolFixture) Close() error { return f.closeErr }

func TestServiceQueryRejectsUnknownAndIncompleteFacts(t *testing.T) {
	valid := serviceStateFixture()
	state, err := parseServiceState([]byte(valid))
	if err != nil || state.pid != 123 {
		t.Fatal(state, err)
	}
	for name, body := range map[string]string{
		"missing":          strings.ReplaceAll(valid, "ControlPID=0\n", ""),
		"duplicate":        valid + "MainPID=123\n",
		"foreign-property": valid + "Command=/tmp/other\n",
		"foreign-unit":     strings.ReplaceAll(valid, "Id="+policyUnitName, "Id=other.service"),
		"foreign-path":     strings.ReplaceAll(valid, "FragmentPath="+policyUnitPath, "FragmentPath=/tmp/other.service"),
		"dropin":           strings.ReplaceAll(valid, "DropInPaths=\n", "DropInPaths=/tmp/override.conf\n"),
		"reload":           strings.ReplaceAll(valid, "NeedDaemonReload=no", "NeedDaemonReload=yes"),
		"uid":              strings.ReplaceAll(valid, "User=root", "User=other"),
		"control":          strings.ReplaceAll(valid, "ControlPID=0", "ControlPID=5"),
		"pid":              strings.ReplaceAll(valid, "MainPID=123", "MainPID=0123"),
		"limit":            strings.Repeat("x", 16<<10+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseServiceState([]byte(body)); err == nil {
				t.Fatal("unknown service facts approved")
			}
		})
	}
	tools := &serviceToolFixture{body: valid}
	controller := &serviceController{tools}
	if _, err := controller.query(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(tools.calls) != 1 || tools.calls[0] != "systemctl show --no-pager --property="+servicePropertyList+" -- "+policyUnitName {
		t.Fatal("query did not select fixed unit", tools.calls)
	}
	tools.body = strings.Repeat("x", 16<<10+1)
	if _, err := controller.query(context.Background()); !errors.Is(err, ErrTransition) {
		t.Fatal("query truncation became absence", err)
	}
	failure := errors.New("kernel/manager query unavailable")
	tools.failure = failure
	if _, err := controller.query(context.Background()); !errors.Is(err, failure) {
		t.Fatal("primary query cause lost")
	}
	if err := controller.command(context.Background(), "stop", "--", policyUnitName); !errors.Is(err, failure) {
		t.Fatal("primary stop cause lost")
	}
	if err := controller.stopped(context.Background(), nil); !errors.Is(err, failure) {
		t.Fatal("failed query became confirmed stop")
	}
	tools.closeErr = failure
	if !errors.Is(controller.Close(), failure) {
		t.Fatal("tool cleanup cause lost")
	}
}

func TestServiceMachineOutputCannotBypassByteBudget(t *testing.T) {
	tools := &serviceToolFixture{body: "oversized machine record"}
	controller := &serviceController{tools}
	if body, err := controller.output(context.Background(), 4, "systemctl", "show"); err == nil || len(body) != 0 {
		t.Fatal("machine output exceeded its explicit byte budget")
	}
}

func TestServiceAndDrainRejectMissingAuthorityWithoutHostSetup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newServiceController(ctx); err == nil {
		t.Fatal("canceled root setup admitted")
	}
	controller := &serviceController{&serviceToolFixture{}}
	if _, err := controller.active(ctx, "v1.0.0", 1000); !errors.Is(err, context.Canceled) {
		t.Fatal("active cancellation cause lost", err)
	}
	if _, err := controller.start(ctx, "v1.0.0", 1000); !errors.Is(err, context.Canceled) {
		t.Fatal("start cancellation cause lost", err)
	}
	if err := controller.stop(context.Background(), nil); err == nil {
		t.Fatal("unowned process stopped")
	}
	if err := controller.drained(context.Background(), nil, nil); err == nil {
		t.Fatal("unowned drain approved")
	}
	if _, err := pinServiceProcess(0, "v1.0.0", 1000); err == nil {
		t.Fatal("unknown process pinned")
	}
	if err := protectedConsumersAbsent(ctx, 1, 1); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled scan lost cause")
	}
	for name, body := range map[string]string{"root": "Uid:\t0\t0\t0\t0\nGid:\t0\t0\t0\t0\n", "missing": "Uid:\t0\t0\t0\t0\n", "mixed": "Uid:\t0\t1000\t0\t0\nGid:\t0\t0\t0\t0\n", "duplicate": "Uid:\t0\t0\t0\t0\nUid:\t0\t0\t0\t0\nGid:\t0\t0\t0\t0\n"} {
		if rootKernelCredentials([]byte(body)) != (name == "root") {
			t.Fatal("root kernel credential parser differs", name)
		}
	}
	for _, body := range []string{"", "unrelated\n"} {
		if err := noHAAContainers([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	for _, body := range []string{"heliopause-job\n", "unrelated\nheliopause-job\n", "malformed line\n", "bad\x00name"} {
		if err := noHAAContainers([]byte(body)); err == nil {
			t.Fatal("busy or unknown container query admitted")
		}
	}
	for _, body := range []string{"State:\tZ (zombie)\n", "Kthread:\t1\n"} {
		if !kernelProcessWithoutExecutable([]byte(body)) {
			t.Fatal("kernel terminal facts lost")
		}
	}
	for _, body := range []string{"State:\tR (running)\nKthread:\t0\n", "", "Kthread:\t1\nKthread:\t0\n"} {
		if kernelProcessWithoutExecutable([]byte(body)) {
			t.Fatal("missing executable became confirmed absence")
		}
	}
	for _, path := range []string{"/usr/lib/heliopause/versions/v1.0.0/helox-linux-amd64", "/usr/libexec/heliopause/haa_gvisor_observer (deleted)"} {
		if !protectedConsumerPath(path) {
			t.Fatal("protected consumer ignored")
		}
	}
	for _, path := range []string{"/usr/bin/unrelated", "/tmp/heliopause/helox", "/usr/lib/heliopause-other/versions/helox"} {
		if protectedConsumerPath(path) {
			t.Fatal("unrelated consumer selected")
		}
	}
}
