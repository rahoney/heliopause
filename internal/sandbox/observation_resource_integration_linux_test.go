//go:build linux

package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLinuxPyPIWheelDynamicIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYPI_DYNAMIC_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux Python/gVisor dynamic integration")
	}
	root := t.TempDir()
	wheel := linuxDynamicWheel(t)
	path := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "wheel.whl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, wheel, 0o600); err != nil {
		t.Fatal(err)
	}
	// The dynamic introducer binds the in-container name to the verified intake
	// filename record, exactly as production intake does.
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "filename"), []byte("example-1.0-py3-none-any.whl"), 0o400); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(wheel)
	static, err := artifactpypi.InspectWheel(bytes.NewReader(wheel), int64(len(wheel)), "example-1.0-py3-none-any.whl", hex.EncodeToString(sum[:]), artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"}, artifactpypi.DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("pypi")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "wheel")
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:wheel", uint64(len(wheel)))
	if err != nil {
		t.Fatal(err)
	}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	runner := integrationRunner{t: t}
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewPythonDynamicBackend(runner, introducer, supervisor.Observer(), integrationPythonCapabilityProbe(runner))
	if err != nil {
		t.Fatal(err)
	}
	resources, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	backend.resources = integrationObservationResources{client: resources}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := backend.InspectWheel(ctx, artifact, static.ImportNames)
	if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("dynamic result = %#v observer_reason=%s, %v", result, integrationObserverFaultReason(supervisor), err)
	}
}

type integrationObservationResources struct {
	client *hosttool.ObservationResourceClient
}

func integrationObservationLease(lease hosttool.ObservationResourceLease) ObservationResourceLease {
	return ObservationResourceLease{CgroupParent: lease.CgroupParent, CPU: lease.CPU, UsageUsec: lease.UsageUsec}
}

func (a integrationObservationResources) Create(ctx context.Context, transaction, profile string) (ObservationResourceLease, error) {
	lease, err := a.client.Create(ctx, transaction, profile)
	return integrationObservationLease(lease), err
}
func (a integrationObservationResources) Register(ctx context.Context, transaction, container, role string) (ObservationResourceLease, error) {
	lease, err := a.client.Register(ctx, transaction, container, role)
	return integrationObservationLease(lease), err
}
func (a integrationObservationResources) Read(ctx context.Context, transaction string) (ObservationResourceLease, error) {
	lease, err := a.client.Read(ctx, transaction)
	return integrationObservationLease(lease), err
}
func (a integrationObservationResources) Terminate(ctx context.Context, transaction, container string) (ObservationResourceLease, error) {
	lease, err := a.client.Terminate(ctx, transaction, container)
	return integrationObservationLease(lease), err
}
func (a integrationObservationResources) Close(ctx context.Context, transaction string) (ObservationResourceLease, error) {
	lease, err := a.client.Close(ctx, transaction)
	return integrationObservationLease(lease), err
}

// This exercises the real typed helper, shared streams and transaction ledger
// under each root policy with a tiny authenticated fixture. It is not CUDA
// wheel qualification. CPU requests follow failed CUDA experiments in the same
// observer/backend so state leakage cannot hide behind separate processes.
// This runtime utility remains an actionable ARTIFACT exec; its minimal
// loader reads must not abort the observer in any Python root policy.
func TestLinuxPythonUtilityLoaderIntegration(t *testing.T) {
	runLinuxPythonRootPolicyTransactions(t, "utility-loader")
}

func TestLinuxPythonRootPolicyTransactionSequenceIntegration(t *testing.T) {
	runLinuxPythonRootPolicyTransactions(t, "sequence")
}

// CUDA torch reads this guest procfs identity in its native-loader startup.
// The pinned observer already declares that read applicable to Python. This
// tiny fixture diagnoses the producer/consumer contract without a CUDA closure.
func TestLinuxPythonProcMapsTransactionIntegration(t *testing.T) {
	runLinuxPythonRootPolicyTransactions(t, "proc-maps")
}

func TestLinuxPythonRenamedProcessTransactionIntegration(t *testing.T) {
	runLinuxPythonRootPolicyTransactions(t, "renamed")
}

