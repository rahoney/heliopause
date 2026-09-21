package pypi

import (
	"crypto/sha256"
	"encoding/hex"
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

func TestBoundedMarkerBooleanPrecedenceAndExtras(t *testing.T) {
	profile := PublicPyPIProfile()
	dependency, active, err := parseDeclaredDependencyForProfile(`child; sys_platform == 'linux' or platform_system == 'Windows' and platform_machine == 'arm64'`, profile, "3.14.7")
	if err != nil || !active || dependency != "child" {
		t.Fatalf("precedence marker = %q/%t/%v", dependency, active, err)
	}
	dependency, active, err = parseDeclaredDependencyForProfileWithExtras(`nvidia-cublas==1.0.0.*; extra == 'cublas'`, profile, "3.14.7", []string{"cublas"})
	if err != nil || !active || dependency != "nvidia-cublas" {
		t.Fatalf("extra activated dependency = %q/%t/%v", dependency, active, err)
	}
	if _, active, err = parseDeclaredDependencyForProfileWithExtras(`nvidia-cublas==1.0.0.*; extra == 'cublas'`, profile, "3.14.7", nil); err != nil || active {
		t.Fatalf("unrequested extra = active=%t err=%v", active, err)
	}
	dependency, active, err = parseDeclaredDependencyForProfileWithExtras(`nvidia-cublas==1.0.0.*; extra != 'cublas'`, profile, "3.14.7", []string{"cublas", "cudart"})
	if err != nil || !active || dependency != "nvidia-cublas" {
		t.Fatalf("multi-extra activation = %q/%t/%v", dependency, active, err)
	}
	dependency, active, err = parseDeclaredDependencyForProfileWithExtras(`child; platform_system == 'Linux' or extra == 'foo' and extra == 'bar'`, profile, "3.14.7", []string{"cublas", "cudart"})
	if err != nil || !active || dependency != "child" {
		t.Fatalf("extra precedence marker = %q/%t/%v", dependency, active, err)
	}
	dependency, active, err = parseDeclaredDependencyForProfileWithExtras(`child; (extra == 'cublas' or extra == 'cudart') and platform_system == 'Linux'`, profile, "3.14.7", []string{"cublas", "cudart"})
	if err != nil || !active || dependency != "child" {
		t.Fatalf("parenthesized extra marker = %q/%t/%v", dependency, active, err)
	}
	if _, _, err = parseDeclaredDependencyForProfile(`child; os_name == 'posix' or extra == 'unused'`, profile, "3.14.7"); err == nil {
		t.Fatal("unsupported disjunctive marker was accepted")
	}
}

func TestBoundedMarkerGrammarValidationBeforeEvaluation(t *testing.T) {
	profile := PublicPyPIProfile()

	// 1. Unsupported atom on left of AND (right would evaluate false)
	if _, _, err := parseDeclaredDependencyForProfile(`child; os_name == "posix" and platform_system == "Windows"`, profile, "3.14.7"); err == nil {
		t.Fatal("expected unsupported left atom in AND to be rejected even though right is false")
	}

	// 2. Unsupported atom on right of AND (left would evaluate false)
	if _, _, err := parseDeclaredDependencyForProfile(`child; platform_system == "Windows" and os_name == "posix"`, profile, "3.14.7"); err == nil {
		t.Fatal("expected unsupported right atom in AND to be rejected even though left is false")
	}

	// 3. Unsupported atom on left of OR (right would evaluate true)
	if _, _, err := parseDeclaredDependencyForProfile(`child; os_name == "posix" or platform_system == "Linux"`, profile, "3.14.7"); err == nil {
		t.Fatal("expected unsupported left atom in OR to be rejected even though right is true")
	}

	// 4. Unsupported atom on right of OR (left would evaluate true)
	if _, _, err := parseDeclaredDependencyForProfile(`child; platform_system == "Linux" or os_name == "posix"`, profile, "3.14.7"); err == nil {
		t.Fatal("expected unsupported right atom in OR to be rejected even though left is true")
	}

	// 5. Extra-dependent AND: inactive extra on left, unsupported on right
	if _, _, err := parseDeclaredDependencyForProfileWithExtras(`child; extra == "foo" and unsupported_name == "x"`, profile, "3.14.7", []string{"cublas"}); err == nil {
		t.Fatal("expected unsupported right atom in AND to be rejected even though extra is inactive")
	}

	// 6. Extra-dependent OR: active extra on right, unsupported on left
	if _, _, err := parseDeclaredDependencyForProfileWithExtras(`child; unsupported_name == "x" or extra == "cublas"`, profile, "3.14.7", []string{"cublas"}); err == nil {
		t.Fatal("expected unsupported left atom in OR to be rejected even though extra is active")
	}

	// 7. Nested/grouped unsupported atom where outer branch is true
	if _, _, err := parseDeclaredDependencyForProfileWithExtras(`child; platform_system == "Linux" or (extra == "cublas" and unsupported_name == "x")`, profile, "3.14.7", []string{"cublas"}); err == nil {
		t.Fatal("expected nested unsupported atom in OR branch to be rejected even though outer left is true")
	}
	if _, _, err := parseDeclaredDependencyForProfileWithExtras(`child; (extra == "cublas" and unsupported_name == "x") or platform_system == "Linux"`, profile, "3.14.7", []string{"cublas"}); err == nil {
		t.Fatal("expected nested unsupported atom in grouped AND to be rejected even though outer right is true")
	}
	if _, _, err := parseDeclaredDependencyForProfile(`child; (platform_system == "Windows" and os_name == "posix") and sys_platform == "linux"`, profile, "3.14.7"); err == nil {
		t.Fatal("expected nested unsupported atom in grouped left to be rejected")
	}

	// 8. Production report path: unsupported requirement does not disappear as inactive, it fails candidate processing
	reference, err := ParseReference("parent@1.0")
	if err != nil {
		t.Fatal(err)
	}
	reportJSON := `{"version":"1","pip_version":"26.2.1","environment":{"implementation_name":"cpython","implementation_version":"3.14.7","python_full_version":"3.14.7","platform_machine":"x86_64","sys_platform":"linux"},"install":[{"download_info":{"url":"https://files.pythonhosted.org/packages/parent-1.0-py3-none-any.whl","archive_info":{"hashes":{"sha256":"` + sampleSHA256 + `"}}},"is_direct":false,"requested":true,"metadata":{"name":"parent","version":"1.0","requires_python":"","requires_dist":["child; platform_system == 'Windows' and os_name == 'posix'"]}}]}`
	if _, err := ParseInstallationReportForProfile(reference, []byte(reportJSON), "26.2.1", "3.14.7", profile); err == nil {
		t.Fatal("expected production report with short-circuited unsupported marker to fail closed rather than disappearing as inactive")
	}
}

func TestReportExtrasActivateEdges(t *testing.T) {
	reference, err := ParseReference("cuda-toolkit@1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Replace(sampleReportJSON(), `"name":"Primary","version":"1.0","requires_python":">=3.14","requires_dist":["child>=2"]`, `"name":"cuda-toolkit","version":"1.0.0","requires_python":"","requires_dist":["nvidia-cublas==1.0.0.*; extra != 'cublas'"]`, 1)
	body = strings.Replace(body, "primary-1.0-py3-none-any.whl", "cuda-toolkit-1.0.0-py3-none-any.whl", 2)
	report, err := ParseInstallationReportForProfileWithExtras(reference, []byte(body), pipRuntimeVersionForTest, pythonRuntimeVersionForTest, PublicPyPIProfile(), []string{"cublas", "cudart"})
	if err != nil {
		t.Fatal(err)
	}
	if deps := report.Candidates()[1].Dependencies(); len(deps) != 1 || deps[0] != "nvidia-cublas" {
		t.Fatalf("extra dependency edges = %#v", deps)
	}
}

func TestPyTorchSimpleLinkMetadataIsIndependentAuthority(t *testing.T) {
	profile, ok := PyTorchProfile("cpu")
	if !ok {
		t.Fatal("CPU profile is missing")
	}
	filename := "torch-2.14.0+cpu-cp314-cp314-manylinux_2_28_x86_64.whl"
	linkURL := profile.IndexURL() + "torch/" + filename
	candidate := Candidate{
		project: "torch", source: profile.Source(), filename: filename,
		url: linkURL, sha256: sampleSHA256, requiresPython: ">=3.10",
	}
	coreMetadata := []byte("Metadata-Version: 2.4\nName: torch\nVersion: 2.14.0+cpu\nRequires-Python: >=3.10\n\n")
	coreSum := sha256.Sum256(coreMetadata)
	coreDigest := hex.EncodeToString(coreSum[:])
	// This is a minimized official-shaped link: PyTorch's selected 2.14 links
	// carry the hash-bound Core Metadata attributes but omit optional
	// data-requires-python and data-yanked attributes.
	page, err := ParsePyTorchSimpleProject("torch", []byte(`<a href="`+linkURL+`#sha256=`+sampleSHA256+`" data-core-metadata="sha256=`+coreDigest+`" data-dist-info-metadata="sha256=`+coreDigest+`">torch</a>`), profile)
	if err != nil {
		t.Fatal(err)
	}
	metadataURL, err := PyTorchCoreMetadataURL(candidate, page)
	if err != nil || metadataURL != linkURL+".metadata" {
		t.Fatalf("Core Metadata URL = %q / %v", metadataURL, err)
	}
	if err := VerifyPyTorchCoreMetadata(candidate, page, coreMetadata); err != nil {
		t.Fatalf("hash-bound Core Metadata rejected: %v", err)
	}
	if err := VerifyPyTorchCoreMetadata(candidate, page, []byte("Metadata-Version: 2.4\nRequires-Python: >=3.10\n\n")); err == nil {
		t.Fatal("unhashed Core Metadata was accepted")
	}
	mismatch := candidate
	mismatch.requiresPython = ">=3.11"
	if err := VerifyPyTorchCoreMetadata(mismatch, page, coreMetadata); err == nil {
		t.Fatal("pip Requires-Python disagreement was accepted")
	}
	metadataWithoutRequiresPython := []byte("Metadata-Version: 2.4\nName: torch\n\n")
	withoutRequiresSum := sha256.Sum256(metadataWithoutRequiresPython)
	withoutRequiresPage, err := ParsePyTorchSimpleProject("torch", []byte(`<a href="`+linkURL+`#sha256=`+sampleSHA256+`" data-core-metadata="sha256=`+hex.EncodeToString(withoutRequiresSum[:])+`">torch</a>`), profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyPyTorchCoreMetadata(candidate, withoutRequiresPage, metadataWithoutRequiresPython); err == nil {
		t.Fatal("authenticated Requires-Python absence authorized pip metadata")
	}

	present, err := ParsePyTorchSimpleProject("torch", []byte(`<a href="`+linkURL+`#sha256=`+sampleSHA256+`" data-requires-python=">=3.10">torch</a>`), profile)
	if err != nil {
		t.Fatal(err)
	}
	if metadataURL, err := PyTorchCoreMetadataURL(candidate, present); err != nil || metadataURL != "" {
		t.Fatalf("present Requires-Python authentication = %q / %v", metadataURL, err)
	}
	if _, err := PyTorchCoreMetadataURL(mismatch, present); err == nil {
		t.Fatal("Simple Requires-Python mismatch was accepted")
	}
	emptyPresent, err := ParsePyTorchSimpleProject("torch", []byte(`<a href="`+linkURL+`#sha256=`+sampleSHA256+`" data-requires-python="">torch</a>`), profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PyTorchCoreMetadataURL(candidate, emptyPresent); err == nil {
		t.Fatal("empty Simple Requires-Python authorized pip metadata")
	}

	for _, attribute := range []string{"data-yanked", `data-yanked=""`, `data-yanked="false"`, `data-yanked="withdrawn"`} {
		yanked, err := ParsePyTorchSimpleProject("torch", []byte(`<a href="`+linkURL+`#sha256=`+sampleSHA256+`" `+attribute+`>torch</a>`), profile)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := PyTorchCoreMetadataURL(candidate, yanked); err == nil {
			t.Fatalf("yanked link %q was accepted", attribute)
		}
	}

	absent, err := ParsePyTorchSimpleProject("torch", []byte(`<a href="`+linkURL+`#sha256=`+sampleSHA256+`">torch</a>`), profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PyTorchCoreMetadataURL(candidate, absent); err == nil {
		t.Fatal("missing Requires-Python and Core Metadata authentication was accepted")
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
