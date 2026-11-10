package releaseprovenance

import (
	"bytes"
	"context"
	"crypto"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	openpgp "github.com/ProtonMail/go-crypto/openpgp/v2"
)

const bunActualArchiveDigest = "bf8ccbfb2b0b80159e0f9a6d15c55317add8077534dbc6130a3edb52fccb395c"

func bunActualSignedChecksums(t *testing.T) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "bun-1.4.4", "SHASUMS256.txt.asc"))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func bunVerificationTime() time.Time { return time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC) }

func TestBunOfficialSignedChecksumsHaveOnlyArchiveAuthority(t *testing.T) {
	body := bunActualSignedChecksums(t)
	r, err := VerifyBunChecksums(context.Background(), body, bunActualArchiveDigest, bunVerificationTime())
	if err != nil {
		t.Fatal(err)
	}
	if r.ArchiveSHA256() != bunActualArchiveDigest || r.SigningFingerprint() != BunPublisherFingerprint || r.SignedDocumentSHA256() != uvDigest(body) || r.ChecksumsSHA256() == "" || r.SignatureTime().IsZero() {
		t.Fatal("incomplete vendor checksum receipt")
	}
	if err := r.CheckArchive(context.Background(), []byte("different zip")); err == nil {
		t.Fatal("signed checksum approved different archive")
	}
	if err := (BunChecksumReceipt{}).CheckArchive(context.Background(), []byte("different zip")); err == nil {
		t.Fatal("zero receipt approved archive")
	}
	cause := errors.New("cancel archive measurement")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if err := r.CheckArchive(ctx, []byte("different zip")); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal("archive cancellation cause lost", err)
	}
}

func TestBunChecksumsRejectWrongPublisherEnvelopeAndSubject(t *testing.T) {
	for _, change := range []string{"payload", "hash-header", "prefix", "suffix", "second-block", "missing", "oversize", "subject", "future", "zero-time", "nil-context", "cancel", "unknown-signer"} {
		t.Run(change, func(t *testing.T) {
			body := bunActualSignedChecksums(t)
			expected := bunActualArchiveDigest
			now := bunVerificationTime()
			ctx := context.Background()
			cause := errors.New("cancel bun checksum verification")
			switch change {
			case "payload":
				body = bytes.Replace(body, []byte("c724ac51d1ffff2d5"), []byte("d724ac51d1ffff2d5"), 1)
			case "hash-header":
				body = bytes.Replace(body, []byte("Hash: SHA512"), []byte("Hash: SHA1"), 1)
			case "prefix":
				body = append([]byte("untrusted prefix\n"), body...)
			case "suffix":
				body = append(body, []byte("untrusted suffix")...)
			case "second-block":
				body = append(body, body...)
			case "missing":
				body = nil
			case "oversize":
				body = bytes.Repeat([]byte("x"), bunSignedChecksumsLimit+1)
			case "subject":
				expected = strings.Repeat("0", 64)
			case "future":
				now = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			case "zero-time":
				now = time.Time{}
			case "nil-context":
				ctx = nil
			case "cancel":
				var cancel context.CancelCauseFunc
				ctx, cancel = context.WithCancelCause(ctx)
				cancel(cause)
			case "unknown-signer":
				config := &packet.Config{Time: bunVerificationTime, DefaultHash: crypto.SHA512, RSABits: 2048}
				entity, err := openpgp.NewEntity("Untrusted fixture", "", "fixture@example.invalid", config)
				if err != nil {
					t.Fatal(err)
				}
				var output bytes.Buffer
				writer, err := clearsign.Encode(&output, entity.PrivateKey, config)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := writer.Write([]byte(expected + "  " + BunArchiveFilename + "\n")); err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				body = output.Bytes()
			}
			if receipt, err := VerifyBunChecksums(ctx, body, expected, now); err == nil || receipt.ArchiveSHA256() != "" {
				t.Fatal("invalid bun publisher proof admitted", change)
			} else if change == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal("verification cancellation cause lost", err)
			} else if change == "payload" && !strings.Contains(err.Error(), "signature did not verify") {
				t.Fatal("changed non-target checksum did not reach cryptographic rejection", err)
			}
		})
	}
}

func TestBunChecksumDocumentRejectsAmbiguousAndUnsafeRecords(t *testing.T) {
	good := bunActualArchiveDigest + "  " + BunArchiveFilename
	for name, body := range map[string]string{"duplicate": good + "\n" + good, "unsafe": good + "\n" + bunActualArchiveDigest + "  ../escape.zip", "bad-hash": "not-a-digest  " + BunArchiveFilename, "missing": bunActualArchiveDigest + "  unrelated.zip", "empty-line": good + "\n\n", "trailing-space": good + " ", "oversized-record": good + strings.Repeat("x", 400)} {
		t.Run(name, func(t *testing.T) {
			if _, err := bunChecksumForArchive([]byte(body)); err == nil {
				t.Fatal("ambiguous checksum document accepted")
			}
		})
	}
	if digest, err := bunChecksumForArchive([]byte(good + "\n")); err != nil || digest != bunActualArchiveDigest {
		t.Fatal("ordinary checksum record rejected", err)
	}
}
