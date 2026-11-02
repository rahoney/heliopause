package sandbox

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"sort"
	"strings"
)

// Paths and link targets exist only in this transient controller inventory.
// Public results receive counts and the inventory digest, never this map.
type environmentFileState struct {
	kind     byte
	mode     int64
	uid, gid int
	size     int64
	digest   string
}

type environmentState struct {
	files  map[string]environmentFileState
	digest string
	bytes  int64
}

type environmentStateLimits struct {
	entries      int
	bytes        int64
	archiveBytes int64
}

var errEnvironmentState = errors.New("environment state is unavailable or incomplete")

// readContainerTree reuses the existing Docker DATA stream. The caller must
// first attest the owned mount and its attachments; this transport grants no
// execution or filesystem authority. No artifact-selected root is accepted.
func readContainerTree(ctx context.Context, runner CommandRunner, container, root string, scan func(io.Reader) error) error {
	output, ok := runner.(streamingDockerOutput)
	if !ok || ctx == nil || !exactObservationContainerID(container) ||
		(root != pythonSitePath && root != "/tmp") || scan == nil {
		return errors.New("container DATA tree transport is unavailable")
	}
	reader, writer := io.Pipe()
	result := make(chan error, 1)
	go func() {
		err := output.RunOutput(ctx, writer, "docker", "cp", container+":"+root+"/.", "-")
		_ = writer.CloseWithError(err)
		result <- err
	}()
	scanErr := scan(reader)
	_ = reader.CloseWithError(scanErr)
	commandErr := <-result
	return errors.Join(scanErr, commandErr, ctx.Err())
}

func captureEnvironmentState(ctx context.Context, runner CommandRunner, container string, limits environmentStateLimits) (environmentState, error) {
	var state environmentState
	err := readContainerTree(ctx, runner, container, "/tmp", func(input io.Reader) error {
		var err error
		state, err = scanEnvironmentState(ctx, input, limits)
		return err
	})
	if err != nil {
		return environmentState{}, errors.Join(errEnvironmentState, err)
	}
	return state, nil
}

