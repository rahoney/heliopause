package domain_test

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestArtifactIdentityFlow(t *testing.T) {
	t.Parallel()

	source := mustSource(t, "fixture")
	reference, err := domain.NewArtifactReference(source, "safe@latest")
	if err != nil {
		t.Fatalf("NewArtifactReference() error = %v", err)
	}
	identity := mustIdentity(t, source, "safe", "1.0.0", "default")
	digest, err := domain.NewSHA256Digest(strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("NewSHA256Digest() error = %v", err)
	}
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "fixture-content:safe", 42)
	if err != nil {
		t.Fatalf("NewAcquiredArtifact() error = %v", err)
	}

	if reference.Source() != source || reference.Locator() != "safe@latest" {
		t.Fatalf("reference = %#v", reference)
	}
	if artifact.Identity() != identity || artifact.Digest() != digest || artifact.ContentHandle() != "fixture-content:safe" || artifact.SizeBytes() != 42 {
		t.Fatalf("artifact getters returned inconsistent subject")
	}
}

func TestArtifactValueValidation(t *testing.T) {
	t.Parallel()

	validSource := mustSource(t, "fixture")
	validIdentity := mustIdentity(t, validSource, "safe", "1.0.0", "default")
	validDigest, err := domain.NewSHA256Digest(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]func() error{
		"uppercase source": func() error { _, err := domain.NewSourceID("Fixture"); return err },
		"empty locator":    func() error { _, err := domain.NewArtifactReference(validSource, ""); return err },
		"control locator":  func() error { _, err := domain.NewArtifactReference(validSource, "safe\nother"); return err },
		"empty variant": func() error {
			_, err := domain.NewResolvedArtifactIdentity(validSource, "safe", "1.0.0", "")
			return err
		},
		"uppercase digest": func() error { _, err := domain.NewSHA256Digest(strings.Repeat("A", 64)); return err },
		"short digest":     func() error { _, err := domain.NewSHA256Digest("ab"); return err },
		"empty handle":     func() error { _, err := domain.NewAcquiredArtifact(validIdentity, validDigest, "", 0); return err },
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := test(); err == nil {
				t.Fatal("error = nil")
			}
		})
	}
}

func mustSource(t *testing.T, value string) domain.SourceID {
	t.Helper()
	source, err := domain.NewSourceID(value)
	if err != nil {
		t.Fatalf("NewSourceID() error = %v", err)
	}
	return source
}

func mustIdentity(t *testing.T, source domain.SourceID, name, version, variant string) domain.ResolvedArtifactIdentity {
	t.Helper()
	identity, err := domain.NewResolvedArtifactIdentity(source, name, version, variant)
	if err != nil {
		t.Fatalf("NewResolvedArtifactIdentity() error = %v", err)
	}
	return identity
}
