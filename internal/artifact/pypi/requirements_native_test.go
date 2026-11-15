package pypi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestRequirementsNativeProjectionRequiresSourceAndRuntimeBinding(t *testing.T) {
	input, err := NewRequirementsInput(nil, nil, []string{"requests[security]>=2"})
	if err != nil {
		t.Fatal(err)
	}
	control, _ := input.Control()
	parserControl, _ := input.ParserControl()
	textHash := sha256.Sum256([]byte("requests[security]>=2"))
	clause := map[string]any{"active": true, "source": "@positional", "line": 1, "text_sha256": hex.EncodeToString(textHash[:]), "requirement": "requests[security]>=2", "project": "requests", "extras": []string{"security"}, "constraint": false, "hashes": map[string][]string{}}
	original := map[string]any{"version": 1, "input_sha256": control.Digest().String(), "parser_sha256": parserControl.Digest().String(), "python": runtimeidentity.PythonVersion, "pip": runtimeidentity.PipVersion, "require_hashes": false, "clauses": []any{clause}}
	body, _ := json.Marshal(original)
	parsed, err := input.ParseNativeInput(body)
	if err != nil || !parsed.Valid() || len(parsed.Clauses()) != 1 {
		t.Fatal("bound projection rejected", err)
	}
	for _, field := range []string{"parser_sha256", "input_sha256", "python", "pip", "require_hashes", "clauses"} {
		copy := map[string]any{}
		for key, value := range original {
			copy[key] = value
		}
		delete(copy, field)
		body, _ := json.Marshal(copy)
		if _, err := input.ParseNativeInput(body); err == nil {
			t.Fatal("missing binding accepted", field)
		}
	}
	for _, bad := range []string{"requests @ https://evil.invalid/a.whl", "requests --index-url=evil", "wrong[security]>=2"} {
		clause["requirement"] = bad
		body, _ := json.Marshal(original)
		if _, err := input.ParseNativeInput(body); err == nil {
			t.Fatal("unsafe projection accepted", bad)
		}
	}
	clause["requirement"] = "requests[security]>=2"
	for _, field := range []string{"constraint", "hashes", "extras", "active", "source", "line", "text_sha256"} {
		value := clause[field]
		delete(clause, field)
		body, _ := json.Marshal(original)
		if _, err := input.ParseNativeInput(body); err == nil {
			t.Fatal("missing clause fact accepted", field)
		}
		clause[field] = value
	}
	body, _ = json.Marshal(original)
	duplicate := strings.Replace(string(body), `"version":1`, `"version":1,"version":1`, 1)
	if _, err := input.ParseNativeInput([]byte(duplicate)); err == nil {
		t.Fatal("duplicate projection key accepted")
	}
}

func TestRequirementsProposalPreservesFrozenVersionsAndOriginalMetadata(t *testing.T) {
	base, err := NewRequirementsInput(nil, nil, []string{"alpha>=1", "beta"})
	if err != nil {
		t.Fatal(err)
	}
	files := []InstalledMetadataFile{{"alpha-1.0.dist-info", []byte("Metadata-Version: 2.1\nName: alpha\nVersion: 1.0\n\n")}, {"retained-1.0.dist-info", []byte("Metadata-Version: 2.1\nName: retained\nVersion: 1.0\nRequires-Dist: beta>=1\n\n")}}
	for _, client := range []string{"pip", "uv"} {
		t.Run(client, func(t *testing.T) {
			input, err := base.WithInstallation(client, true, files)
			if err != nil {
				t.Fatal(err)
			}
			control, _ := input.Control()
			native := RequirementsNativeInput{inputDigest: control.Digest(), clauses: []RequirementsClause{{Active: true, Project: "alpha", Requirement: "alpha>=1"}, {Active: true, Project: "beta", Requirement: "beta"}}}
			delta := requirementsReport(t, upgradeCandidate("alpha", "2.0", true), upgradeCandidate("beta", "1.0", true))
			if client == "uv" {
				delta = []byte(`{"schema":{"version":"preview"},"dry_run":true,"changes":[{"name":"alpha","version":"1.0","action":"uninstalled"},{"name":"alpha","version":"2.0","action":"installed"},{"name":"beta","version":"1.0","action":"installed"}]}`)
			}
			constraints, err := input.SelectionConstraints(native, delta)
			if err != nil {
				t.Fatal(err)
			}
			if string(constraints) != "alpha==2.0\nbeta==1.0\nretained==1.0\n" {
				t.Fatal("native proposal or retained version pins lost", string(constraints))
			}
			if client == "uv" {
				delta = []byte(`{"schema":{"version":"preview"},"dry_run":true,"changes":[]}`)
			} else {
				delta = requirementsReport(t)
			}
			if _, err := input.SelectionConstraints(native, delta); err == nil {
				t.Fatal("missing requested root accepted")
			}
			baseControl, _ := base.Control()
			native.inputDigest = baseControl.Digest()
			if _, err := input.SelectionConstraints(native, delta); err == nil {
				t.Fatal("unbound source accepted")
			}
		})
	}
}

