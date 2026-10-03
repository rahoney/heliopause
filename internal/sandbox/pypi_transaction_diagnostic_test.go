package sandbox

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type terminationDiagnosticClient struct {
	lease ObservationResourceLease
	err   error
}

func (c terminationDiagnosticClient) Create(context.Context, string, string) (ObservationResourceLease, error) {
	return c.lease, c.err
}
func (c terminationDiagnosticClient) Register(context.Context, string, string, string) (ObservationResourceLease, error) {
	return c.lease, c.err
}
func (c terminationDiagnosticClient) Read(context.Context, string) (ObservationResourceLease, error) {
	return c.lease, c.err
}
func (c terminationDiagnosticClient) Terminate(context.Context, string, string) (ObservationResourceLease, error) {
	return c.lease, c.err
}
func (c terminationDiagnosticClient) Close(context.Context, string) (ObservationResourceLease, error) {
	return c.lease, c.err
}

type terminationDiagnosticTrace struct{ err error }

func (r terminationDiagnosticTrace) Next(context.Context) (TraceRecord, error) {
	return TraceRecord{}, r.err
}

func TestPythonTerminationDiagnosticAllRootPolicies(t *testing.T) {
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		t.Run(name, func(t *testing.T) {
			root, _ := artifactpypi.PyTorchProfile(name)
			ctx, err := artifactpypi.ContextWithResourcePolicy(context.Background(), root)
			if err != nil {
				t.Fatal(err)
			}
			profile, err := pythonDynamicObserverProfile(artifactpypi.RootSourceProfileNameFromContext(ctx))
			if err != nil {
				t.Fatal(err)
			}
			policy := artifactpypi.ResourcePolicyFromContext(ctx)
			for _, test := range []struct {
				name   string
				helper error
				trace  error
				parent string
				want   string
			}{
				{"complete", nil, io.EOF, "parent", ""},
				{"helper", context.DeadlineExceeded, io.EOF, "parent", "helper=DEADLINE_EXCEEDED"},
				{"trace fault", nil, observerFault{reason: "EVENT_LIMIT"}, "parent", "reason=EVENT_LIMIT"},
				{"incomplete trace", nil, context.DeadlineExceeded, "parent", "session_complete=false"},
				{"parent", nil, io.EOF, "substitute", "parent_match=false"},
			} {
				t.Run(test.name, func(t *testing.T) {
					id := strings.Repeat("a", 64)
					executor := &pythonTransactionExecutor{profile: profile, policy: policy, transaction: "sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa", started: time.Now(), requestCtx: ctx,
						lease: ObservationResourceLease{CgroupParent: "parent"}, resources: terminationDiagnosticClient{lease: ObservationResourceLease{CgroupParent: test.parent}, err: test.helper},
						runtimes: map[string]*pythonPhaseRuntime{id: {id: id, phase: phaseObservation, registered: true, trace: terminationDiagnosticTrace{err: test.trace}}}}
					err := executor.terminatePhase(id)
					if test.want == "" {
						if err != nil || len(executor.runtimes) != 0 {
							t.Fatalf("complete: %v", err)
						}
						return
					}
					if err == nil || !strings.Contains(err.Error(), test.want) || !strings.Contains(err.Error(), "profile="+profile) || len(executor.runtimes) != 1 {
						t.Fatalf("diagnostic/lifecycle lost: %v", err)
					}
				})
			}
		})
	}
}

func TestPythonCleanupPreservesPrimaryIncompleteResult(t *testing.T) {
	session, _ := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	for _, primary := range []string{"M5_PYPI_DYNAMIC_INSTALL_FAILED", "M5_PYPI_DYNAMIC_OBSERVATION_INCOMPLETE", ""} {
		var result domain.SandboxResult
		if primary == "" {
			result, _ = domain.NewSandboxResult(session, domain.SandboxCompleted, "", nil)
		} else {
			result, _ = pythonIncomplete(session, primary)
		}
		first := errors.New("first trusted failure")
		cleanup := errors.New("independent cleanup failure")
		actual, err := pythonCleanupFailure(session, result, first, cleanup)
		code, _ := actual.LimitationCode()
		want := primary
		if want == "" {
			want = "M5_PYPI_DYNAMIC_CLEANUP_FAILED"
		}
		if code != want || actual.Status() != domain.SandboxIncomplete || !errors.Is(err, first) || !errors.Is(err, cleanup) {
			t.Fatalf("primary=%s result=%s err=%v", primary, code, err)
		}
	}
}

func TestPythonResourceDiagnosticRedactsTransportPayload(t *testing.T) {
	if pythonResourceErrorReason(errors.New("/host/private/path credential=secret")) != "RESOURCE_ERROR" {
		t.Fatal("transport payload exposed")
	}
}

