// Package terraformprovider verifies signed provider checksum documents. A
// registry key ID/source label, a project lock and a valid community signature
// do not supply publisher trust or permission to install a provider.
package terraformprovider

import (
	"bytes"
	"crypto"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	openpgp "github.com/ProtonMail/go-crypto/openpgp/v2"
	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
)

const OfficialFingerprint = "C874011F0AB405110D02105534365D9472D7468F"
const PartnerRootFingerprint = "51890657C5ACDB4B823086567D72D4268E4660FC"

// The certificates are public data from the reviewed official security and
// upstream public-key sources, not Terraform implementation code.
//
//go:embed keys/hashicorp.asc
var officialCertificate []byte

//go:embed keys/partners.asc
var partnerCertificate []byte

type TrustTier string

const (
	TrustOfficial  TrustTier = "HASHICORP_OFFICIAL"
	TrustPartner   TrustTier = "RECOGNIZED_PARTNER"
	TrustCommunity TrustTier = "COMMUNITY_REQUIRES_REVIEW"
)

type SignatureBinding struct {
	Fingerprint        string
	SigningFingerprint string
	Trust              TrustTier
	ChecksumSHA256     string
}

// VerifySignedChecksums is the sole provider signer classification table.
// Official: pinned full primary fingerprint + current authoritative certificate.
// Partner: current provider certificate + detached endorsement over its exact
// decoded certificate bytes by the pinned partner authority. Community stays
// incomplete for automatic approval. Namespace is never a vendor trust list.
func VerifySignedChecksums(artifact artifactterraform.ProviderArtifact, checksums, signature []byte, now time.Time) (SignatureBinding, error) {
	if now.IsZero() || len(checksums) == 0 || len(checksums) > 1<<20 || len(artifact.SigningKeys) == 0 || len(artifact.SigningKeys) > 8 {
		return SignatureBinding{}, errors.New("terraform signed package verification input is invalid")
	}
	checksum, err := ChecksumForPackage(checksums, artifact.Filename)
	if err != nil || checksum != artifact.SHA256 {
		return SignatureBinding{}, errors.New("terraform signed checksum differs from exact registry package")
	}
	config := &packet.Config{Time: func() time.Time { return now }, MinRSABits: 2048}
	official, _, err := readCertificate(officialCertificate, config)
	if err != nil || fingerprint(official.PrimaryKey) != OfficialFingerprint {
		return SignatureBinding{}, errors.New("terraform official trust anchor is invalid")
	}
	partnerAuthority, _, err := readCertificate(partnerCertificate, config)
	if err != nil || fingerprint(partnerAuthority.PrimaryKey) != PartnerRootFingerprint {
		return SignatureBinding{}, errors.New("terraform partner trust anchor is invalid")
	}
	var verified []SignatureBinding
	seen := map[string]bool{}
	for _, declaration := range artifact.SigningKeys {
		entity, certificateBody, err := parseCertificate([]byte(declaration.ASCIIArmor))
		if err != nil {
			return SignatureBinding{}, errors.New("terraform registry signing certificate is invalid")
		}
		primary := fingerprint(entity.PrimaryKey)
		if seen[primary] || declaration.KeyID != fmt.Sprintf("%016X", entity.PrimaryKey.KeyId) {
			return SignatureBinding{}, errors.New("terraform signing certificate identity is ambiguous")
		}
		seen[primary] = true
		tier := TrustCommunity
		current := entity
		if primary == OfficialFingerprint {
			// Registry-supplied self-signatures cannot replace the independently
			// pinned current certificate or suppress expiration/revocation.
			if entity.Revoked(config.Now()) {
				return SignatureBinding{}, errors.New("terraform registry official certificate carries a valid revocation")
			}
			current = official
			tier = TrustOfficial
		} else if declaration.TrustSignature != "" {
			if _, err := verifyDetached(partnerAuthority, certificateBody, []byte(declaration.TrustSignature), config); err != nil {
				return SignatureBinding{}, errors.New("terraform partner certificate endorsement is invalid")
			}
			tier = TrustPartner
		}
		if _, err := current.VerifyPrimaryKey(config.Now(), config); err != nil {
			return SignatureBinding{}, errors.New("terraform signing certificate is not currently valid")
		}
		signing, err := verifyDetachedWithCurrent(entity, current, checksums, signature, config)
		if err != nil {
			continue
		}
		if tier == TrustOfficial && !strings.HasPrefix(artifact.Reference.Locator(), "hashicorp/") {
			return SignatureBinding{}, errors.New("terraform official signer namespace does not match")
		}
		if strings.HasPrefix(artifact.Reference.Locator(), "hashicorp/") && tier != TrustOfficial {
			return SignatureBinding{}, errors.New("terraform official namespace has a different signer")
		}
		verified = append(verified, SignatureBinding{primary, signing, tier, checksum})
	}
	if len(verified) != 1 {
		return SignatureBinding{}, errors.New("terraform checksum signature is missing, invalid or ambiguous")
	}
	return verified[0], nil
}

