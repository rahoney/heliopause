package sandbox

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/rahoney/heliopause/internal/core/domain"
	"io"
	"sort"
	"strings"
)

// Diagnostic DATA does not grant source identity, admission or Policy approval.
type uvRunCensusFact struct {
	Name         string
	Kind         byte
	UID, GID     int
	Mode, Size   int64
	Link, SHA256 string
	Body         []byte `json:",omitempty"`
}

func scanUVRunCensus(input io.Reader, command string) ([]uvRunCensusFact, error) {
	if input == nil || command == "" || domain.ValidateExecutionCommand(command) != nil {
		return nil, errors.New("native census requires a bounded selected command and stream")
	}
	bounded := &io.LimitedReader{R: input, N: 4<<20 + 64<<10}
	archive := tar.NewReader(bounded)
	var facts []uvRunCensusFact
	seen := map[string]bool{}
	var expanded, controlBytes int64
	for {
		h, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("native cache census tar: %w", err)
		}
		name := strings.TrimSuffix(strings.TrimPrefix(h.Name, "./"), "/")
		if name != "" && name != "." && !validClosureDestination(name) || seen[name] || h.Size < 0 || h.Size > (4<<20)-expanded || len(facts) >= 4096 {
			return nil, errors.New("native cache census path/duplicate/count/size bound")
		}
		seen[name] = true
		expanded += h.Size
		row := uvRunCensusFact{Name: name, Kind: h.Typeflag, UID: h.Uid, GID: h.Gid, Mode: h.Mode, Size: h.Size, Link: h.Linkname}
		if h.Typeflag == tar.TypeReg || h.Typeflag == 0 {
			digest := sha256.New()
			capture := strings.HasSuffix(name, "/METADATA") || strings.HasSuffix(name, "/WHEEL") || strings.HasSuffix(name, "/pyvenv.cfg") || strings.HasSuffix(name, "/bin/"+command)
			if capture {
				if h.Size > 64<<10 || h.Size > (256<<10)-controlBytes {
					return nil, errors.New("native census control bytes exceed diagnostic bound")
				}
				row.Body, err = io.ReadAll(io.TeeReader(io.LimitReader(archive, h.Size), digest))
				if err != nil || int64(len(row.Body)) != h.Size {
					return nil, errors.New("native census control truncated")
				}
				controlBytes += h.Size
			} else if n, err := io.CopyN(digest, archive, h.Size); err != nil || n != h.Size {
				return nil, errors.New("native census regular file truncated")
			}
			row.SHA256 = hex.EncodeToString(digest.Sum(nil))
		}
		facts = append(facts, row)
	}
	if err := drainUVTarPadding(bounded); err != nil {
		return nil, err
	}
	if bounded.N == 0 || len(facts) == 0 {
		return nil, errors.New("native census absent or transport bound exhausted")
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].Name < facts[j].Name })
	return facts, nil
}