func TestLinuxPythonPinnedLibutilTransactionIntegration(t *testing.T) {
	runLinuxPythonRootPolicyTransactions(t, "libutil")
}

// The bounded pinned stdlib query is an expected operation of an ARTIFACT
// child. It grants no CONTROL or completion authority.
func TestLinuxPythonLibraryDiscoveryBoundaryIntegration(t *testing.T) {
	for _, profile := range []string{"cpu", "cu126", "cu130", "cu132"} {
		t.Run(profile, func(t *testing.T) {
			t.Setenv("HELOX_PYTORCH_PROFILE", profile)
			runLinuxPythonRootPolicyTransactions(t, "library-discovery")
		})
	}
}

func TestLinuxPythonLibraryQuerySequenceIntegration(t *testing.T) {
	runLinuxPythonRootPolicyTransactions(t, "query-sequence")
}

// A nested implicit namespace must be selected without executing its parents
// during admission. Failed parent imports remain nonqualifying, including
// when a subsequent CPU transaction uses the same observer/backend.
func TestLinuxPythonCommandSequenceIntegration(t *testing.T) {
	runLinuxPythonRootPolicyTransactions(t, "command-sequence")
}

func TestLinuxPythonNamespaceSequenceIntegration(t *testing.T) {
	runLinuxPythonRootPolicyTransactions(t, "namespace-sequence")
}

// Real helper and shared observer, with prerequisite-first failure followed
// by an ordinary CPU request. The input is artifact code, never CONTROL.
func TestLinuxInspectionPrerequisiteSequenceIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYPI_PROFILE_TRANSACTION_INTEGRATION") != "1" {
		t.Skip("requires authenticated installed test client and pinned Linux bundle")
	}
	runner := integrationRunner{t: t}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	client, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewPythonDynamicBackend(runner, introducer, supervisor.Observer(), integrationPythonCapabilityProbe(runner))
	if err != nil {
		t.Fatal(err)
	}
	makeInput := func(project, program string) (domain.AcquiredArtifact, artifactpypi.ObservationPlan) {
		wheel := linuxDynamicProjectWheel(t, project, program, nil)
		run, _ := domain.NewRunID()
		dir := filepath.Join(root, run.String())
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		filename := project + "-1.0-py3-none-any.whl"
		for name, body := range map[string][]byte{"wheel.whl": wheel, "filename": []byte(filename)} {
			if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
				t.Fatal(err)
			}
		}
		sum := sha256.Sum256(wheel)
		hash := hex.EncodeToString(sum[:])
		source, _ := domain.NewSourceID("pypi")
		static, err := artifactpypi.InspectWheelForSource(bytes.NewReader(wheel), int64(len(wheel)), filename, hash, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"}, artifactpypi.DefaultWheelLimits(), source)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := artifactpypi.BuildObservationPlan(static, artifactpypi.PublicPyPIProfile().ResourcePolicy())
		if err != nil {
			t.Fatal(err)
		}
		identity, _ := domain.NewResolvedArtifactIdentity(source, project, "1.0", "wheel")
		digest, _ := domain.NewSHA256Digest(hash)
		a, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":wheel", uint64(len(wheel)), "sha256:"+hash)
		if err != nil {
			t.Fatal(err)
		}
		return a, plan
	}
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		for _, fail := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "-positive", true: "-input-exit37"}[fail], func(t *testing.T) {
				inputProgram := "VALUE=1\n"
				if fail {
					inputProgram = "raise SystemExit(37)\n"
				}
				input, _ := makeInput("support", inputProgram)
				target, plan := makeInput("example", "import support\n")
				profile, _ := artifactpypi.PyTorchProfile(name)
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				ctx, err = artifactpypi.ContextWithResourcePolicy(ctx, profile)
				if err != nil {
					t.Fatal(err)
				}
				resources := &prerequisiteIntegrationResources{ObservationResourceClient: integrationObservationResources{client: client}}
				backend.resources = resources
				result, err := backend.InspectWheelWithPrerequisites(ctx, target, plan, []domain.AcquiredArtifact{target}, []domain.AcquiredArtifact{input})
				wantUnits := 2
				if fail {
					wantUnits = 1
					if err == nil || result.Status() != domain.SandboxIncomplete || !strings.Contains(err.Error(), "command=EXIT_STATUS_37") || !strings.Contains(err.Error(), "unit_artifact_digest="+input.Digest().String()) {
						t.Fatalf("input failure hidden: %s %v", result.Status(), err)
					}
				} else if err != nil || result.Status() != domain.SandboxCompleted {
					t.Fatalf("normal input: %s %v", result.Status(), err)
				}
				if resources.creates != 1 || resources.closes != 1 || resources.units != wantUnits {
					t.Fatalf("reset/missing units: %+v", resources)
				}
				cpu, _ := artifactpypi.PyTorchProfile("cpu")
				cpuCtx, cpuCancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cpuCancel()
				cpuCtx, err = artifactpypi.ContextWithResourcePolicy(cpuCtx, cpu)
				if err != nil {
					t.Fatal(err)
				}
				ordinary, ordinaryPlan := makeInput("example", "VALUE=1\n")
				result, err = backend.InspectWheelWithPlan(cpuCtx, ordinary, ordinaryPlan, []domain.AcquiredArtifact{ordinary})
				if err != nil || result.Status() != domain.SandboxCompleted {
					t.Fatalf("following CPU contaminated: %s %v", result.Status(), err)
				}
				t.Logf("profile=%s input_failure=%t own_units=%d creates=1 closes=1 next_cpu=%s", name, fail, wantUnits, result.Status())
			})
		}
	}
}