func ChecksumForPackage(body []byte, filename string) (string, error) {
	if len(body) == 0 || len(body) > 1<<20 || !bytes.HasSuffix(body, []byte("\n")) {
		return "", errors.New("terraform checksum document is incomplete")
	}
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	if len(lines) > 8192 {
		return "", errors.New("terraform checksum document exceeds bound")
	}
	seen := map[string]bool{}
	selected := ""
	for _, line := range lines {
		if len(line) < 67 || line[64:66] != "  " || len(line) > 1024 {
			return "", errors.New("terraform checksum declaration is invalid")
		}
		digest, name := line[:64], line[66:]
		decoded, err := hex.DecodeString(digest)
		if err != nil || len(decoded) != 32 || digest != strings.ToLower(digest) || name == "" || strings.ContainsAny(name, "/\\\x00\r\t ") || seen[name] {
			return "", errors.New("terraform checksum declaration is ambiguous")
		}
		seen[name] = true
		if name == filename {
			selected = digest
		}
	}
	if selected == "" {
		return "", errors.New("terraform exact provider package is absent from signed checksums")
	}
	return selected, nil
}

func fingerprint(key *packet.PublicKey) string {
	return strings.ToUpper(hex.EncodeToString(key.Fingerprint))
}

func decodeArmor(body []byte, kind string, maximum int) ([]byte, error) {
	if len(body) == 0 || len(body) > maximum || !bytes.HasPrefix(body, []byte("-----BEGIN "+kind+"-----")) || !bytes.HasSuffix(bytes.TrimSpace(body), []byte("-----END "+kind+"-----")) || bytes.Count(body, []byte("-----BEGIN")) != 1 {
		return nil, errors.New("openpgp armor is invalid or ambiguous")
	}
	block, err := armor.Decode(bytes.NewReader(body))
	if err != nil || block.Type != kind {
		return nil, errors.New("openpgp armor type is invalid")
	}
	decoded, err := io.ReadAll(io.LimitReader(block.Body, int64(maximum)+1))
	if err != nil || len(decoded) == 0 || len(decoded) > maximum {
		return nil, errors.New("openpgp armor body exceeds bounds")
	}
	return decoded, nil
}

func readCertificate(body []byte, config *packet.Config) (*openpgp.Entity, []byte, error) {
	entity, decoded, err := parseCertificate(body)
	if err != nil {
		return nil, nil, err
	}
	if _, err := entity.VerifyPrimaryKey(config.Now(), config); err != nil {
		return nil, nil, errors.New("openpgp primary certificate is not currently valid")
	}
	return entity, decoded, nil
}

func parseCertificate(body []byte) (*openpgp.Entity, []byte, error) {
	decoded, err := decodeArmor(body, "PGP PUBLIC KEY BLOCK", 128<<10)
	if err != nil {
		return nil, nil, err
	}
	reader := bytes.NewReader(decoded)
	for count := 0; reader.Len() > 0; count++ {
		if count >= 256 {
			return nil, nil, errors.New("openpgp certificate packet count exceeds bound")
		}
		p, err := packet.Read(reader)
		if err != nil {
			return nil, nil, errors.New("openpgp certificate packet is invalid")
		}
		switch p.(type) {
		case *packet.PublicKey, *packet.UserId, *packet.UserAttribute, *packet.Signature:
		default:
			return nil, nil, errors.New("openpgp certificate contains unsupported packet")
		}
	}
	entities, err := openpgp.ReadKeyRing(bytes.NewReader(decoded))
	if err != nil || len(entities) != 1 || entities[0].PrivateKey != nil {
		return nil, nil, errors.New("openpgp public certificate identity is ambiguous")
	}
	entity := entities[0]
	return entity, decoded, nil
}

func verifyDetached(entity *openpgp.Entity, body, signature []byte, config *packet.Config) (string, error) {
	return verifyDetachedWithCurrent(entity, entity, body, signature, config)
}

// Historical bindings must be valid at signature creation; the same full
// primary and signing key must also be valid now in the current authority's
// certificate. Neither expiry nor revocation is ignored at either boundary.
func verifyDetachedWithCurrent(entity, current *openpgp.Entity, body, signature []byte, config *packet.Config) (string, error) {
	if len(signature) == 0 || len(signature) > 16<<10 {
		return "", errors.New("openpgp detached signature exceeds bounds")
	}
	if bytes.HasPrefix(signature, []byte("-----BEGIN")) {
		var err error
		signature, err = decodeArmor(signature, "PGP SIGNATURE", 16<<10)
		if err != nil {
			return "", err
		}
	}
	reader := bytes.NewReader(signature)
	p, err := packet.Read(reader)
	if err != nil || reader.Len() != 0 {
		return "", errors.New("openpgp detached signature is invalid or ambiguous")
	}
	sig, ok := p.(*packet.Signature)
	if !ok || sig.IssuerKeyId == nil || sig.SigType != packet.SigTypeBinary || sig.CreationTime.After(config.Now()) || (sig.Hash != crypto.SHA256 && sig.Hash != crypto.SHA384 && sig.Hash != crypto.SHA512) {
		return "", errors.New("openpgp signature type, hash or time is unsupported")
	}
	key, valid := current.SigningKeyById(config.Now(), *sig.IssuerKeyId, config)
	if !valid || fingerprint(current.PrimaryKey) != fingerprint(entity.PrimaryKey) {
		return "", errors.New("openpgp signing key is expired, revoked or unbound")
	}
	historical, valid := entity.SigningKeyById(sig.CreationTime, *sig.IssuerKeyId, config)
	if !valid || fingerprint(historical.PublicKey) != fingerprint(key.PublicKey) {
		return "", errors.New("openpgp historical and current signing keys differ")
	}
	_, signer, err := openpgp.VerifyDetachedSignature(openpgp.EntityList{entity}, bytes.NewReader(body), bytes.NewReader(signature), config)
	if err != nil || signer != entity {
		return "", errors.New("openpgp detached signature does not verify")
	}
	return fingerprint(key.PublicKey), nil
}
