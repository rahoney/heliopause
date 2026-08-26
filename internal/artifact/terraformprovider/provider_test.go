package terraformprovider

import "testing"

func TestProviderReferenceAndPackageBinding(t *testing.T) {
	reference, err := ParseReference("hashicorp/aws@5.50.0")
	if err != nil || reference.Source() != Source() {
		t.Fatalf("reference = %#v, error = %v", reference, err)
	}
	platform := Platform{OS: "linux", Arch: "amd64"}
	body := []byte(`{"protocol":"5.0","os":"linux","arch":"amd64","filename":"terraform-provider-aws_v5.50.0_linux_amd64.zip","download_url":"https://releases.hashicorp.com/terraform-provider-aws/5.50.0/terraform-provider-aws_5.50.0_linux_amd64.zip","shasums_url":"https://releases.hashicorp.com/terraform-provider-aws/5.50.0/terraform-provider-aws_5.50.0_SHA256SUMS","shasums_signature_url":"https://releases.hashicorp.com/terraform-provider-aws/5.50.0/terraform-provider-aws_5.50.0_SHA256SUMS.sig","shasum":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","signing_keys":[{"key_id":"ABC123"}]}`)
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