type prerequisiteIntegrationResources struct {
	ObservationResourceClient
	creates, closes, units int
}

func (r *prerequisiteIntegrationResources) Create(ctx context.Context, transaction, profile string) (ObservationResourceLease, error) {
	r.creates++
	return r.ObservationResourceClient.Create(ctx, transaction, profile)
}
func (r *prerequisiteIntegrationResources) Register(ctx context.Context, transaction, container, role string) (ObservationResourceLease, error) {
	if role == string(phaseObservation) {
		r.units++
	}
	return r.ObservationResourceClient.Register(ctx, transaction, container, role)
}
func (r *prerequisiteIntegrationResources) Close(ctx context.Context, transaction string) (ObservationResourceLease, error) {
	r.closes++
	return r.ObservationResourceClient.Close(ctx, transaction)
}

func TestLinuxPythonRejectedLibraryQueryIntegration(t *testing.T) {
	for _, profile := range []string{"cpu", "cu126", "cu130", "cu132"} {
		for _, mode := range []string{"query-negative-args", "query-negative-env", "query-negative-image"} {
			t.Run(profile+"/"+mode, func(t *testing.T) {
				t.Setenv("HELOX_PYTORCH_PROFILE", profile)
				runLinuxPythonRootPolicyTransactions(t, mode)
			})
		}
	}
}

