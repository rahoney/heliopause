package terraformprovider

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

// These are bounded, frozen actual public registry responses, not a synthetic
// signing_keys schema or proof that the declared keys are trusted.
func TestProviderActualRegistryMetadata(t *testing.T) {
	for _, fixture := range []struct{ file, reference string }{
		{"hashicorp-random-3.7.2.json", "hashicorp/random@3.7.2"},
		{"integrations-github-6.6.0.json", "integrations/github@6.6.0"},
	} {
		t.Run(fixture.reference, func(t *testing.T) {
			body, err := os.ReadFile("testdata/" + fixture.file)
			if err != nil {
				t.Fatal(err)
			}
			reference, err := ParseReference(fixture.reference)
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := ParsePackageResponseWithAllowedHosts(reference, body, Platform{OS: "linux", Arch: "amd64"}, map[string]bool{"releases.hashicorp.com": true, "github.com": true})
			if err != nil || len(artifact.SignerKeyIDs) != 1 || artifact.SHA256 == "" {
				t.Fatalf("actual registry metadata: %v", err)
			}
		})
	}
}

func TestProviderRejectsRegistryBindingDrift(t *testing.T) {
	body, err := os.ReadFile("testdata/hashicorp-random-3.7.2.json")
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := ParseReference("hashicorp/random@3.7.2")
	for _, fault := range []struct{ name, old, replacement string }{
		{"platform", `"arch":"amd64"`, `"arch":"arm64"`},
		{"filename", `"filename":"terraform-provider-random_3.7.2_linux_amd64.zip"`, `"filename":"terraform-provider-random_3.7.1_linux_amd64.zip"`},
		{"mirror", "releases.hashicorp.com", "mirror.example"},
		{"port", "releases.hashicorp.com", "releases.hashicorp.com:443"},
		{"version path", "/3.7.2/", "/3.7.1/"},
		{"user", "https://releases", "https://user@releases"},
		{"escaped path", "/terraform-provider-random/", "/%74erraform-provider-random/"},
		{"duplicate", `"os":"linux"`, `"os":"linux","os":"linux"`},
		{"case alias", `"os":"linux"`, `"os":"linux","OS":"linux"`},
		{"old schema", `"protocols":["5.0"]`, `"protocol":"5.0"`},
		{"ambiguous protocols", `"protocols":["5.0"]`, `"protocols":["5.0","5.1"]`},
		{"missing certificate", `PGP PUBLIC`, `PGP PRIVATE`},
	} {
		t.Run(fault.name, func(t *testing.T) {
			mutated := strings.ReplaceAll(string(body), fault.old, fault.replacement)
			if mutated == string(body) {
				t.Fatal("fixture mutation did not apply")
			}
			if _, err := ParsePackageResponse(reference, []byte(mutated), Platform{OS: "linux", Arch: "amd64"}); err == nil {
				t.Fatal("accepted metadata drift")
			}
		})
	}
	partner, err := os.ReadFile("testdata/integrations-github-6.6.0.json")
	if err != nil {
		t.Fatal(err)
	}
	partnerRef, _ := ParseReference("integrations/github@6.6.0")
	for _, body := range []string{
		strings.ReplaceAll(string(partner), "/integrations/", "/attacker/"),
		strings.ReplaceAll(string(partner), "/v6.6.0/", "/v6.6.1/"),
		strings.ReplaceAll(string(partner), "SHA256SUMS.sig", "SHA256SUMS.sig?mirror=yes"),
	} {
		if _, err := ParsePackageResponse(partnerRef, []byte(body), Platform{OS: "linux", Arch: "amd64"}); err == nil {
			t.Fatal("accepted release path drift")
		}
	}
}

func TestProviderVersionAndDiscoveryAreUnambiguous(t *testing.T) {
	platform := Platform{OS: "linux", Arch: "amd64"}
	entry := `{"version":"3.7.2","protocols":["5.0"],"platforms":[{"os":"linux","arch":"amd64"}]}`
	if err := ParseVersionResponse([]byte(`{"versions":[`+entry+`]}`), "3.7.2", platform); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{"versions":[` + entry + `,` + entry + `]}`, `{"versions":[` + entry + `],"versions":[]}`, `{"versions":[` + strings.ReplaceAll(entry, "amd64", "arm64") + `]}`} {
		if err := ParseVersionResponse([]byte(body), "3.7.2", platform); err == nil {
			t.Fatal("accepted ambiguous or unavailable version")
		}
	}
	if err := ValidateDiscovery([]byte(`{"providers.v1":"/v1/providers/"}`)); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{"providers.v1":"https://mirror.example/v1/providers/"}`, `{"providers.v1":"/v1/providers/","providers.v1":"/v1/providers/"}`, `{"providers.v1":"/v1/providers"}`} {
		if ValidateDiscovery([]byte(body)) == nil {
			t.Fatal("accepted alternate discovery")
		}
	}
}

