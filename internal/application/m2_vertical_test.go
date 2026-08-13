package application_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	npmartifact "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence/local"
	npminspection "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/policy"
	npmverification "github.com/rahoney/heliopause/internal/verification/npm"
)

func TestM3ControlledVerticalWorkflow(t *testing.T) {
	tests := []struct {
		name         string
		observations []domain.SandboxObservation
		status       domain.SandboxStatus
		limitation   string
		want         domain.Decision
		reason       string
	}{
		{"clean allow", nil, domain.SandboxCompleted, "", domain.DecisionAllow, "M3_REQUIRED_CHECKS_COMPLETED"},
		{"network review", []domain.SandboxObservation{m3Observation(t, domain.ObservationNetwork, "network-attempt")}, domain.SandboxCompleted, "", domain.DecisionManualReview, "M3_NETWORK_ATTEMPT"},
		{"honeytoken block", []domain.SandboxObservation{m3Observation(t, domain.ObservationHoneytoken, "honeytoken-access")}, domain.SandboxCompleted, "", domain.DecisionBlock, "M3_HONEYTOKEN_ACCESS"},
		{"unavailable review", nil, domain.SandboxIncomplete, "M3_LINUX_ONLY", domain.DecisionManualReview, "M3_REQUIRED_CHECK_INCOMPLETE"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := canonicalRoot(t)
			registryURL := "https://registry.test/"
			tarball := archiveBytes(t, []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}})
			observed := sha512.Sum512(tarball)
			integrity := "sha512-" + base64.StdEncoding.EncodeToString(observed[:])
			transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.EscapedPath() == "/tiny" {
					return response(http.StatusOK, "application/json", metadataBody(registryURL, integrity)), nil
				}
				return response(http.StatusOK, "application/octet-stream", string(tarball)), nil
			})
			resolver, _ := npmartifact.NewResolver(registryURL, &http.Client{Transport: transport}, filepath.Join(root, "intake"))
			static, _ := npminspection.NewInspector(filepath.Join(root, "intake"))
			session, _ := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
			result, _ := domain.NewSandboxResult(session, test.status, test.limitation, test.observations)
			dynamic, _ := npminspection.NewDynamicInspector(m3Sandbox{result: result})
			inspector, _ := npminspection.NewCompositeInspector(static, dynamic)
			store, _ := local.NewStore(filepath.Join(root, "evidence"))
			service, _ := application.NewInspectService(resolver, npmverification.IntegrityVerifier{}, inspector, store, policy.M3{}, fixedOperationID, fixedRunID)
			reference, _ := npmartifact.ParseReference("tiny@1.2.3")
			request, _ := application.NewInspectRequest(reference)
			operation, err := service.Inspect(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			decision, ok := operation.PolicyDecision()
			if !ok || decision.Decision() != test.want {
				t.Fatalf("decision = %#v", decision)
			}
			if reasons := decision.Reasons(); len(reasons) != 1 || reasons[0] != test.reason {
				t.Fatalf("reasons = %#v", reasons)
			}
		})
	}
}

type m3Sandbox struct{ result domain.SandboxResult }

func (m m3Sandbox) Execute(context.Context, domain.SandboxRequest) (domain.SandboxResult, error) {
	return m.result, nil
}

var _ ports.Sandbox = m3Sandbox{}

func m3Observation(t *testing.T, category domain.ObservationCategory, subject string) domain.SandboxObservation {
	t.Helper()
	observation, err := domain.NewSandboxObservation(category, subject)
	if err != nil {
		t.Fatal(err)
	}
	return observation
}

