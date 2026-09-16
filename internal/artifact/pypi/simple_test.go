package pypi

import (
	"errors"
	"strings"
	"testing"
)

const sampleSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestSimpleReportCrossCheckBuildsExactGraph(t *testing.T) {
	t.Parallel()

	reference, err := ParseReference("Primary@1.0")
	if err != nil {
		t.Fatal(err)
	}
	report, err := ParseInstallationReport(reference, []byte(sampleReportJSON()), pipRuntimeVersionForTest, pythonRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}
	primary, err := ParseSimpleProject("primary", []byte(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ">=3.14")))
	if err != nil {
		t.Fatal(err)
	}
	child, err := ParseSimpleProject("child", []byte(sampleSimpleJSON("child", "child-2.0-py3-none-any.whl", "")))
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := CrossCheckReport(report, []SimpleProject{primary, child})
	if err != nil {
		t.Fatal(err)
	}
	graph, err := BuildLockedGraph(reference, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Nodes()) != 2 || len(graph.Edges()) != 1 || graph.Primary().String() == "" {
		t.Fatalf("graph = %#v", graph)
	}
	for _, node := range graph.Nodes() {
		if node.Artifact().Identity().Variant() != "wheel" || node.Artifact().DeclaredIntegrity() != "sha256:"+sampleSHA256 {
			t.Fatalf("node = %#v", node)
		}
	}
}

func TestSimpleAPIAndReportRejectIncompleteOrUnsafeMetadata(t *testing.T) {
	t.Parallel()

	reference, err := ParseReference("primary@1.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		strings.Replace(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ""), `"api-version":"1.4"`, `"api-version":"2.0"`, 1),
		strings.Replace(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ""), `"size":123`, `"size":0`, 1),
		strings.Replace(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ""), `files.pythonhosted.org`, `mirror.example`, 1),
	} {
		if _, err := ParseSimpleProject("primary", []byte(body)); err == nil {
			t.Fatalf("ParseSimpleProject accepted %s", body)
		}
	}
	for _, body := range []string{
		strings.Replace(sampleReportJSON(), `"pip_version":"26.2.1"`, `"pip_version":"26.1"`, 1),
		strings.Replace(sampleReportJSON(), `"is_direct":false`, `"is_direct":true`, 1),
		strings.Replace(sampleReportJSON(), `"requires_dist":["child>=2"]`, `"requires_dist":["child; os_name == 'posix'"]`, 1),
		strings.Replace(sampleReportJSON(), `"url":"https://files.pythonhosted.org/packages/primary-1.0-py3-none-any.whl"`, `"url":"https://files.pythonhosted.org/packages/primary-1.0-py3-none-any.whl?bad=1"`, 1),
	} {
		if _, err := ParseInstallationReport(reference, []byte(body), pipRuntimeVersionForTest, pythonRuntimeVersionForTest); err == nil {
			t.Fatalf("ParseInstallationReport accepted %s", body)
		}
	}
}

func TestPublicPyPIEvaluatesSupportedDeterministicMarkers(t *testing.T) {
	profile := PublicPyPIProfile()

	t.Run("exact triton false marker omits dependency", func(t *testing.T) {
		dep, active, err := parseDeclaredDependencyForProfile(`importlib-metadata; python_version < "3.10"`, profile, "3.14.7")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if active {
			t.Fatalf("expected inactive dependency, got active=true (dep=%q)", dep)
		}
	})

	t.Run("supported true marker retains dependency", func(t *testing.T) {
		dep, active, err := parseDeclaredDependencyForProfile(`filelock; sys_platform == 'linux'`, profile, "3.14.7")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !active || dep != "filelock" {
			t.Fatalf("expected active filelock, got dep=%q active=%t", dep, active)
		}

		dep, active, err = parseDeclaredDependencyForProfile(`typing-extensions>=4.0.0; python_version >= '3.10'`, profile, "3.14.7")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !active || dep != "typing-extensions" {
			t.Fatalf("expected active typing-extensions, got dep=%q active=%t", dep, active)
		}
	})

	t.Run("unsupported marker fails closed", func(t *testing.T) {
		for _, invalid := range []string{
			"pkg; os_name == 'posix'",
			"pkg; python_version == 3.14",
			"pkg; extra == 'foo' or extra == 'bar'",
			"pkg; implementation_name == 'cpython'",
			"pkg; sys_platform == 'linux'; extra == 'test'",
		} {
			_, _, err := parseDeclaredDependencyForProfile(invalid, profile, "3.14.7")
			if err == nil {
				t.Fatalf("parseDeclaredDependencyForProfile accepted unsupported marker: %q", invalid)
			}
		}
	})
}

