package npm

import (
	"context"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestStaticArchiveInstallActionFactsRequireValidArchive(t *testing.T) {
	for _, test := range []struct {
		name, manifest string
		extra          []tarEntry
		want           string
	}{
		{"plain", `{"name":"tiny","version":"1.2.3"}`, nil, domain.EvidenceHostInstallActionNotRequired},
		{"ordinary-script", `{"name":"tiny","version":"1.2.3","scripts":{"test":"node test.js"}}`, nil, domain.EvidenceHostInstallActionNotRequired},
		{"preinstall", `{"name":"tiny","version":"1.2.3","scripts":{"preinstall":"echo check"}}`, nil, domain.EvidenceHostInstallActionRequired},
		{"install-empty", `{"name":"tiny","version":"1.2.3","scripts":{"install":""}}`, nil, domain.EvidenceHostInstallActionRequired},
		{"postinstall", `{"name":"tiny","version":"1.2.3","scripts":{"postinstall":"echo check"}}`, nil, domain.EvidenceHostInstallActionRequired},
		{"prepublish", `{"name":"tiny","version":"1.2.3","scripts":{"prepublish":"echo check"}}`, nil, domain.EvidenceHostInstallActionRequired},
		{"preprepare", `{"name":"tiny","version":"1.2.3","scripts":{"preprepare":"echo check"}}`, nil, domain.EvidenceHostInstallActionRequired},
		{"prepare", `{"name":"tiny","version":"1.2.3","scripts":{"prepare":"echo check"}}`, nil, domain.EvidenceHostInstallActionRequired},
		{"implicit-gyp", `{"name":"tiny","version":"1.2.3"}`, []tarEntry{{name: "package/binding.gyp", body: "{}"}}, domain.EvidenceHostInstallActionRequired},
		{"gyp-opt-out-conservative", `{"name":"tiny","version":"1.2.3","gypfile":false}`, []tarEntry{{name: "package/binding.gyp", body: "{}"}}, domain.EvidenceHostInstallActionRequired},
		{"nested-gyp", `{"name":"tiny","version":"1.2.3"}`, []tarEntry{{name: "package/examples/binding.gyp", body: "{}"}}, domain.EvidenceHostInstallActionNotRequired},
		{"invalid-identity", `{"name":"other","version":"1.2.3"}`, nil, ""},
		{"invalid-manifest", `{"name":"tiny","version":"1.2.3","scripts":{"install":42}}`, nil, ""},
		{"late-archive-violation", `{"name":"tiny","version":"1.2.3"}`, []tarEntry{{name: "../escape", body: "{}"}}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := canonicalRoot(t)
			artifact := writeArtifact(t, root, append([]tarEntry{{name: "package/package.json", body: test.manifest}}, test.extra...))
			inspector, err := NewInspector(root)
			if err != nil {
				t.Fatal(err)
			}
			report, err := inspector.Inspect(context.Background(), artifact)
			if err != nil {
				t.Fatal(err)
			}
			facts := 0
			for _, evidence := range report.Evidence() {
				if evidence.Kind() != domain.EvidenceHostInstallActionRequired && evidence.Kind() != domain.EvidenceHostInstallActionNotRequired {
					continue
				}
				facts++
				if test.want == "" || evidence.Kind() != test.want || evidence.Identity() != artifact.Identity() || evidence.Digest() != artifact.Digest() || evidence.CheckID().String() != "npm-static-archive" || evidence.ID().String() != "npm-host-install-action" {
					t.Fatalf("invalid installation fact: %v", evidence)
				}
			}
			if (test.want == "" && (facts != 0 || len(report.Findings()) == 0)) || (test.want != "" && (facts != 1 || len(report.Findings()) != 0)) {
				t.Fatalf("facts=%d findings=%v", facts, report.Findings())
			}
		})
	}
}

func TestCompositePreservesArchiveFactAndDynamicRequirement(t *testing.T) {
	root := canonicalRoot(t)
	artifact := writeArtifact(t, root, []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3","scripts":{"install":"echo check"}}`}})
	static, err := NewInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	dynamic := completedReport(t, artifact, "npm-dynamic-lifecycle", "dynamic-evidence")
	composite, err := NewCompositeInspector(static, reportInspector{report: dynamic})
	if err != nil {
		t.Fatal(err)
	}
	report, err := composite.Inspect(context.Background(), artifact)
	if err != nil || len(report.Executions()) != 2 || len(report.Evidence()) != 3 || len(report.Findings()) != 0 {
		t.Fatalf("checks/facts were lost: %v %v", report, err)
	}
}
