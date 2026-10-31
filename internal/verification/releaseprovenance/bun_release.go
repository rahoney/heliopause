package releaseprovenance

import (
	"bufio"
	"bytes"
	"context"
	"crypto"
	_ "embed"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	openpgp "github.com/ProtonMail/go-crypto/openpgp/v2"
)

const (
	BunPublisherFingerprint = "F3DCC08A8572C0749B3E18888EAB4D40A7B22B59"
	BunArchiveFilename      = "bun-linux-x64.zip"
	BunArchiveLimit         = 64 << 20
	bunSignedChecksumsLimit = 64 << 10
)

// The reviewed exact-tag official Dockerfile embeds this public vendor key.
// This authority is specific to Bun checksum signatures, not other publishers.
//
//go:embed keys/bun.asc
var bunCertificate []byte

// BunChecksumReceipt authenticates one archive digest under Bun's pinned
// publisher key. It asserts no release version, build source, SLSA provenance,
// native ABI, runtime safety, inspection, Policy or execution approval.
type BunChecksumReceipt struct {
	archive, document, checksums, signer string
	signedAt                             time.Time
}

func (r BunChecksumReceipt) ArchiveSHA256() string        { return r.archive }
func (r BunChecksumReceipt) SignedDocumentSHA256() string { return r.document }
func (r BunChecksumReceipt) ChecksumsSHA256() string      { return r.checksums }
func (r BunChecksumReceipt) SigningFingerprint() string   { return r.signer }
func (r BunChecksumReceipt) SignatureTime() time.Time     { return r.signedAt }

// VerifyBunChecksums verifies the complete single cleartext signature under
// the fixed official key, then selects the exact Linux x64 archive filename.
// expectedArchiveSHA256 is frozen independent release metadata, not a digest
// inferred from the archive supplied for verification.
func VerifyBunChecksums(ctx context.Context, signed []byte, expectedArchiveSHA256 string, now time.Time) (BunChecksumReceipt, error) {
	if ctx == nil || now.IsZero() || len(signed) == 0 || len(signed) > bunSignedChecksumsLimit || !hexSHA256.MatchString(expectedArchiveSHA256) {
		return BunChecksumReceipt{}, errors.New("bun signed checksum inputs are invalid or excessive")
	}
	if err := ctx.Err(); err != nil {
		return BunChecksumReceipt{}, errors.Join(err, context.Cause(ctx))
	}
	if !bytes.HasPrefix(signed, []byte("-----BEGIN PGP SIGNED MESSAGE-----")) || bytes.Count(signed, []byte("-----BEGIN PGP SIGNED MESSAGE-----")) != 1 || bytes.Count(signed, []byte("-----BEGIN PGP SIGNATURE-----")) != 1 {
		return BunChecksumReceipt{}, errors.New("bun signed checksum envelope is ambiguous")
	}
	block, rest := clearsign.Decode(signed)
	if block == nil || len(rest) != 0 || block.ArmoredSignature == nil || block.ArmoredSignature.Type != "PGP SIGNATURE" || len(block.Headers) != 1 || len(block.Headers.Values("Hash")) != 1 {
		return BunChecksumReceipt{}, errors.New("bun signed checksum envelope is invalid or has trailing data")
	}
	signature, err := io.ReadAll(io.LimitReader(block.ArmoredSignature.Body, 16<<10+1))
	if err != nil || len(signature) == 0 || len(signature) > 16<<10 {
		return BunChecksumReceipt{}, errors.New("bun checksum signature is invalid or excessive")
	}
	reader := bytes.NewReader(signature)
	value, err := packet.Read(reader)
	sig, ok := value.(*packet.Signature)
	if err != nil || !ok || reader.Len() != 0 || sig.SigType != packet.SigTypeText || sig.IssuerKeyId == nil || sig.CreationTime.After(now) || sig.SigExpired(now) {
		return BunChecksumReceipt{}, errors.New("bun checksum signature type, time or coverage is invalid")
	}
	hashHeader := map[crypto.Hash]string{crypto.SHA256: "SHA256", crypto.SHA384: "SHA384", crypto.SHA512: "SHA512"}[sig.Hash]
	if hashHeader == "" || block.Headers.Get("Hash") != hashHeader {
		return BunChecksumReceipt{}, errors.New("bun checksum signature hash is unsupported or inconsistent")
	}
	config := &packet.Config{Time: func() time.Time { return now }, MinRSABits: 2048}
	keys, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(bunCertificate))
	if err != nil || len(keys) != 1 || keys[0].PrivateKey != nil || bunFingerprint(keys[0].PrimaryKey) != BunPublisherFingerprint {
		return BunChecksumReceipt{}, errors.New("bun pinned publisher certificate is invalid")
	}
	entity := keys[0]
	if _, err := entity.VerifyPrimaryKey(now, config); err != nil {
		return BunChecksumReceipt{}, errors.New("bun publisher certificate is expired, revoked or unbound")
	}
	if _, err := entity.VerifyPrimaryKey(sig.CreationTime, config); err != nil {
		return BunChecksumReceipt{}, errors.New("bun publisher certificate was invalid at signing")
	}
	current, valid := entity.SigningKeyById(now, *sig.IssuerKeyId, config)
	if !valid {
		return BunChecksumReceipt{}, errors.New("bun signing key is not currently valid")
	}
	historical, valid := entity.SigningKeyById(sig.CreationTime, *sig.IssuerKeyId, config)
	if !valid || bunFingerprint(current.PublicKey) != bunFingerprint(historical.PublicKey) {
		return BunChecksumReceipt{}, errors.New("bun signing key was not bound at signing")
	}
	_, signer, err := openpgp.VerifyDetachedSignature(keys, bytes.NewReader(block.Bytes), bytes.NewReader(signature), config)
	if err != nil || signer != entity {
		return BunChecksumReceipt{}, errors.New("bun checksum publisher signature did not verify")
	}
	digest, err := bunChecksumForArchive(block.Plaintext)
	if err != nil || digest != expectedArchiveSHA256 {
		return BunChecksumReceipt{}, errors.New("bun signed checksum differs from the exact frozen archive")
	}
	if err := ctx.Err(); err != nil {
		return BunChecksumReceipt{}, errors.Join(err, context.Cause(ctx))
	}
	return BunChecksumReceipt{digest, uvDigest(signed), uvDigest(block.Plaintext), bunFingerprint(current.PublicKey), sig.CreationTime}, nil
}