func requirementsReport(t *testing.T, items ...pipInstall) []byte {
	t.Helper()
	var report pipReport
	if err := json.Unmarshal(upgradeReport(t, items...), &report); err != nil {
		t.Fatal(err)
	}
	report.PipVersion = runtimeidentity.PipVersion
	report.Environment.ImplementationVersion = runtimeidentity.PythonVersion
	report.Environment.PythonFullVersion = runtimeidentity.PythonVersion
	body, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestRequirementsFinalPipReportRequiresEveryRealRootWithoutFakePrimary(t *testing.T) {
	input, _ := NewRequirementsInput(nil, nil, []string{"alpha>=1", "beta"})
	control, _ := input.Control()
	native := RequirementsNativeInput{inputDigest: control.Digest(), clauses: []RequirementsClause{{Active: true, Project: "alpha"}, {Active: true, Project: "beta"}}}
	alpha, beta, child := upgradeCandidate("alpha", "2.0", true), upgradeCandidate("beta", "1.0", true), upgradeCandidate("child", "1.0", false)
	report, err := native.ParsePipReport(requirementsReport(t, alpha, beta, child))
	if err != nil || len(report.Candidates()) != 3 {
		t.Fatal("complete multi-root native report rejected", err)
	}
	for _, kind := range []string{"missing-root", "duplicate", "wrong-root", "foreign-source", "sdist", "wrong-runtime"} {
		t.Run(kind, func(t *testing.T) {
			items := []pipInstall{alpha, beta, child}
			switch kind {
			case "missing-root":
				items = items[:1]
			case "duplicate":
				items = append(items, alpha)
			case "wrong-root":
				items[2].Requested = true
			case "foreign-source":
				items[0].DownloadInfo.URL = "https://evil.invalid/alpha-2.0-py3-none-any.whl"
			case "sdist":
				items[0].DownloadInfo.URL = "https://files.pythonhosted.org/packages/alpha-2.0.tar.gz"
			}
			body := requirementsReport(t, items...)
			if kind == "wrong-runtime" {
				body = []byte(strings.ReplaceAll(string(body), runtimeidentity.PipVersion, "0.0"))
			}
			if _, err := native.ParsePipReport(body); err == nil {
				t.Fatal("invalid complete native selection accepted", kind)
			}
		})
	}
}

func TestRequirementsNativeRefusalReportsOnlyBoundSourceLocation(t *testing.T) {
	input, err := NewRequirementsInput([]RequirementsSource{{Name: "requirements.txt"}}, map[string][]byte{"requirements.txt": []byte("# comment\nalpha>=\n")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	control, _ := input.Control()
	parser, _ := input.ParserControl()
	for _, kind := range []string{"bound", "foreign-file", "foreign-line", "wrong-input", "wrong-parser", "wrong-runtime", "unknown-code", "success-and-error", "diagnostic-injection"} {
		t.Run(kind, func(t *testing.T) {
			refusal := map[string]any{"source": "requirements.txt", "line": 2, "code": "INVALID_REQUIREMENT"}
			wire := map[string]any{"version": 1, "input_sha256": control.Digest().String(), "parser_sha256": parser.Digest().String(), "python": runtimeidentity.PythonVersion, "pip": runtimeidentity.PipVersion, "error": refusal}
			switch kind {
			case "foreign-file":
				refusal["source"] = "credentials"
			case "foreign-line":
				refusal["line"] = 1
			case "wrong-input":
				wire["input_sha256"] = strings.Repeat("a", 64)
			case "wrong-parser":
				wire["parser_sha256"] = strings.Repeat("a", 64)
			case "wrong-runtime":
				wire["python"] = "0.0"
			case "unknown-code":
				refusal["code"] = "secret-from-stderr"
			case "success-and-error":
				wire["clauses"] = []any{}
				wire["require_hashes"] = false
			case "diagnostic-injection":
				refusal["message"] = "secret-from-stderr"
			}
			body, _ := json.Marshal(wire)
			parsed, cause := input.ParseNativeInput(body)
			if cause == nil || parsed.Valid() {
				t.Fatal("refusal became selection", kind)
			}
			expected := `requirements "requirements.txt":2: invalid or unsupported syntax for the pinned runtime`
			if kind == "bound" && cause.Error() != expected {
				t.Fatal(cause)
			}
			if kind != "bound" && (cause.Error() == expected || strings.Contains(cause.Error(), "secret-from-stderr")) {
				t.Fatal("unbound or arbitrary diagnostic exposed", cause)
			}
		})
	}
}