func TestSimpleProjectMetadataDiagnosticsAreBounded(t *testing.T) {
	base := sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", "")
	cases := []struct{ name, body, want string }{
		{"name invalid", strings.Replace(base, `"name":"primary"`, `"name":"!!!"`, 1), "reason=NAME_INVALID project=primary response= files=1 limit=1024"},
		{"name mismatch", strings.Replace(base, `"name":"primary"`, `"name":"other"`, 1), "reason=NAME_MISMATCH project=primary response=other files=1 limit=1024"},
		{"files empty", `{"meta":{"api-version":"1.4"},"name":"primary","files":[]}`, "reason=FILES_EMPTY project=primary response=primary files=0 limit=1024"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseSimpleProject("primary", []byte(test.body))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
	var body strings.Builder
	body.WriteString(`{"meta":{"api-version":"1.4"},"name":"numpy","files":[`)
	for i := 0; i < maxPyPIReportEntries+1; i++ {
		if i > 0 {
			body.WriteByte(',')
		}
		body.WriteString(`{}`)
	}
	body.WriteString(`]}`)
	_, err := ParseSimpleProject("numpy", []byte(body.String()))
	if err == nil || !strings.Contains(err.Error(), "reason=FILES_LIMIT project=numpy response=numpy files=1025 limit=1024") {
		t.Fatalf("files limit diagnostic = %v", err)
	}
}

func TestSimpleFilesLimitIsSelectedOnlyByCanonicalPyTorchRoot(t *testing.T) {
	cpu, _ := PyTorchProfile("cpu")
	cu126, _ := PyTorchProfile("cu126")
	if simpleFilesLimit(PublicPyPIProfile()) != 1024 || simpleFilesLimit(SourceProfile{}) != 1024 {
		t.Fatal("default root acquired enlarged Simple limit")
	}
	if simpleFilesLimit(cpu) != 8192 || simpleFilesLimit(cu126) != 8192 {
		t.Fatalf("PyTorch root limits = %d/%d", simpleFilesLimit(cpu), simpleFilesLimit(cu126))
	}
}

func TestUnsupportedRequirementDiagnosticIsSanitized(t *testing.T) {
	err := unsupportedRequirementDiagnostic("parent", "setuptools; python_version < '3.14' and extra == 'test'", errors.New("unsupported dependency requirement marker"))
	got := err.Error()
	for _, want := range []string{"reason=UNSUPPORTED_REQUIREMENT", "package=parent", "dependency=setuptools", "shape=compound", "detail=MARKER"} {
		if !strings.Contains(got, want) {
			t.Fatalf("diagnostic %q missing %q", got, want)
		}
	}
	if strings.Contains(got, "python_version") || strings.Contains(got, "test") {
		t.Fatalf("diagnostic leaked raw marker: %q", got)
	}
}

func TestActiveMarkerProducesEffectiveChildRequirement(t *testing.T) {
	profile, _ := PyTorchProfile("cpu")
	dependency, active, err := parseDeclaredDependencyForProfile("foo>=1.2; python_version >= '3.10'", profile, "3.14.7")
	if err != nil || !active || dependency != "foo" {
		t.Fatalf("marker parse = %q/%t/%v", dependency, active, err)
	}
	effective := strings.TrimSpace(strings.SplitN("foo>=1.2; python_version >= '3.10'", ";", 2)[0])
	if effective != "foo>=1.2" {
		t.Fatalf("effective requirement = %q", effective)
	}
}

func TestInactiveMarkerDoesNotProduceChildRequirement(t *testing.T) {
	profile, _ := PyTorchProfile("cpu")
	_, active, err := parseDeclaredDependencyForProfile("foo>=1.2; python_version < '3.10'", profile, "3.14.7")
	if err != nil || active {
		t.Fatalf("inactive marker = active=%t err=%v", active, err)
	}
}

func TestInstallationReportAcceptsCanonicalHashesWithoutLegacyHash(t *testing.T) {
	t.Parallel()

	reference, err := ParseReference("primary@1.0")
	if err != nil {
		t.Fatal(err)
	}
	body := strings.ReplaceAll(sampleReportJSON(), `"hash":"sha256=`+sampleSHA256+`",`, "")
	if _, err := ParseInstallationReport(reference, []byte(body), pipRuntimeVersionForTest, pythonRuntimeVersionForTest); err != nil {
		t.Fatalf("ParseInstallationReport rejected pip schema hashes without legacy hash: %v", err)
	}
}

func TestInstallationReportIgnoresUnrequestedExtras(t *testing.T) {
	t.Parallel()

	reference, err := ParseReference("Primary@1.0")
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Replace(sampleReportJSON(), `"requires_dist":["child>=2"]`, `"requires_dist":["child>=2","dask[dataframe,test]; extra == 'test-downstream'","backports-zstd; (python_version < '3.14') and extra == 'test-full'"]`, 1)
	report, err := ParseInstallationReport(reference, []byte(body), pipRuntimeVersionForTest, pythonRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range report.Candidates() {
		if candidate.Project() != "primary" {
			continue
		}
		if deps := candidate.Dependencies(); len(deps) != 1 || deps[0] != "child" {
			t.Fatalf("inactive extra dependency was retained: %#v", deps)
		}
	}
}

func TestCrossCheckRejectsYankedAndMismatchedFiles(t *testing.T) {
	t.Parallel()

	reference, _ := ParseReference("primary@1.0")
	report, err := ParseInstallationReport(reference, []byte(sampleReportJSON()), pipRuntimeVersionForTest, pythonRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}
	primary, _ := ParseSimpleProject("primary", []byte(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ">=3.14")))
	child, _ := ParseSimpleProject("child", []byte(sampleSimpleJSON("child", "child-2.0-py3-none-any.whl", "")))
	if _, err := CrossCheckReport(report, []SimpleProject{primary, child}); err != nil {
		t.Fatal(err)
	}
	yanked, err := ParseSimpleProject("primary", []byte(strings.Replace(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ">=3.14"), `"yanked":false`, `"yanked":"withdrawn"`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CrossCheckReport(report, []SimpleProject{yanked, child}); err == nil {
		t.Fatal("CrossCheckReport accepted yanked selection")
	}
	badURL, err := ParseSimpleProject("primary", []byte(strings.Replace(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ">=3.14"), `packages/primary-1.0-py3-none-any.whl`, `packages/other/primary-1.0-py3-none-any.whl`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CrossCheckReport(report, []SimpleProject{badURL, child}); err == nil {
		t.Fatal("CrossCheckReport accepted mismatched URL")
	}
	badHash, err := ParseSimpleProject("primary", []byte(strings.Replace(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ">=3.14"), sampleSHA256, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 1)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CrossCheckReport(report, []SimpleProject{badHash, child}); err == nil {
		t.Fatal("CrossCheckReport accepted mismatched hash")
	}
	badFilename, err := ParseSimpleProject("primary", []byte(sampleSimpleJSON("primary", "primary-2.0-py3-none-any.whl", ">=3.14")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CrossCheckReport(report, []SimpleProject{badFilename, child}); err == nil {
		t.Fatal("CrossCheckReport accepted mismatched filename")
	}
}

func TestCrossCheckRequiresPythonSerializationOrderAndFailClosed(t *testing.T) {
	t.Parallel()

	reference, err := ParseReference("primary@1.0")
	if err != nil {
		t.Fatal(err)
	}

	// Proven triton condition:
	// candidate has ">=3.10,<3.15"
	// simple page has "<3.15,>=3.10"
	reportJSON := strings.Replace(sampleReportJSON(), `">=3.14"`, `">=3.10,<3.15"`, 1)
	report, err := ParseInstallationReport(reference, []byte(reportJSON), pipRuntimeVersionForTest, pythonRuntimeVersionForTest)
	if err != nil {
		t.Fatal(err)
	}

	primaryTritonOrder, err := ParseSimpleProject("primary", []byte(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", "<3.15,>=3.10")))
	if err != nil {
		t.Fatal(err)
	}
	child, err := ParseSimpleProject("child", []byte(sampleSimpleJSON("child", "child-2.0-py3-none-any.whl", "")))
	if err != nil {
		t.Fatal(err)
	}

	candidates, err := CrossCheckReport(report, []SimpleProject{primaryTritonOrder, child})
	if err != nil {
		t.Fatalf("CrossCheckReport rejected order-equivalent requires-python: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	// Operator mismatch: ">=3.10,<=3.15" vs ">=3.10,<3.15"
	opMismatch, err := ParseSimpleProject("primary", []byte(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ">=3.10,<=3.15")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CrossCheckReport(report, []SimpleProject{opMismatch, child}); err == nil {
		t.Fatal("CrossCheckReport accepted operator mismatch >=3.10,<=3.15")
	}

	// Additional constraint: ">=3.10,<3.15,!=3.12" vs ">=3.10,<3.15"
	extraConstraint, err := ParseSimpleProject("primary", []byte(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ">=3.10,<3.15,!=3.12")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CrossCheckReport(report, []SimpleProject{extraConstraint, child}); err == nil {
		t.Fatal("CrossCheckReport accepted extra constraint !=3.12")
	}

	// Duplicate multiplicity: ">=3.10,<3.15,<3.15" vs ">=3.10,<3.15"
	duplicateConstraint, err := ParseSimpleProject("primary", []byte(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ">=3.10,<3.15,<3.15")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CrossCheckReport(report, []SimpleProject{duplicateConstraint, child}); err == nil {
		t.Fatal("CrossCheckReport accepted duplicate constraint preserving multiplicity")
	}

	// Whitespace difference: ">=3.10, <3.15" vs ">=3.10,<3.15"
	wsMismatch, err := ParseSimpleProject("primary", []byte(sampleSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ">=3.10, <3.15")))
	if err == nil {
		if _, err := CrossCheckReport(report, []SimpleProject{wsMismatch, child}); err == nil {
			t.Fatal("CrossCheckReport accepted whitespace difference >=3.10, <3.15")
		}
	}
}

func TestRequiresPythonMetadataMatches(t *testing.T) {
	t.Parallel()

	// 1. Proven triton order equivalence
	if !requiresPythonMetadataMatches("<3.15,>=3.10", ">=3.10,<3.15") {
		t.Fatal("triton condition <3.15,>=3.10 did not match >=3.10,<3.15")
	}
	if !requiresPythonMetadataMatches(">=3.10,<3.15", "<3.15,>=3.10") {
		t.Fatal("triton condition >=3.10,<3.15 did not match <3.15,>=3.10")
	}

	// 2. Whitespace normalization is strictly forbidden
	if requiresPythonMetadataMatches(">=3.10,<3.15", ">=3.10, <3.15") {
		t.Fatal("whitespace after comma must not compare equal")
	}
	if requiresPythonMetadataMatches(">=3.10,<3.15", ">= 3.10,<3.15") {
		t.Fatal("operator whitespace must not compare equal")
	}

	// 3. Exact match
	if !requiresPythonMetadataMatches(">=3.10,<3.15", ">=3.10,<3.15") {
		t.Fatal("exact byte-equal specifiers did not match")
	}

	// 4. Empty simple preserves no equality comparison
	if !requiresPythonMetadataMatches("", ">=3.10,<3.15") {
		t.Fatal("empty simple requires-python must match any candidate")
	}
	if !requiresPythonMetadataMatches("", "") {
		t.Fatal("empty simple and candidate must match")
	}

	// 5. Non-empty simple requires non-empty candidate
	if requiresPythonMetadataMatches(">=3.10,<3.15", "") {
		t.Fatal("non-empty simple must not match empty candidate")
	}

	// 6. Operator mismatch fails closed
	if requiresPythonMetadataMatches(">=3.10,<=3.15", ">=3.10,<3.15") {
		t.Fatal("operator mismatch >=3.10,<=3.15 matched >=3.10,<3.15")
	}

	// 7. Extra constraint fails closed
	if requiresPythonMetadataMatches(">=3.10,<3.15,!=3.12", ">=3.10,<3.15") {
		t.Fatal("extra constraint !=3.12 matched >=3.10,<3.15")
	}

	// 8. Empty/malformed components fail closed even when identical on both sides
	if requiresPythonMetadataMatches(">=3.10,,<3.15", ">=3.10,,<3.15") {
		t.Fatal("identical empty component >=3.10,,<3.15 must fail closed")
	}
	if requiresPythonMetadataMatches(">=3.10,,<3.15", ">=3.10,<3.15") {
		t.Fatal("empty component >=3.10,,<3.15 matched >=3.10,<3.15")
	}
	if requiresPythonMetadataMatches(",>=3.10", ",>=3.10") {
		t.Fatal("identical leading comma must fail closed")
	}
	if requiresPythonMetadataMatches(",>=3.10,<3.15", ">=3.10,<3.15") {
		t.Fatal("leading comma matched >=3.10,<3.15")
	}
	if requiresPythonMetadataMatches(">=3.10,", ">=3.10,") {
		t.Fatal("identical trailing comma must fail closed")
	}
	if requiresPythonMetadataMatches(">=3.10,<3.15,", ">=3.10,<3.15") {
		t.Fatal("trailing comma matched >=3.10,<3.15")
	}

	// 9. Multiplicity preserved (duplicates fail closed)
	if requiresPythonMetadataMatches(">=3.10,<3.15,<3.15", ">=3.10,<3.15") {
		t.Fatal("multiplicity difference matched")
	}
	if requiresPythonMetadataMatches(">=3.10,<3.15", ">=3.10,<3.15,<3.15") {
		t.Fatal("multiplicity difference matched")
	}

	// 10. No range simplification (solver not present)
	if requiresPythonMetadataMatches(">=3.11", ">=3.10,>=3.11") {
		t.Fatal("range simplification incorrectly matched")
	}

	// 11. Malformed component missing operator fails closed
	if requiresPythonMetadataMatches("<3.15,3.10", "3.10,<3.15") {
		t.Fatal("specifier without operator matched")
	}
	if requiresPythonMetadataMatches("3.10,<3.15", "3.10,<3.15") {
		t.Fatal("identical specifier without operator must fail closed")
	}

	// 12. Arbitrary non-specifier strings fail closed
	if requiresPythonMetadataMatches("foo,bar", "bar,foo") {
		t.Fatal("arbitrary non-specifier comma string matched")
	}
	if requiresPythonMetadataMatches("foo,bar", "foo,bar") {
		t.Fatal("identical arbitrary non-specifier comma string must fail closed")
	}
}

func TestNormalizeProjectAndFinalVersion(t *testing.T) {
	t.Parallel()

	if got, err := NormalizeProjectName("Example_Package"); err != nil || got != "example-package" {
		t.Fatalf("NormalizeProjectName() = %q, %v", got, err)
	}
	if !IsFinalVersion("1.0.post1") || IsFinalVersion("1.0rc1") || IsFinalVersion("1.0.dev1") {
		t.Fatal("IsFinalVersion() did not distinguish final releases")
	}
}

const (
	pipRuntimeVersionForTest    = "26.2.1"
	pythonRuntimeVersionForTest = "3.14.7"
)

func sampleSimpleJSON(project, filename, requiresPython string) string {
	return `{"meta":{"api-version":"1.4"},"name":"` + project + `","files":[{"filename":"` + filename + `","url":"https://files.pythonhosted.org/packages/` + filename + `","hashes":{"sha256":"` + sampleSHA256 + `"},"requires-python":"` + requiresPython + `","yanked":false,"size":123}]}`
}

func sampleReportJSON() string {
	return `{"version":"1","pip_version":"26.2.1","environment":{"implementation_name":"cpython","implementation_version":"3.14.7","python_full_version":"3.14.7","platform_machine":"x86_64","sys_platform":"linux"},"install":[{"download_info":{"url":"https://files.pythonhosted.org/packages/primary-1.0-py3-none-any.whl","archive_info":{"hash":"sha256=` + sampleSHA256 + `","hashes":{"sha256":"` + sampleSHA256 + `"}}},"is_direct":false,"requested":true,"metadata":{"name":"Primary","version":"1.0","requires_python":">=3.14","requires_dist":["child>=2"]}},{"download_info":{"url":"https://files.pythonhosted.org/packages/child-2.0-py3-none-any.whl","archive_info":{"hash":"sha256=` + sampleSHA256 + `","hashes":{"sha256":"` + sampleSHA256 + `"}}},"is_direct":false,"requested":false,"metadata":{"name":"child","version":"2.0","requires_python":"","requires_dist":[]}}]}`
}