func TestPythonCommandDiagnosticOnlyFixedReasons(t *testing.T) {
	for _, c := range []struct {
		err  error
		want string
	}{
		{context.DeadlineExceeded, "DEADLINE_EXCEEDED"},
		{observerFault{reason: "LIFECYCLE_ERROR"}, "LIFECYCLE_ERROR"},
		{errors.New("credential secret"), "COMMAND_ERROR"},
		{errors.New("/host/private"), "COMMAND_ERROR"},
		{observerFault{reason: "SECRET"}, "COMMAND_ERROR"},
	} {
		if got := pythonCommandErrorReason(c.err); got != c.want {
			t.Fatalf("got %s want %s", got, c.want)
		}
	}
}

type onceFaultTrace struct{ reads int }

func (r *onceFaultTrace) Next(context.Context) (TraceRecord, error) {
	r.reads++
	if r.reads == 1 {
		return TraceRecord{Kind: "process-exec-expected", Bytes: 17}, nil
	}
	return TraceRecord{}, observerFault{reason: "STREAM_FAULT", site: "OPEN_RESULT"}
}
func TestPythonFailedPhaseEvidenceIsConsumedOnce(t *testing.T) {
	reader := &onceFaultTrace{}
	id := strings.Repeat("a", 64)
	tx := &pythonTransactionExecutor{started: time.Now(), resources: terminationDiagnosticClient{lease: ObservationResourceLease{CgroupParent: "parent"}}, lease: ObservationResourceLease{CgroupParent: "parent"}, runtimes: map[string]*pythonPhaseRuntime{id: {id: id, phase: phaseObservation, registered: true, trace: reader}}}
	for i := 0; i < 2; i++ {
		err := tx.terminatePhase(id)
		if err == nil || !strings.Contains(err.Error(), "events=1 bytes=17") || !strings.Contains(err.Error(), "fault_site=OPEN_RESULT") {
			t.Fatalf("attempt=%d err=%v", i, err)
		}
	}
	if reader.reads != 2 || len(tx.runtimes) != 1 {
		t.Fatalf("failed stream reconsumed or marked complete: reads=%d runtimes=%d", reader.reads, len(tx.runtimes))
	}
}
func TestObserverFaultSiteEnvelopeBoundaries(t *testing.T) {
	for _, test := range []struct {
		kind, site string
		valid      bool
	}{{"stream-fault", "OPEN_RESULT", true}, {"stream-fault", "OPEN_RESULT_CLASSIFICATION_PROCESS_NAME", true}, {"stream-fault", "/private/secret", false}, {"stream-end", "OPEN_RESULT", false}, {"stream-fault", "", true}} {
		payload := []byte(`{"container_id":"` + strings.Repeat("a", 64) + `","kind":"` + test.kind + `","reason":"STREAM_FAULT","fault_site":"` + test.site + `"}`)
		_, err := decodeHelperRecord(payload)
		if (err == nil) != test.valid {
			t.Fatalf("%+v err=%v", test, err)
		}
	}
}

type accountingFaultDrainClient struct {
	terminationDiagnosticClient
	terminated int
}

func (c *accountingFaultDrainClient) Read(context.Context, string) (ObservationResourceLease, error) {
	return ObservationResourceLease{}, errors.New("accounting unavailable")
}
func (c *accountingFaultDrainClient) Terminate(context.Context, string, string) (ObservationResourceLease, error) {
	c.terminated++
	return c.lease, nil
}
func TestPythonAccountingFailureStillDrainsExactPhase(t *testing.T) {
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		t.Run(name, func(t *testing.T) {
			root, _ := artifactpypi.PyTorchProfile(name)
			policy := root.ResourcePolicy()
			obsPolicy, err := observationPolicyForPython(policy, "profile", 1, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			client := &accountingFaultDrainClient{terminationDiagnosticClient: terminationDiagnosticClient{lease: ObservationResourceLease{CgroupParent: "parent"}}}
			id := strings.Repeat("a", 64)
			tx := &pythonTransactionExecutor{resources: client, lease: client.lease, runtimes: map[string]*pythonPhaseRuntime{id: {id: id, phase: phaseObservation, registered: true, trace: terminationDiagnosticTrace{err: io.EOF}}}, watch: &observationCPUWatch{client: client, ledger: &observationTransaction{policy: obsPolicy}}, started: time.Now()}
			err = tx.terminatePhase(id)
			if err == nil || !strings.Contains(err.Error(), "CPU counter is unavailable") || client.terminated != 1 || len(tx.runtimes) != 0 {
				t.Fatalf("accounting failure skipped drain or qualified: err=%v terminated=%d runtimes=%d", err, client.terminated, len(tx.runtimes))
			}
		})
	}
}