func runLinuxPythonRootPolicyTransactions(t *testing.T, mode string) {
	if os.Getenv("HELOX_PYPI_PROFILE_TRANSACTION_INTEGRATION") != "1" {
		t.Skip("requires authenticated installed test client and pinned Linux bundle")
	}
	runner := integrationRunner{t: t}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	client, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewPythonDynamicBackend(runner, introducer, supervisor.Observer(), integrationPythonCapabilityProbe(runner))
	if err != nil {
		t.Fatal(err)
	}
	backend.resources = integrationObservationResources{client: client}
	tests := []struct {
		profile string
		fail    bool
	}{{"cpu", false}}
	for _, name := range []string{"cu126", "cu130", "cu132"} {
		tests = append(tests, struct {
			profile string
			fail    bool
		}{name, false}, struct {
			profile string
			fail    bool
		}{name, true}, struct {
			profile string
			fail    bool
		}{"cpu", false})
	}
	if mode != "sequence" && mode != "query-sequence" && mode != "namespace-sequence" && mode != "command-sequence" {
		if mode != "proc-maps" && mode != "library-discovery" && !strings.HasPrefix(mode, "query-negative-") {
			tests = tests[:0]
			for _, profile := range []string{"cpu", "cu126", "cu130", "cu132"} {
				tests = append(tests, struct {
					profile string
					fail    bool
				}{profile, false})
			}
		} else {
			profile := os.Getenv("HELOX_PYTORCH_PROFILE")
			if _, ok := artifactpypi.PyTorchProfile(profile); !ok {
				t.Fatal("invalid diagnostic root profile")
			}
			tests = tests[:1]
			tests[0].profile = profile
		}
	}
	for _, test := range tests {
		t.Run(test.profile+map[bool]string{true: "-nonzero", false: "-zero"}[test.fail], func(t *testing.T) {
			program := "VALUE = 'ok'\n"
			switch mode {
			case "sequence", "namespace-sequence", "command-sequence":
			case "utility-loader":
				program = "import subprocess\nsubprocess.run(['/usr/bin/uname', '-p'], check=True)\n"
			case "proc-maps":
				program = "with open('/proc/self/maps') as maps:\n    maps.read(4096)\n"
			case "renamed":
				program = "import ctypes\nif ctypes.CDLL(None).prctl(15, b'worker', 0, 0, 0) != 0:\n    raise RuntimeError('process name change failed')\nwith open('/proc/self/maps') as maps:\n    maps.read(4096)\n"
			case "libutil":
				program = "import ctypes\nctypes.CDLL('libutil.so.1')\n"
			case "library-discovery", "query-sequence":
				program = "import ctypes.util\nif ctypes.util.find_library('dl') != 'libdl.so.2':\n    raise RuntimeError('library lookup failed')\n"
			case "query-negative-args":
				program = "import subprocess\nsubprocess.run(['/sbin/ldconfig', '-p', '-N'], env={'LC_ALL':'C', 'LANG':'C'}, stdout=subprocess.PIPE, check=True)\n"
			case "query-negative-env":
				program = "import subprocess\nsubprocess.run(['/sbin/ldconfig', '-p'], env={'LC_ALL':'C', 'LANG':'C', 'LD_PRELOAD':'/tmp/fake.so'}, stdout=subprocess.PIPE, check=True)\n"
			case "query-negative-image":
				program = "import subprocess\nsubprocess.run(['/sbin/ldconfig', '-p'], executable='/bin/true', env={'LC_ALL':'C', 'LANG':'C'}, stdout=subprocess.PIPE, check=True)\n"
			default:
				t.Fatal("unknown diagnostic fixture")
			}
			if test.fail {
				if mode == "query-sequence" {
					program += "raise SystemExit(37)\n"
				} else {
					program = "raise SystemExit(37)\n"
				}
			}
			wheel := linuxDynamicWheelWithProgram(t, program)
			if mode == "command-sequence" {
				wheel = linuxDynamicWheelWithProgramAndFiles(t, "VALUE = 'base'\n", map[string][]byte{
					"example/cli.py":                         []byte(program),
					"example-1.0.dist-info/entry_points.txt": []byte("[console_scripts]\ncommand = example.cli:main\n"),
				})
			}
			if mode == "namespace-sequence" {
				wheel = linuxDynamicWheelWithProgramAndFiles(t, program, map[string][]byte{
					"example/lib/assets/opaque.payload":      []byte("untrusted resource\n"),
					"example-1.0.dist-info/entry_points.txt": []byte("[example.locations]\nassets = example.lib.assets\n"),
				})
			}
			run, err := domain.NewRunID()
			if err != nil {
				t.Fatal(err)
			}
			directory := filepath.Join(root, run.String())
			if err := os.Mkdir(directory, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "wheel.whl"), wheel, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "filename"), []byte("example-1.0-py3-none-any.whl"), 0400); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(wheel)
			hash := hex.EncodeToString(sum[:])
			profile, _ := artifactpypi.PyTorchProfile(test.profile)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			ctx, err = artifactpypi.ContextWithResourcePolicy(ctx, profile)
			if err != nil {
				t.Fatal(err)
			}
			source, _ := domain.NewSourceID("pypi")
			static, err := artifactpypi.InspectWheelForSource(bytes.NewReader(wheel), int64(len(wheel)), "example-1.0-py3-none-any.whl", hash, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"}, profile.ResourcePolicy().WheelLimits(), source)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := artifactpypi.BuildObservationPlan(static, profile.ResourcePolicy())
			if err != nil {
				t.Fatal(err)
			}
			if mode == "namespace-sequence" && (!plan.Admissible() || len(plan.Units) != 2 || plan.Units[1].Candidate != "example.lib.assets") {
				t.Fatalf("exact namespace unit missing: %+v", plan)
			}
			identity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "wheel")
			digest, _ := domain.NewSHA256Digest(hash)
			artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+run.String()+":wheel", uint64(len(wheel)))
			if err != nil {
				t.Fatal(err)
			}
			result, err := backend.InspectWheelWithPlan(ctx, artifact, plan, []domain.AcquiredArtifact{artifact})
			if mode == "utility-loader" {
				unexpected := false
				for _, observation := range result.Observations() {
					if observation.Subject() == "process-exec-unexpected" {
						unexpected = true
					}
				}
				if err != nil || result.Status() != domain.SandboxCompleted || !unexpected {
					t.Fatalf("utility loader lost completed observation/actionable exec: status=%s err=%v observations=%v", result.Status(), err, result.Observations())
				}
			} else if strings.HasPrefix(mode, "query-negative-") {
				unexpected := false
				for _, observation := range result.Observations() {
					if observation.Subject() == "process-exec-unexpected" {
						unexpected = true
					}
				}
				if !unexpected && (err == nil || !strings.Contains(err.Error(), "process-exec-unexpected:1")) {
					t.Fatalf("unsupported query lost actionable exec: status=%s err=%v observations=%v", result.Status(), err, result.Observations())
				}
			} else if mode == "library-discovery" {
				if err != nil || result.Status() != domain.SandboxCompleted {
					t.Fatalf("bounded library query: status=%s err=%v", result.Status(), err)
				}
				if reason := integrationObserverFaultReason(supervisor); reason != "" {
					t.Fatalf("library query observer fault: %s", reason)
				}
			} else if mode == "command-sequence" {
				if err != nil || result.Status() != domain.SandboxCompleted || len(result.Commands) != 1 || result.Commands[0].Module != "example.cli" || result.Commands[0].ZeroExit == test.fail {
					t.Fatalf("command sequence: status=%s commands=%+v err=%v", result.Status(), result.Commands, err)
				}
			} else if test.fail {
				code, _ := result.LimitationCode()
				if err == nil || !strings.Contains(err.Error(), "stage=unit-outcome") || !strings.Contains(err.Error(), "command=EXIT_STATUS_37") || strings.Contains(err.Error(), "python transaction cleanup") || result.Status() != domain.SandboxIncomplete || code != "M5_PYPI_DYNAMIC_OBSERVATION_INCOMPLETE" {
					t.Fatalf("failed unit gained completion or lost code: status=%s code=%s err=%v", result.Status(), code, err)
				}
			} else if err != nil || result.Status() != domain.SandboxCompleted {
				t.Fatalf("normal transaction: status=%s err=%v", result.Status(), err)
			}
			if reason := integrationObserverFaultReason(supervisor); mode != "library-discovery" && !strings.HasPrefix(mode, "query-negative-") && reason != "" {
				t.Fatalf("observer fault leaked across transactions: %s", reason)
			}
			t.Logf("root=%s fixture_sha256=%s units=%d status=%s transaction=%s", profile.Name(), hash, len(plan.Units), result.Status(), result.SessionID().String())
		})
	}
}

