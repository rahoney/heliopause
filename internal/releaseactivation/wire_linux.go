//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sort"

	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
	"golang.org/x/mod/semver"
)

const updateWireMagic = "HAAUPD1\n"

var ErrProtocol = errors.New("SELF_UPDATE_PAYLOAD_PROTOCOL_INVALID")

// wireRelease contains declarations and proof only. No constructor here can
// create a Policy decision, prepared root stage or activation receipt.
type wireRelease struct {
	body     []byte
	manifest releaseinstall.ReleaseManifest
	proofs   [][]byte
	files    []releaseinstall.ReleaseFile
}

func newWireRelease(body []byte, proofs [][]byte) (wireRelease, error) {
	manifest, err := releaseinstall.ParseReleaseManifest(body)
	if err != nil {
		return wireRelease{}, errors.Join(ErrProtocol, err)
	}
	files, err := releasePayloads(manifest, int64(len(body)))
	if err != nil {
		return wireRelease{}, err
	}
	if len(proofs) == 0 || len(proofs) > 16 {
		return wireRelease{}, ErrProtocol
	}
	owned := make([][]byte, len(proofs))
	for index, proof := range proofs {
		if len(proof) == 0 || len(proof) > 1<<20 {
			return wireRelease{}, ErrProtocol
		}
		owned[index] = append([]byte(nil), proof...)
	}
	return wireRelease{append([]byte(nil), body...), manifest, owned, files}, nil
}

func releasePayloads(manifest releaseinstall.ReleaseManifest, manifestSize int64) ([]releaseinstall.ReleaseFile, error) {
	native, err := manifest.NativeUpdateFiles()
	if err != nil {
		return nil, errors.Join(ErrProtocol, err)
	}
	files := append([]releaseinstall.ReleaseFile(nil), native...)
	for _, file := range manifest.Files() {
		if file.Role == "release-metadata" {
			files = append(files, file)
		}
	}
	if len(files) != 6 || manifestSize <= 0 || manifestSize > 1<<20 {
		return nil, ErrProtocol
	}
	total := manifestSize
	for _, file := range files {
		if file.Size <= 0 || file.Size > 512<<20 || file.Size > (1<<30)-total {
			return nil, ErrProtocol
		}
		total += file.Size
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, nil
}

func readWirePreamble(ctx context.Context, reader io.Reader) (wireRelease, error) {
	if ctx == nil || reader == nil {
		return wireRelease{}, ErrProtocol
	}
	if err := ctx.Err(); err != nil {
		return wireRelease{}, err
	}
	input := protectedContextReader{ctx, reader}
	magic := make([]byte, len(updateWireMagic))
	if _, err := io.ReadFull(input, magic); err != nil {
		return wireRelease{}, errors.Join(ErrProtocol, err, ctx.Err())
	}
	if !bytes.Equal(magic, []byte(updateWireMagic)) {
		return wireRelease{}, ErrProtocol
	}
	body, err := readWireFrame(ctx, input, 1<<20)
	if err != nil {
		return wireRelease{}, err
	}
	count, err := readWireNumber(ctx, input)
	if err != nil {
		return wireRelease{}, err
	}
	if count == 0 || count > 16 {
		return wireRelease{}, ErrProtocol
	}
	proofs := make([][]byte, 0, count)
	for index := uint32(0); index < count; index++ {
		proof, err := readWireFrame(ctx, input, 1<<20)
		if err != nil {
			return wireRelease{}, err
		}
		proofs = append(proofs, proof)
	}
	return newWireRelease(body, proofs)
}

func writeWirePreamble(ctx context.Context, writer io.Writer, release wireRelease) error {
	if ctx == nil || writer == nil {
		return ErrProtocol
	}
	checked, err := newWireRelease(release.body, release.proofs)
	if err != nil {
		return err
	}
	if err := writeWireBytes(ctx, writer, []byte(updateWireMagic)); err != nil {
		return err
	}
	if err := writeWireFrame(ctx, writer, checked.body); err != nil {
		return err
	}
	if err := writeWireNumber(ctx, writer, uint32(len(checked.proofs))); err != nil {
		return err
	}
	for _, proof := range checked.proofs {
		if err := writeWireFrame(ctx, writer, proof); err != nil {
			return err
		}
	}
	return nil
}

func readWireNumber(ctx context.Context, reader io.Reader) (uint32, error) {
	var number [4]byte
	if _, err := io.ReadFull(reader, number[:]); err != nil {
		return 0, errors.Join(ErrProtocol, err, ctx.Err())
	}
	return binary.BigEndian.Uint32(number[:]), nil
}

func readWireFrame(ctx context.Context, reader io.Reader, limit uint32) ([]byte, error) {
	size, err := readWireNumber(ctx, reader)
	if err != nil {
		return nil, err
	}
	if size == 0 || size > limit {
		return nil, ErrProtocol
	}
	body := make([]byte, size)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, errors.Join(ErrProtocol, err, ctx.Err())
	}
	return body, nil
}

func writeWireNumber(ctx context.Context, writer io.Writer, value uint32) error {
	var body [4]byte
	binary.BigEndian.PutUint32(body[:], value)
	return writeWireBytes(ctx, writer, body[:])
}

func writeWireFrame(ctx context.Context, writer io.Writer, body []byte) error {
	if len(body) == 0 || len(body) > 1<<20 {
		return ErrProtocol
	}
	if err := writeWireNumber(ctx, writer, uint32(len(body))); err != nil {
		return err
	}
	return writeWireBytes(ctx, writer, body)
}

func writeWireBytes(ctx context.Context, writer io.Writer, body []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	count, err := writer.Write(body)
	if err != nil {
		return errors.Join(ErrProtocol, err, ctx.Err())
	}
	if count != len(body) {
		return errors.Join(ErrProtocol, io.ErrShortWrite)
	}
	return ctx.Err()
}

func (r wireRelease) verify(ctx context.Context, current InstalledRelease) ([]byte, error) {
	if ctx == nil {
		return nil, ErrProtocol
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if current.record.Schema != registrationSchema || len(current.proof) == 0 {
		return nil, ErrUntrusted
	}
	comparison := semver.Compare(r.manifest.Version(), current.Version())
	if comparison < 0 || (comparison == 0 && r.manifest.SHA256() != current.manifest.SHA256()) || r.manifest.RuntimeLockSHA256() != current.manifest.RuntimeLockSHA256() || r.manifest.GVisorRelease() != current.manifest.GVisorRelease() || r.manifest.GVisorCommit() != current.manifest.GVisorCommit() {
		return nil, ErrUntrusted
	}
	subjects := []releaseprovenance.Subject{{Name: "helox-release-manifest.json", SHA256: r.manifest.SHA256()}}
	for _, file := range r.files {
		subjects = append(subjects, releaseprovenance.Subject{Name: file.Name, SHA256: file.SHA256})
	}
	expected := releaseprovenance.Identity{Version: r.manifest.Version(), SourceCommit: r.manifest.SourceCommit(), WorkflowRun: r.manifest.WorkflowRun()}
	for _, proof := range r.proofs {
		if _, err := releaseprovenance.Verify(ctx, proof, expected, subjects); err == nil {
			return append([]byte(nil), proof...), nil
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	return nil, ErrUntrusted
}