// CheckArchive measures complete bytes without extraction or execution. A
// valid signature over a different filename or a zero receipt cannot approve it.
func (r BunChecksumReceipt) CheckArchive(ctx context.Context, body []byte) error {
	if ctx == nil || r.archive == "" || r.document == "" || r.signer == "" || len(body) == 0 || len(body) > BunArchiveLimit {
		return errors.New("bun archive lacks its bounded authenticated checksum")
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(err, context.Cause(ctx))
	}
	if uvDigest(body) != r.archive {
		return errors.New("bun complete archive bytes differ from signed checksum")
	}
	return errors.Join(ctx.Err(), context.Cause(ctx))
}

func bunFingerprint(key *packet.PublicKey) string {
	return strings.ToUpper(hex.EncodeToString(key.Fingerprint))
}

func bunChecksumForArchive(body []byte) (string, error) {
	if len(body) == 0 || len(body) > bunSignedChecksumsLimit {
		return "", errors.New("bun checksum document is invalid")
	}
	seen := map[string]bool{}
	selected := ""
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if len(seen) >= 128 || len(line) < 67 || len(line) > 320 || line[64:66] != "  " || !hexSHA256.MatchString(line[:64]) {
			return "", errors.New("bun checksum record is invalid")
		}
		name := line[66:]
		if seen[name] || name == "." || name == ".." {
			return "", errors.New("bun checksum filename is duplicated or invalid")
		}
		for _, r := range name {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
				return "", errors.New("bun checksum filename is unsafe")
			}
		}
		seen[name] = true
		if name == BunArchiveFilename {
			selected = line[:64]
		}
	}
	if scanner.Err() != nil || selected == "" {
		return "", errors.New("bun exact archive checksum is missing")
	}
	return selected, nil
}