// This opt-in replay consumes retained production-intake bytes, revalidates
// every identity/digest and invokes the normal transaction entry point. It is
// a focused diagnosis, not full install/promotion qualification or resolver replay.
func TestLinuxRetainedPythonTransactionIntegration(t *testing.T) {
	if os.Getenv("HELOX_RETAINED_TRANSACTION_INTEGRATION") != "1" {
		t.Skip("requires authenticated retained intake and installed test client")
	}
	profile, ok := artifactpypi.PyTorchProfile(os.Getenv("HELOX_PYTORCH_PROFILE"))
	if !ok {
		t.Fatal("invalid root profile")
	}
	data, err := os.ReadFile(os.Getenv("HELOX_RETAINED_INTAKE_MANIFEST"))
	if err != nil {
		t.Fatal(err)
	}
	var nodes []struct {
		Source, Project, Version, Integrity string
		LocalBytes                          string `json:"local_bytes"`
	}
	if err := json.Unmarshal(data, &nodes); err != nil || len(nodes) == 0 {
		t.Fatalf("invalid retained manifest: %v", err)
	}
	root := os.Getenv("HELOX_RETAINED_INTAKE_ROOT")
	nativeProbe := os.Getenv("HELOX_RETAINED_NATIVE_PROBE") == "1"
	if nativeProbe {
		root = t.TempDir()
	}
	nativePath := ""
	nativeMode := os.Getenv("HELOX_RETAINED_NATIVE_MODE")
	if nativeProbe && nativeMode != "" && nativeMode != "torch-global-deps" {
		t.Fatal("invalid native probe mode")
	}
	var closure []domain.AcquiredArtifact
	var prerequisites []domain.AcquiredArtifact
	prerequisiteUnits := 0
	var target domain.AcquiredArtifact
	var plan artifactpypi.ObservationPlan
	for _, n := range nodes {
		relative, err := filepath.Rel(os.Getenv("HELOX_RETAINED_INTAKE_ROOT"), n.LocalBytes)
		if err != nil || filepath.IsAbs(relative) || strings.HasPrefix(relative, "..") {
			t.Fatal("retained path outside intake")
		}
		run, err := domain.ParseRunID(filepath.Base(filepath.Dir(n.LocalBytes)))
		if err != nil || filepath.Base(n.LocalBytes) != "wheel.whl" {
			t.Fatal("invalid intake handle")
		}
		filename, err := os.ReadFile(filepath.Join(filepath.Dir(n.LocalBytes), "filename"))
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(n.LocalBytes)
		if err != nil {
			t.Fatal(err)
		}
		info, err := f.Stat()
		if err != nil {
			f.Close()
			t.Fatal(err)
		}
		source, err := domain.NewSourceID(n.Source)
		if err != nil {
			f.Close()
			t.Fatal(err)
		}
		hash := strings.TrimPrefix(n.Integrity, "sha256:")
		inspected, err := artifactpypi.InspectWheelForSource(f, info.Size(), string(filename), hash, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"}, profile.ResourcePolicy().WheelLimits(), source)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if inspected.Project != n.Project || inspected.Version != n.Version {
			t.Fatal("retained identity mismatch")
		}
		identity, err := domain.NewResolvedArtifactIdentity(source, n.Project, n.Version, "wheel")
		if err != nil {
			t.Fatal(err)
		}
		digest, err := domain.NewSHA256Digest(hash)
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":wheel", uint64(info.Size()), n.Integrity)
		if err != nil {
			t.Fatal(err)
		}
		if n.Project == os.Getenv("HELOX_RETAINED_PREREQUISITE_TARGET") {
			prerequisites = append(prerequisites, artifact)
			inputPlan, err := artifactpypi.BuildObservationPlan(inspected, profile.ResourcePolicy())
			if err != nil || !inputPlan.Admissible() {
				t.Fatalf("input plan: %v", err)
			}
			prerequisiteUnits += len(inputPlan.Units)
		} else {
			closure = append(closure, artifact)
		}
		if nativeProbe {
			directory := filepath.Join(root, run.String())
			if err := os.Mkdir(directory, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Link(n.LocalBytes, filepath.Join(directory, "wheel.whl")); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "filename"), filename, 0400); err != nil {
				t.Fatal(err)
			}
			for _, file := range inspected.Files {
				if (nativeMode == "" && strings.HasPrefix(filepath.Base(file.Path), "libcublasLt.so.")) || (nativeMode == "torch-global-deps" && filepath.Base(file.Path) == "libtorch_global_deps.so") {
					nativePath = file.Path
				}
			}
		}
		if n.Project == os.Getenv("HELOX_RETAINED_TARGET") {
			target = artifact
			plan, err = artifactpypi.BuildObservationPlan(inspected, profile.ResourcePolicy())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if nativeProbe {
		if (nativeMode == "" && len(nodes) != 1) || (nativeMode == "torch-global-deps" && len(nodes) != 2) || nativePath == "" {
			t.Fatal("native diagnostic does not match the bounded authenticated input set")
		}
		wheel := linuxDynamicWheelWithProgram(t, "import ctypes\nctypes.CDLL("+pythonLiteral("/haa-site/"+nativePath)+")\n")
		run, err := domain.NewRunID()
		if err != nil {
			t.Fatal(err)
		}
		directory := filepath.Join(root, run.String())
		if err := os.Mkdir(directory, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "wheel.whl"), wheel, 0600); err != nil {
			t.Fatal(err)
		}
		const filename = "example-1.0-py3-none-any.whl"
		if err := os.WriteFile(filepath.Join(directory, "filename"), []byte(filename), 0400); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(wheel)
		hash := hex.EncodeToString(sum[:])
		source, _ := domain.NewSourceID("pypi")
		inspected, err := artifactpypi.InspectWheelForSource(bytes.NewReader(wheel), int64(len(wheel)), filename, hash, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"}, profile.ResourcePolicy().WheelLimits(), source)
		if err != nil {
			t.Fatal(err)
		}
		plan, err = artifactpypi.BuildObservationPlan(inspected, profile.ResourcePolicy())
		if err != nil {
			t.Fatal(err)
		}
		identity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "wheel")
		digest, _ := domain.NewSHA256Digest(hash)
		target, err = domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":wheel", uint64(len(wheel)), "sha256:"+hash)
		if err != nil {
			t.Fatal(err)
		}
		closure = append(closure, target)
	}
	if target.Identity().Name() == "" {
		t.Fatal("target missing")
	}
	runner := integrationRunner{t: t}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewPythonDynamicBackend(runner, introducer, supervisor.Observer(), integrationPythonCapabilityProbe(runner))
	if err != nil {
		t.Fatal(err)
	}
	client, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	resources := &prerequisiteIntegrationResources{ObservationResourceClient: integrationObservationResources{client: client}}
	backend.resources = resources
	ctx, cancel := context.WithTimeout(context.Background(), profile.ResourcePolicy().Duration())
	defer cancel()
	ctx, err = artifactpypi.ContextWithResourcePolicy(ctx, profile)
	if err != nil {
		t.Fatal(err)
	}
	var result PythonObservationResult
	if len(prerequisites) != 0 {
		result, err = backend.InspectWheelWithPrerequisites(ctx, target, plan, closure, prerequisites)
	} else {
		result, err = backend.InspectWheelWithPlan(ctx, target, plan, closure)
	}
	for _, command := range result.Commands {
		t.Logf("command observation: %+v", command)
	}
	if expected := os.Getenv("HELOX_RETAINED_NOT_ATTESTED_MODULE"); expected != "" {
		summary, summaryErr := result.ObservationSummary()
		if summaryErr != nil {
			t.Fatal(summaryErr)
		}
		t.Logf("command transaction normalized observations: %s", summary)
		for _, obs := range result.Observations() {
			switch obs.Category() {
			case domain.ObservationNetwork, domain.ObservationHoneytoken, domain.ObservationResource:
				t.Fatalf("command fixture produced actionable observation: %s %s", obs.Category(), obs.Subject())
			}
			switch obs.Subject() {
			case "filesystem-violation", "filesystem-outside-workspace", "process-unexpected", "process-exec-unexpected":
				t.Fatalf("command fixture produced actionable observation: %s", obs.Subject())
			}
		}
		matches := 0
		for _, command := range result.Commands {
			if command.Module == expected {
				matches++
				if command.ZeroExit || command.OwnerSHA256 != target.Digest().String() {
					t.Fatalf("command observation mismatch: %+v", command)
				}
			}
		}
		if matches != 1 || len(prerequisites) != 0 {
			t.Fatalf("command observation mismatch: %+v", result.Commands)
		}
	}
	code, _ := result.LimitationCode()
	t.Logf("root=%s artifact=%s digest=%s closure=%d prerequisites=%d target_units=%d input_units=%d executed_units=%d authorizations=%d closes=%d status=%s code=%s", profile.Name(), target.Identity().Name(), target.Digest().String(), len(closure), len(prerequisites), len(plan.Units), prerequisiteUnits, resources.units, resources.creates, resources.closes, result.Status(), code)
	if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("retained transaction: %v", err)
	}
	if resources.creates != 1 || resources.closes != 1 || resources.units != len(plan.Units)+prerequisiteUnits {
		t.Fatal("shared authorization/unit reconciliation failed")
	}
}