func TestPythonTerminationAccountingDoesNotRepairFailedUnit(t *testing.T) {
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		t.Run(name, func(t *testing.T) {
			root, _ := artifactpypi.PyTorchProfile(name)
			now := time.Now()
			policy, err := observationPolicyForPython(root.ResourcePolicy(), "profile", 1, now)
			if err != nil {
				t.Fatal(err)
			}
			for _, test := range []struct {
				name  string
				usage uint64
				at    time.Time
				valid bool
			}{{"nonqualifying unit", 11, now.Add(time.Millisecond), true},
				{"regressed counter", 9, now.Add(time.Millisecond), false},
				{"late counter", 11, now.Add(2 * policy.pollInterval), false},
				{"exhausted reserve", policy.cpuCeilingUsec, now.Add(time.Millisecond), false}} {
				t.Run(test.name, func(t *testing.T) {
					ledger, err := newObservationTransaction(policy, nil, now)
					if err != nil {
						t.Fatal(err)
					}
					if err := ledger.sampleCPU(10, now); err != nil {
						t.Fatal(err)
					}
					ledger.failed = true
					err = ledger.accountCPULocked(test.usage, test.at, true)
					if (err == nil) != test.valid || !ledger.failed {
						t.Fatalf("err=%v failed=%t", err, ledger.failed)
					}
					if ledger.sampleCPU(test.usage, test.at) == nil || ledger.finalize(test.usage, test.at) == nil {
						t.Fatal("termination sample restored qualification")
					}
				})
			}
			ledger, _ := newObservationTransaction(policy, nil, now)
			_ = ledger.sampleCPU(10, now)
			ledger.failed = true
			if err := ledger.accountCPUAfterTransition(11, now.Add(time.Millisecond), now.Add(2*time.Millisecond), true); err != nil || !ledger.failed {
				t.Fatalf("termination transition: %v", err)
			}
			if ledger.sampleCPUAfterTransition(12, now.Add(3*time.Millisecond), now.Add(4*time.Millisecond)) == nil {
				t.Fatal("ordinary transition repaired failed ledger")
			}
		})
	}
}

func TestObserverImageLocatorRemainsNonqualifyingDiagnostic(t *testing.T) {
	for _, test := range []struct {
		kind, site, value string
		valid             bool
	}{
		{"stream-fault", "OPEN_RESULT_CLASSIFICATION_IMAGE", "123", true},
		{"stream-fault", "OPEN_RESULT_CLASSIFICATION_IMAGE", "18446744073709551615", true},
		{"stream-fault", "OPEN_RESULT_CLASSIFICATION_IMAGE", "18446744073709551616", false},
		{"stream-fault", "OPEN_RESULT_CLASSIFICATION_IMAGE", "-1", false},
		{"stream-fault", "OPEN_RESULT_CLASSIFICATION_IMAGE", "\"secret\"", false},
		{"stream-fault", "OPEN_RESULT_CLASSIFICATION_PROC", "123", false},
		{"stream-end", "OPEN_RESULT_CLASSIFICATION_IMAGE", "123", false},
	} {
		payload := []byte(`{"container_id":"` + strings.Repeat("a", 64) + `","kind":"` + test.kind + `","reason":"STREAM_FAULT","fault_site":"` + test.site + `","fault_image_locator":` + test.value + `}`)
		_, err := decodeHelperRecord(payload)
		if (err == nil) != test.valid {
			t.Fatalf("%+v err=%v", test, err)
		}
	}
	_, limitation, diagnostic := collectTraceDiagnostic(context.Background(), terminationDiagnosticTrace{err: observerFault{reason: "STREAM_FAULT", site: "OPEN_RESULT_CLASSIFICATION_IMAGE", imageLocator: 123}})
	if limitation != "M3_DYNAMIC_OBSERVER_FAILED" || diagnostic.SessionComplete || !strings.Contains(diagnostic.String(), "image_locator_fnv1a64=000000000000007b") {
		t.Fatalf("locator gained authority or lost diagnostics: %s %+v", limitation, diagnostic)
	}
}

func TestArtifactCacheQueryDiagnosticDoesNotBecomeExpected(t *testing.T) {
	record := helperRecord{ContainerID: strings.Repeat("a", 64), Kind: "process-exec-unexpected", EventSource: "SENTRY_EXEC", ProcessClass: "OTHER", ClassificationReason: "ARTIFACT_LDCONFIG_QUERY", ParentRelation: "ARTIFACT_GROUP"}
	if !validAttribution(record) {
		t.Fatal("fixed diagnostic rejected")
	}
	record.Kind = "process-exec-expected"
	if validAttribution(record) {
		t.Fatal("diagnostic granted expected execution")
	}
	record.Kind = "trusted-control-network"
	if validAttribution(record) {
		t.Fatal("diagnostic granted CONTROL network")
	}
}
