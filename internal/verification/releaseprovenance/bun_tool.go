package releaseprovenance

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"math/bits"
	"os"
)

const maxBunNativeBytes = 128 << 20

// BunToolReceipt measures the one native file inside an authenticated archive.
// It grants no native version, CPU/guest compatibility or execution authority.
type BunToolReceipt struct {
	archive   BunChecksumReceipt
	digest    string
	size      int64
	libraries []string
}

func (r BunToolReceipt) ArchiveSHA256() string { return r.archive.ArchiveSHA256() }
func (r BunToolReceipt) SHA256() string        { return r.digest }
func (r BunToolReceipt) Size() int64           { return r.size }
func (r BunToolReceipt) Libraries() []string   { return append([]string(nil), r.libraries...) }

// ReadToolArchive authenticates complete bytes before exposing any native file.
// Every member/CRC is checked; nothing is written or executed on the Host.
func (r BunChecksumReceipt) ReadToolArchive(ctx context.Context, body []byte) (BunToolReceipt, []byte, error) {
	if err := r.CheckArchive(ctx, body); err != nil {
		return BunToolReceipt{}, nil, err
	}
	native, libraries, err := readBunNativeZIP(ctx, body)
	if err != nil {
		return BunToolReceipt{}, nil, err
	}
	return BunToolReceipt{r, uvDigest(native), int64(len(native)), libraries}, native, nil
}

// This structural reader alone confers no authenticity. Only the receipt-bound
// public caller above may expose it as measured signed native data.
func readBunNativeZIP(ctx context.Context, body []byte) ([]byte, []string, error) {
	if ctx == nil || len(body) == 0 || len(body) > BunArchiveLimit || !bytes.HasPrefix(body, []byte{'P', 'K', 3, 4}) {
		return nil, nil, errors.New("bun ZIP input is unavailable or excessive")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, errors.Join(err, context.Cause(ctx))
	}
	// This fixed small archive has one disk, no ZIP64 and no comment. The
	// standard reader permits trailing transport bytes, so close its boundary
	// explicitly before interpreting members.
	if len(body) < 22 {
		return nil, nil, errors.New("bun ZIP directory boundary is unavailable")
	}
	end := body[len(body)-22:]
	if !bytes.Equal(end[:4], []byte{'P', 'K', 5, 6}) ||
		binary.LittleEndian.Uint16(end[4:]) != 0 || binary.LittleEndian.Uint16(end[6:]) != 0 ||
		binary.LittleEndian.Uint16(end[8:]) != 2 || binary.LittleEndian.Uint16(end[10:]) != 2 ||
		binary.LittleEndian.Uint16(end[20:]) != 0 ||
		uint64(binary.LittleEndian.Uint32(end[12:]))+uint64(binary.LittleEndian.Uint32(end[16:])) != uint64(len(body)-22) {
		return nil, nil, errors.New("bun ZIP directory boundary or coverage is unsupported")
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil || len(archive.File) != 2 || archive.Comment != "" {
		return nil, nil, errors.New("bun ZIP member coverage or format is unsupported")
	}
	seen := map[string]bool{}
	var native []byte
	for _, file := range archive.File {
		if seen[file.Name] || file.CreatorVersion>>8 != 3 || file.Flags&^uint16(0x0808) != 0 || !bunZIPExtra(file.Extra) || file.Comment != "" || file.CompressedSize64 > uint64(len(body)) {
			return nil, nil, errors.New("bun ZIP metadata or member identity is invalid")
		}
		seen[file.Name] = true
		switch file.Name {
		case "bun-linux-x64/":
			if file.Mode() != os.ModeDir|0o755 || file.UncompressedSize64 != 0 || file.CompressedSize64 != 0 || file.Method != zip.Store || file.CRC32 != 0 {
				return nil, nil, errors.New("bun ZIP directory is unsupported")
			}
		case "bun-linux-x64/bun":
			if file.Mode() != 0o755 || file.UncompressedSize64 == 0 || file.UncompressedSize64 > maxBunNativeBytes || file.Method != zip.Deflate && file.Method != zip.Store {
				return nil, nil, errors.New("bun native member type, mode or declared expansion is invalid")
			}
		default:
			return nil, nil, errors.New("bun ZIP contains an unexpected member")
		}
		stream, err := file.Open()
		if err != nil {
			return nil, nil, err
		}
		content, readErr := io.ReadAll(io.LimitReader(bunZIPContextReader{ctx, stream}, maxBunNativeBytes+1))
		closeErr := stream.Close()
		if readErr != nil || closeErr != nil || len(content) > maxBunNativeBytes || uint64(len(content)) != file.UncompressedSize64 {
			return nil, nil, errors.Join(errors.New("bun ZIP expansion, CRC or complete read is invalid"), readErr, closeErr)
		}
		if file.Name == "bun-linux-x64/bun" {
			native = content
		}
	}
	if !seen["bun-linux-x64/"] || !seen["bun-linux-x64/bun"] || len(native) == 0 {
		return nil, nil, errors.New("bun native ZIP coverage is incomplete")
	}
	libraries, err := gnuAMD64NativeELF(native)
	if err != nil {
		return nil, nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, errors.Join(err, context.Cause(ctx))
	}
	return native, libraries, nil
}

type bunZIPContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r bunZIPContextReader) Read(body []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, errors.Join(err, context.Cause(r.ctx))
	}
	return r.reader.Read(body)
}

// The pinned writer uses standard extended times and Unix UID/GID data only.
// These are bounded format metadata; they never choose a Host owner or mode.
func bunZIPExtra(extra []byte) bool {
	seen := map[uint16]bool{}
	for len(extra) > 0 {
		if len(extra) < 4 {
			return false
		}
		kind, length := binary.LittleEndian.Uint16(extra), int(binary.LittleEndian.Uint16(extra[2:]))
		extra = extra[4:]
		if length > len(extra) || seen[kind] {
			return false
		}
		seen[kind] = true
		value := extra[:length]
		extra = extra[length:]
		switch kind {
		case 0x5455:
			if len(value) < 1 || value[0]&^byte(7) != 0 || len(value) != 1+4*bits.OnesCount8(value[0]) {
				return false
			}
		case 0x7875:
			if len(value) < 5 || value[0] != 1 || value[1] < 1 || value[1] > 4 {
				return false
			}
			gidLengthIndex := 2 + int(value[1])
			if gidLengthIndex >= len(value) || value[gidLengthIndex] < 1 || value[gidLengthIndex] > 4 || len(value) != gidLengthIndex+1+int(value[gidLengthIndex]) {
				return false
			}
		default:
			return false
		}
	}
	return true
}