func scanEnvironmentState(ctx context.Context, input io.Reader, limits environmentStateLimits) (environmentState, error) {
	if ctx == nil || input == nil || ctx.Err() != nil || limits.entries <= 0 ||
		limits.bytes <= 0 || limits.archiveBytes <= 0 || limits.archiveBytes == math.MaxInt64 {
		return environmentState{}, errEnvironmentState
	}
	// Bound the complete transport too: empty files, padding and extended TAR
	// headers must not evade the logical-content budget.
	bounded := &io.LimitedReader{R: input, N: limits.archiveBytes + 1}
	archive := tar.NewReader(bounded)
	state := environmentState{files: make(map[string]environmentFileState)}
	entries := 0
	for {
		if ctx.Err() != nil {
			return environmentState{}, errors.Join(errEnvironmentState, ctx.Err())
		}
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return environmentState{}, errEnvironmentState
		}
		entries++
		if entries > limits.entries || header.Mode < 0 || header.Mode&^int64(07777) != 0 || header.Uid < 0 || header.Gid < 0 {
			return environmentState{}, errEnvironmentState
		}
		name := strings.TrimSuffix(strings.TrimPrefix(header.Name, "./"), "/")
		if name == "." || name == "" {
			if header.Typeflag != tar.TypeDir {
				return environmentState{}, errEnvironmentState
			}
			// The root itself participates in mode/ownership changes.
			name = "."
		} else if len(name) > 4096 || !validClosureDestination(name) {
			return environmentState{}, errEnvironmentState
		}
		if _, duplicate := state.files[name]; duplicate {
			return environmentState{}, errEnvironmentState
		}
		file := environmentFileState{kind: header.Typeflag, mode: header.Mode, uid: header.Uid, gid: header.Gid, size: header.Size}
		switch header.Typeflag {
		case tar.TypeReg, 0: // Legacy NUL regular-file type, as in the closure scanner.
			file.kind = tar.TypeReg
			if header.Size < 0 || header.Size > limits.bytes-state.bytes {
				return environmentState{}, errEnvironmentState
			}
			hash := sha256.New()
			if _, err := io.CopyN(hash, archive, header.Size); err != nil {
				return environmentState{}, errEnvironmentState
			}
			file.digest = hex.EncodeToString(hash.Sum(nil))
			state.bytes += header.Size
		case tar.TypeDir:
			if header.Size != 0 {
				return environmentState{}, errEnvironmentState
			}
		case tar.TypeLink:
			// Docker emits already-seen inode aliases. Inventory their regular
			// content without extraction, a second read or order-dependent identity.
			// Forward, cyclic, external and nonregular references stay incomplete.
			target := strings.TrimPrefix(header.Linkname, "./")
			original, present := state.files[target]
			if header.Size != 0 || len(target) > 4096 || !validClosureDestination(target) ||
				!present || original.kind != tar.TypeReg || original.mode != file.mode || original.uid != file.uid || original.gid != file.gid {
				return environmentState{}, errEnvironmentState
			}
			file = original
		case tar.TypeSymlink:
			// Inventory the link itself; never follow it, including links into
			// proc/dev or outside the attested writable root.
			if header.Size != 0 || len(header.Linkname) > 4096 || header.Linkname == "" ||
				path.IsAbs(header.Linkname) || strings.ContainsAny(header.Linkname, "\\\x00\n\r") {
				return environmentState{}, errEnvironmentState
			}
			target := path.Clean(path.Join(path.Dir(name), header.Linkname))
			if target != "." && !validClosureDestination(target) {
				return environmentState{}, errEnvironmentState
			}
			digest := sha256.Sum256([]byte(header.Linkname))
			file.digest = hex.EncodeToString(digest[:])
		default:
			// Devices, FIFO and unknown types are incomplete,
			// rather than silently missing content from the snapshot.
			return environmentState{}, errEnvironmentState
		}
		state.files[name] = file
	}
	// A path cannot simultaneously be a file/link and an ancestor directory,
	// regardless of archive order. This also rejects a link followed by data.
	for name := range state.files {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if file, exists := state.files[parent]; exists && file.kind != tar.TypeDir {
				return environmentState{}, errEnvironmentState
			}
		}
	}
	if root, present := state.files["."]; !present || root.kind != tar.TypeDir {
		return environmentState{}, errEnvironmentState
	}
	// Drain TAR termination/padding so a successful early end does not conceal
	// a failed producer or an oversized transport.
	if _, err := io.Copy(io.Discard, bounded); err != nil || bounded.N <= 0 || ctx.Err() != nil {
		return environmentState{}, errEnvironmentState
	}
	names := make([]string, 0, len(state.files))
	for name := range state.files {
		names = append(names, name)
	}
	sort.Strings(names)
	hash := sha256.New()
	for _, name := range names {
		file := state.files[name]
		_, _ = fmt.Fprintf(hash, "%d:%s\x00%d:%o:%d:%d:%d:%s\n", len(name), name, file.kind, file.mode, file.uid, file.gid, file.size, file.digest)
	}
	state.digest = hex.EncodeToString(hash.Sum(nil))
	return state, nil
}

type environmentChanges struct {
	created, deleted, modified uint64
}

// Comparison is linear in the bounded inventory. Neither file creation nor
// ordinary config reads alone imply a malicious finding.
func compareEnvironmentStates(before, after environmentState) (environmentChanges, error) {
	if before.digest == "" || after.digest == "" || before.files == nil || after.files == nil {
		return environmentChanges{}, errEnvironmentState
	}
	var changes environmentChanges
	for name, file := range before.files {
		current, present := after.files[name]
		if !present {
			changes.deleted++
		} else if current != file {
			changes.modified++
		}
	}
	for name := range after.files {
		if _, present := before.files[name]; !present {
			changes.created++
		}
	}
	return changes, nil
}