func TestM2ControlledVerticalWorkflow(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		entries   []tarEntry
		integrity string
		decision  domain.Decision
		reason    string
	}{
		{name: "safe static requires review", entries: []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}}, decision: domain.DecisionManualReview, reason: "M2_DYNAMIC_INSPECTION_UNAVAILABLE"},
		{name: "integrity mismatch blocks", entries: []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}}, integrity: "sha512-AAAA", decision: domain.DecisionBlock, reason: "M2_INTEGRITY_MISMATCH"},
		{name: "unsafe archive blocks", entries: []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}, {name: "package/link", typeflag: tar.TypeSymlink}}, decision: domain.DecisionBlock, reason: "M2_ARCHIVE_TYPE_INVALID"},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := canonicalRoot(t)
			registryURL := "https://registry.test/"
			tarball := archiveBytes(t, test.entries)
			observed := sha512.Sum512(tarball)
			declared := test.integrity
			if declared == "" {
				declared = "sha512-" + base64.StdEncoding.EncodeToString(observed[:])
			}
			transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
				switch request.URL.EscapedPath() {
				case "/tiny":
					return response(http.StatusOK, "application/json", metadataBody(registryURL, declared)), nil
				case "/tiny/-/tiny-1.2.3.tgz":
					return response(http.StatusOK, "application/octet-stream", string(tarball)), nil
				default:
					return response(http.StatusNotFound, "text/plain", "missing"), nil
				}
			})
			resolver, err := npmartifact.NewResolver(registryURL, &http.Client{Transport: transport}, filepath.Join(root, "intake"))
			if err != nil {
				t.Fatal(err)
			}
			inspector, err := npminspection.NewInspector(filepath.Join(root, "intake"))
			if err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(filepath.Join(root, "evidence"))
			if err != nil {
				t.Fatal(err)
			}
			service, err := application.NewInspectService(resolver, npmverification.IntegrityVerifier{}, inspector, store, policy.M2{}, fixedOperationID, fixedRunID)
			if err != nil {
				t.Fatal(err)
			}
			reference, err := npmartifact.ParseReference("tiny@1.2.3")
			if err != nil {
				t.Fatal(err)
			}
			request, err := application.NewInspectRequest(reference)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Inspect(context.Background(), request)
			if err != nil {
				t.Fatalf("Inspect() error = %v", err)
			}
			decision, ok := result.PolicyDecision()
			if !ok || decision.Decision() != test.decision {
				t.Fatalf("Policy = %#v, %v", decision, ok)
			}
			if reasons := decision.Reasons(); len(reasons) != 1 || reasons[0] != test.reason {
				t.Fatalf("Reasons = %#v", reasons)
			}
			if result.Status() != domain.OperationCompleted {
				t.Fatalf("result status = %s", result.Status())
			}
			if outcome, ok := result.RunOutcome(); !ok || outcome != domain.RunCompleted {
				t.Fatalf("run outcome = %s, %v", outcome, ok)
			}
		})
	}
}

func TestM2ControlledVerticalResolveFailureIsOperational(t *testing.T) {
	t.Parallel()
	root := canonicalRoot(t)
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return response(http.StatusServiceUnavailable, "application/json", "{}"), nil
	})
	resolver, err := npmartifact.NewResolver("https://registry.test/", &http.Client{Transport: transport}, filepath.Join(root, "intake"))
	if err != nil {
		t.Fatal(err)
	}
	inspector, err := npminspection.NewInspector(filepath.Join(root, "intake"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := local.NewStore(filepath.Join(root, "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewInspectService(resolver, npmverification.IntegrityVerifier{}, inspector, store, policy.M2{}, fixedOperationID, fixedRunID)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := npmartifact.ParseReference("tiny@1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewInspectRequest(reference)
	if err != nil {
		t.Fatal(err)
	}
	result, operationErr := service.Inspect(context.Background(), request)
	if operationErr == nil || result.Status() != domain.OperationFailed {
		t.Fatalf("result/error = %#v, %v", result, operationErr)
	}
	if _, ok := result.PolicyDecision(); ok {
		t.Fatal("operational failure unexpectedly contains Policy")
	}
}

type tarEntry struct {
	name     string
	body     string
	typeflag byte
}

func archiveBytes(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		flag := entry.typeflag
		if flag == 0 {
			flag = tar.TypeReg
		}
		if err := writer.WriteHeader(&tar.Header{Name: entry.name, Size: int64(len(entry.body)), Mode: 0o600, Typeflag: flag}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func metadataBody(registryURL, integrity string) string {
	return `{"name":"tiny","dist-tags":{"latest":"1.2.3"},"versions":{"1.2.3":{"name":"tiny","version":"1.2.3","dist":{"tarball":"` + strings.TrimSuffix(registryURL, "/") + `/tiny/-/tiny-1.2.3.tgz","integrity":"` + integrity + `"}}}}`
}

func canonicalRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func fixedOperationID() (domain.OperationID, error) {
	return domain.ParseOperationID("op_aaaaaaaaaaaaaaaaaaaaaaaaaa")
}

func fixedRunID() (domain.RunID, error) {
	return domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func response(status int, contentType, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(body))}
}
