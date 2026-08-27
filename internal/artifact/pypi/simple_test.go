package pypi

import (
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
		strings.Replace(sampleReportJSON(), `"requires_dist":["child>=2"]`, `"requires_dist":["child; python_version < '3.14'"]`, 1),
		strings.Replace(sampleReportJSON(), `"url":"https://files.pythonhosted.org/packages/primary-1.0-py3-none-any.whl"`, `"url":"https://files.pythonhosted.org/packages/primary-1.0-py3-none-any.whl?bad=1"`, 1),
	} {
		if _, err := ParseInstallationReport(reference, []byte(body), pipRuntimeVersionForTest, pythonRuntimeVersionForTest); err == nil {
			t.Fatalf("ParseInstallationReport accepted %s", body)
		}
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
	body := strings.Replace(sampleReportJSON(), `"requires_dist":["child>=2"]`, `"requires_dist":["child>=2","adlfs; extra == 'abfs'"]`, 1)
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
