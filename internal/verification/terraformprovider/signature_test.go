package terraformprovider

import (
	"bytes"
	"crypto"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	openpgp "github.com/ProtonMail/go-crypto/openpgp/v2"
	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
)

func signedFixture(t *testing.T, name, reference string) (artifactterraform.ProviderArtifact, []byte, []byte) {
	t.Helper()
	read := func(suffix string) []byte {
		body, err := os.ReadFile("testdata/" + name + "." + suffix)
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	ref, err := artifactterraform.ParseReference(reference)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := artifactterraform.ParsePackageResponse(ref, read("json"), artifactterraform.Platform{OS: "linux", Arch: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	return artifact, read("sums"), read("sig")
}

func TestActualOfficialAndPartnerSignedChecksums(t *testing.T) {
	for _, fixture := range []struct {
		name, reference string
		tier            TrustTier
	}{
		{"hashicorp-random-3.7.2", "hashicorp/random@3.7.2", TrustOfficial},
		{"integrations-github-6.6.0", "integrations/github@6.6.0", TrustPartner},
	} {
		t.Run(fixture.reference, func(t *testing.T) {
			artifact, sums, sig := signedFixture(t, fixture.name, fixture.reference)
			binding, err := VerifySignedChecksums(artifact, sums, sig, time.Now().UTC())
			if err != nil || binding.Trust != fixture.tier || len(binding.Fingerprint) != 40 || len(binding.SigningFingerprint) != 40 || binding.ChecksumSHA256 != artifact.SHA256 {
				t.Fatalf("actual signed binding = %+v, %v", binding, err)
			}
		})
	}
}

func TestSignedPackageRejectsTamperingAndAmbiguity(t *testing.T) {
	artifact, sums, sig := signedFixture(t, "hashicorp-random-3.7.2", "hashicorp/random@3.7.2")
	now := time.Now().UTC()
	for _, fault := range []string{"checksum bytes", "registry checksum", "signature", "extra signature", "truncated", "signer ID", "private armor", "duplicate signer", "unsigned", "filename", "future clock"} {
		t.Run(fault, func(t *testing.T) {
			candidate := artifact
			candidate.SigningKeys = append([]artifactterraform.SigningKey(nil), artifact.SigningKeys...)
			checksum := append([]byte(nil), sums...)
			signature := append([]byte(nil), sig...)
			clock := now
			switch fault {
			case "checksum bytes":
				checksum[0] ^= 1
			case "registry checksum":
				candidate.SHA256 = strings.Repeat("a", 64)
			case "signature":
				signature[len(signature)-1] ^= 1
			case "extra signature":
				signature = append(signature, sig...)
			case "truncated":
				signature = signature[:len(signature)-1]
			case "signer ID":
				candidate.SigningKeys[0].KeyID = "0000000000000000"
			case "private armor":
				candidate.SigningKeys[0].ASCIIArmor = strings.ReplaceAll(candidate.SigningKeys[0].ASCIIArmor, "PGP PUBLIC KEY", "PGP PRIVATE KEY")
			case "duplicate signer":
				candidate.SigningKeys = append(candidate.SigningKeys, candidate.SigningKeys[0])
			case "unsigned":
				signature = nil
			case "filename":
				candidate.Filename = "different.zip"
			case "future clock":
				clock = time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
			}
			if binding, err := VerifySignedChecksums(candidate, checksum, signature, clock); err == nil {
				t.Fatalf("accepted %s: %+v", fault, binding)
			}
		})
	}
	partner, partnerSums, partnerSig := signedFixture(t, "integrations-github-6.6.0", "integrations/github@6.6.0")
	partner.SigningKeys[0].TrustSignature = strings.Replace(partner.SigningKeys[0].TrustSignature, "\n", "\nX", 1)
	if _, err := VerifySignedChecksums(partner, partnerSums, partnerSig, now); err == nil {
		t.Fatal("accepted tampered partner endorsement")
	}
}

func TestCommunitySignatureCannotSupplyPublisherTrust(t *testing.T) {
	artifact, sums, _ := signedFixture(t, "hashicorp-random-3.7.2", "hashicorp/random@3.7.2")
	now := time.Now().UTC().Truncate(time.Second)
	config := &packet.Config{Time: func() time.Time { return now }, Algorithm: packet.PubKeyAlgoEdDSA, DefaultHash: crypto.SHA256}
	entity, err := openpgp.NewEntity("Synthetic community fixture", "", "test@example.invalid", config)
	if err != nil {
		t.Fatal(err)
	}
	var public, signature bytes.Buffer
	if err := entity.Serialize(&public); err != nil {
		t.Fatal(err)
	}
	// An independently generated valid public certificate/signature remains
	// community. Namespace labels and unverified project pins cannot make ALLOW.
	var armored bytes.Buffer
	writer, err := armor.Encode(&armored, "PGP PUBLIC KEY BLOCK", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(public.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	armorBytes := armored.Bytes()
	if err := openpgp.DetachSign(&signature, []*openpgp.Entity{entity}, bytes.NewReader(sums), config); err != nil {
		t.Fatal(err)
	}
	ref, _ := artifactterraform.ParseReference("community/random@3.7.2")
	artifact.Reference = ref
	artifact.SigningKeys = []artifactterraform.SigningKey{{KeyID: strings.ToUpper(entity.PrimaryKey.KeyIdString()), ASCIIArmor: string(armorBytes)}}
	binding, err := VerifySignedChecksums(artifact, sums, signature.Bytes(), now)
	if err != nil || binding.Trust != TrustCommunity {
		t.Fatalf("community result=%+v %v", binding, err)
	}
}

func TestOfficialHistoricalBindingRequiresCurrentAuthority(t *testing.T) {
	artifact, sums, sig := signedFixture(t, "hashicorp-random-3.7.2", "hashicorp/random@3.7.2")
	now := time.Now().UTC()
	config := &packet.Config{Time: func() time.Time { return now }, MinRSABits: 2048}
	historical, _, err := parseCertificate([]byte(artifact.SigningKeys[0].ASCIIArmor))
	if err != nil || fingerprint(historical.PrimaryKey) != OfficialFingerprint {
		t.Fatalf("historical certificate: %v", err)
	}
	selfSignature, err := historical.VerifyPrimaryKey(now, config)
	if err == nil || selfSignature == nil || !historical.PrimaryKey.KeyExpired(selfSignature, now) {
		t.Fatal("frozen registry certificate no longer reproduces current expiry")
	}
	current, _, err := readCertificate(officialCertificate, config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifyDetachedWithCurrent(historical, current, sums, sig, config); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyDetachedWithCurrent(historical, historical, sums, sig, config); err == nil {
		t.Fatal("expired current authority accepted")
	}
	later := &packet.Config{Time: func() time.Time { return time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC) }, MinRSABits: 2048}
	if _, err := verifyDetachedWithCurrent(historical, current, sums, sig, later); err == nil {
		t.Fatal("future expired official authority accepted")
	}
}

func TestDetachedSignatureRejectsRevokedAndExpiredCurrentKeys(t *testing.T) {
	created := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	config := &packet.Config{Time: func() time.Time { return created }, Algorithm: packet.PubKeyAlgoEdDSA, DefaultHash: crypto.SHA256, KeyLifetimeSecs: 30}
	entity, err := openpgp.NewEntity("Synthetic validity fixture", "", "test@example.invalid", config)
	if err != nil {
		t.Fatal(err)
	}
	message := []byte("bounded message")
	var signature bytes.Buffer
	if err := openpgp.DetachSign(&signature, []*openpgp.Entity{entity}, bytes.NewReader(message), config); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyDetached(entity, message, signature.Bytes(), config); err != nil {
		t.Fatal(err)
	}
	expired := &packet.Config{Time: func() time.Time { return created.Add(time.Minute) }}
	if _, err := verifyDetached(entity, message, signature.Bytes(), expired); err == nil {
		t.Fatal("expired current key accepted")
	}
	if err := entity.Revoke(packet.KeyCompromised, "synthetic test revocation", config); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyDetached(entity, message, signature.Bytes(), config); err == nil {
		t.Fatal("revoked key accepted")
	}
}