func TestActualVersionListDoesNotApplyCurrentProtocolsToOtherVersions(t *testing.T) {
	body, err := os.ReadFile("testdata/hashicorp-random-versions.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := ParseVersionResponse(body, "3.7.2", Platform{OS: "linux", Arch: "amd64"}); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"3.7.0-alpha1", "1.3.1"} {
		if err := ParseVersionResponse(body, version, Platform{OS: "linux", Arch: "amd64"}); err == nil {
			t.Fatal("selected unavailable protocol declaration accepted")
		}
	}
}

func TestResolverRejectsNonCanonicalEndpoint(t *testing.T) {
	client := &http.Client{}
	for _, raw := range []string{"https://registry.terraform.io/path", "https://user@registry.terraform.io"} {
		endpoint, _ := url.Parse(raw)
		if _, err := newResolver(endpoint, client); err == nil {
			t.Fatalf("accepted resolver endpoint %q", raw)
		}
	}
}

func TestProviderReferenceAndPackageBinding(t *testing.T) {
	reference, err := ParseReference("hashicorp/aws@5.50.0")
	if err != nil || reference.Source() != Source() {
		t.Fatalf("reference = %#v, error = %v", reference, err)
	}
	platform := Platform{OS: "linux", Arch: "amd64"}
	body := []byte(`{"protocols":["5.0"],"os":"linux","arch":"amd64","filename":"terraform-provider-aws_5.50.0_linux_amd64.zip","download_url":"https://releases.hashicorp.com/terraform-provider-aws/5.50.0/terraform-provider-aws_5.50.0_linux_amd64.zip","shasums_url":"https://releases.hashicorp.com/terraform-provider-aws/5.50.0/terraform-provider-aws_5.50.0_SHA256SUMS","shasums_signature_url":"https://releases.hashicorp.com/terraform-provider-aws/5.50.0/terraform-provider-aws_5.50.0_SHA256SUMS.sig","shasum":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","signing_keys":{"gpg_public_keys":[{"key_id":"34365D9472D7468F","ascii_armor":"-----BEGIN PGP PUBLIC KEY BLOCK-----synthetic parser declaration, not a verified certificate"}]}}`)
	artifact, err := ParsePackageResponse(reference, body, platform)
	if err != nil || artifact.SHA256 == "" || len(artifact.SignerKeyIDs) != 1 {
		t.Fatalf("artifact = %#v, error = %v", artifact, err)
	}
	if err := VerifyLockHash([]string{"zh:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, artifact.SHA256); err != nil {
		t.Fatal(err)
	}
	graph, err := BuildLockedGraph(artifact)
	if err != nil || len(graph.Nodes()) != 1 || graph.Nodes()[0].Artifact().Identity().Source() != Source() {
		t.Fatalf("provider graph = %#v, error = %v", graph, err)
	}
}

func TestProviderRejectsUntrustedOrUnsignedBinding(t *testing.T) {
	reference, _ := ParseReference("hashicorp/aws@5.50.0")
	platform := Platform{OS: "linux", Arch: "amd64"}
	body := `{"os":"linux","arch":"amd64","filename":"provider.zip","download_url":"https://evil.example/provider.zip","shasums_url":"https://evil.example/SHA256SUMS","shasums_signature_url":"https://evil.example/SHA256SUMS.sig","shasum":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","signing_keys":[]}`
	if _, err := ParsePackageResponse(reference, []byte(body), platform); err == nil {
		t.Fatal("accepted unsigned/untrusted Provider response")
	}
	for _, value := range []string{"hashicorp/aws", "hashicorp/aws@latest", "hashicorp/aws@5.50"} {
		if _, err := ParseReference(value); err == nil {
			t.Fatalf("accepted invalid Provider reference %q", value)
		}
	}
}
