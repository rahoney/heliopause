package pypi

import (
	"strings"
	"testing"
)

func TestPyTorchProfilesAreNamedAndBounded(t *testing.T) {
	for _, name := range []string{"cpu", "cu126", "cu128"} {
		profile, ok := PyTorchProfile(name)
		if !ok || profile.Name() != "pytorch:"+name || !IsPyTorchSource(profile.Source()) {
			t.Fatalf("profile %q = %#v, %v", name, profile, ok)
		}
		if !strings.HasPrefix(profile.IndexURL(), "https://download.pytorch.org/whl/"+name+"/") {
			t.Fatalf("profile %q index URL = %q", name, profile.IndexURL())
		}
	}
	if _, err := ParseReferenceForSource("torch@2.0.0+cpu", mustPyTorchProfile(t, "cpu").Source()); err != nil {
		t.Fatalf("ParseReferenceForSource(local) error = %v", err)
	}
	if _, err := ParseReferenceForSource("torch@2.0.0+cpu", PublicPyPIProfile().Source()); err == nil {
		t.Fatal("PyPI accepted a local PyTorch version")
	}
}

func TestPyTorchHTMLIndexAndReportPreserveSourceIdentity(t *testing.T) {
	profile := mustPyTorchProfile(t, "cpu")
	reference, err := ParseReferenceForSource("torch", profile.Source())
	if err != nil {
		t.Fatal(err)
	}
	report, err := ParseInstallationReportForProfile(reference, []byte(`{"version":"1","pip_version":"26.2.1","environment":{"implementation_name":"cpython","implementation_version":"3.14.7","python_full_version":"3.14.7","platform_machine":"x86_64","sys_platform":"linux"},"install":[{"download_info":{"url":"https://download.pytorch.org/whl/cpu/torch/torch-2.0.0+cpu-cp314-cp314-linux_x86_64.whl","archive_info":{"hash":"sha256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","hashes":{"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}},"is_direct":false,"requested":true,"metadata":{"name":"torch","version":"2.0.0+cpu","requires_python":">=3.9","requires_dist":["numpy>=1"]}},{"download_info":{"url":"https://files.pythonhosted.org/packages/numpy-1.0-py3-none-any.whl","archive_info":{"hash":"sha256=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","hashes":{"sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}},"is_direct":false,"requested":false,"metadata":{"name":"numpy","version":"1.0","requires_python":"","requires_dist":[]}}]}`), "26.2.1", "3.14.7", profile)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Candidates()) != 2 || report.Candidates()[0].Source() == report.Candidates()[1].Source() {
		t.Fatalf("candidate source identities = %#v", report.Candidates())
	}
	torchPage, err := ParseSimpleProjectForProfile("torch", []byte(`<html><body><a href="https://download.pytorch.org/whl/cpu/torch/torch-2.0.0%2Bcpu-cp314-cp314-linux_x86_64.whl#sha256=`+strings.Repeat("a", 64)+`">torch</a></body></html>`), profile)
	if err != nil {
		t.Fatal(err)
	}
	numpyPage, err := ParseSimpleProject("numpy", []byte(strings.Replace(sampleSimpleJSON("numpy", "numpy-1.0-py3-none-any.whl", ""), sampleSHA256, strings.Repeat("b", 64), 1)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CrossCheckReport(report, []SimpleProject{torchPage, numpyPage}); err != nil {
		t.Fatalf("CrossCheckReport() error = %v", err)
	}
	graph, err := BuildLockedGraph(reference, report.Candidates())
	if err != nil {
		t.Fatalf("graph = %#v, error = %v", graph, err)
	}
	for _, node := range graph.Nodes() {
		if node.Artifact().Identity().Name() == "torch" && node.Artifact().Identity().Source() == PublicPyPIProfile().Source() {
			t.Fatal("torch graph node lost PyTorch source identity")
		}
	}
}

func TestPyTorchSourceOwnershipRejectsConfusion(t *testing.T) {
	profile := mustPyTorchProfile(t, "cpu")
	reference, _ := ParseReferenceForSource("torch", profile.Source())
	body := strings.Replace(`{"version":"1","pip_version":"26.2.1","environment":{"implementation_name":"cpython","implementation_version":"3.14.7","python_full_version":"3.14.7","platform_machine":"x86_64","sys_platform":"linux"},"install":[{"download_info":{"url":"https://files.pythonhosted.org/packages/torch-2.0-py3-none-any.whl","archive_info":{"hash":"sha256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","hashes":{"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}},"is_direct":false,"requested":true,"metadata":{"name":"torch","version":"2.0","requires_python":"","requires_dist":[]}}]}`, "torch-2.0-py3-none-any.whl", "torch-2.0-py3-none-any.whl", 1)
	if _, err := ParseInstallationReportForProfile(reference, []byte(body), "26.2.1", "3.14.7", profile); err == nil {
		t.Fatal("PyTorch report accepted a PyPI-owned torch candidate")
	}
}

func mustPyTorchProfile(t *testing.T, name string) SourceProfile {
	t.Helper()
	profile, ok := PyTorchProfile(name)
	if !ok {
		t.Fatalf("missing PyTorch profile %q", name)
	}
	return profile
}
