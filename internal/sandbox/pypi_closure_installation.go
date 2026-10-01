package sandbox

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
)

type closureInstallationFile struct {
	path   string
	size   int64
	sha256 string
	mode   int64
	uid    int
	gid    int
}

type closureInstallation struct {
	files    map[string]closureInstallationFile
	identity string
	bytes    int64
}

type streamingDockerOutput interface {
	RunOutput(context.Context, io.Writer, string, ...string) error
}

// verifyClosureInstallation streams the installed tree out of the trusted
// preparation runtime. No wheel code executes on the Host. Exact RECORD paths
// and content are checked; the three pip-owned dist-info additions are bounded
// and recorded under their canonical distribution only.
func verifyClosureInstallation(ctx context.Context, runner CommandRunner, containerID string, manifest closureManifest, capacity int64) (closureInstallation, error) {
	output, ok := runner.(streamingDockerOutput)
	if !ok || ctx == nil || !exactObservationContainerID(containerID) || manifest.identity == "" || capacity <= 0 {
		return closureInstallation{}, errors.New("python closure installation verifier is unavailable")
	}
	reader, writer := io.Pipe()
	commandResult := make(chan error, 1)
	go func() {
		err := output.RunOutput(ctx, writer, "docker", "cp", containerID+":"+pythonSitePath+"/.", "-")
		_ = writer.CloseWithError(err)
		commandResult <- err
	}()
	installed, verifyErr := scanClosureTar(reader, manifest, capacity)
	_ = reader.CloseWithError(verifyErr)
	commandErr := <-commandResult
	if verifyErr != nil || commandErr != nil || ctx.Err() != nil {
		return closureInstallation{}, errors.New("python closure installed tree does not match authenticated manifest")
	}
	return installed, nil
}

func scanClosureTar(input io.Reader, manifest closureManifest, capacity int64) (closureInstallation, error) {
	if input == nil || capacity <= 0 || len(manifest.files) == 0 {
		return closureInstallation{}, errors.New("python closure tar input is invalid")
	}
	installed := closureInstallation{files: make(map[string]closureInstallationFile, len(manifest.files))}
	generatedLimit := manifest.artifactCount*3 + len(manifest.generatedScripts)
	generated := 0
	entries := 0
	archive := tar.NewReader(input)
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return closureInstallation{}, errors.New("python closure tar stream is invalid")
		}
		entries++
		if entries > len(manifest.files)*8+generatedLimit+64 {
			return closureInstallation{}, errors.New("python closure has too many installed entries")
		}
		name := strings.TrimPrefix(header.Name, "./")
		if name == "." || name == "" {
			if header.Typeflag != tar.TypeDir {
				return closureInstallation{}, errors.New("python closure root is invalid")
			}
			continue
		}
		name = strings.TrimSuffix(name, "/")
		if !validClosureDestination(name) || header.Mode&^int64(0o777) != 0 || header.Mode&0o022 != 0 ||
			header.Uid != 1000 || header.Gid != 1000 {
			return closureInstallation{}, errors.New("python closure installed path, mode or ownership is invalid")
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		// POSIX tar also permits the legacy NUL regular-file type.
		if header.Typeflag != tar.TypeReg && header.Typeflag != 0 {
			return closureInstallation{}, errors.New("python closure contains unsupported installed file type")
		}
		if _, duplicate := installed.files[name]; duplicate || header.Size < 0 || header.Size > capacity-installed.bytes {
			return closureInstallation{}, errors.New("python closure installed file repeats or exceeds storage bound")
		}
		expected, known := manifest.files[name]
		if known {
			if expected.sha256 != "" && header.Size != expected.size {
				return closureInstallation{}, errors.New("python closure installed size differs from RECORD")
			}
		} else {
			if !allowedInstallerGeneratedPath(name, manifest) {
				return closureInstallation{}, errors.New("python closure has unowned installer output")
			}
			generated++
			if generated > generatedLimit || header.Size > 2<<20 {
				return closureInstallation{}, errors.New("python closure installer output exceeds bound")
			}
		}
		hash := sha256.New()
		if _, err := io.CopyN(hash, archive, header.Size); err != nil {
			return closureInstallation{}, errors.New("python closure installed file is truncated")
		}
		observed := hex.EncodeToString(hash.Sum(nil))
		if known && expected.sha256 != "" && observed != expected.sha256 {
			return closureInstallation{}, errors.New("python closure installed hash differs from RECORD")
		}
		installed.bytes += header.Size
		installed.files[name] = closureInstallationFile{
			path: name, size: header.Size, sha256: observed,
			mode: header.Mode, uid: header.Uid, gid: header.Gid,
		}
	}
	for name := range manifest.files {
		if _, present := installed.files[name]; !present {
			return closureInstallation{}, fmt.Errorf("python closure installed file missing: %s", name)
		}
	}
	for name := range manifest.generatedScripts {
		if _, present := installed.files[name]; !present {
			return closureInstallation{}, fmt.Errorf("python closure generated wrapper missing: %s", name)
		}
	}
	// Include every installer-generated byte and mode in the immutable closure
	// identity. The Docker volume and anchor attest that this tree remains fixed.
	names := make([]string, 0, len(installed.files))
	for name := range installed.files {
		names = append(names, name)
	}
	sort.Strings(names)
	identity := sha256.New()
	for _, name := range names {
		file := installed.files[name]
		_, _ = fmt.Fprintf(identity, "%s\x00%d\x00%s\x00%o\x00%d:%d\n", name, file.size, file.sha256, file.mode, file.uid, file.gid)
	}
	installed.identity = hex.EncodeToString(identity.Sum(nil))
	return installed, nil
}

func allowedInstallerGeneratedPath(name string, manifest closureManifest) bool {
	if _, expected := manifest.generatedScripts[name]; expected {
		return true
	}
	base := path.Base(name)
	if base != "INSTALLER" && base != "REQUESTED" && base != "direct_url.json" {
		return false
	}
	directory := path.Dir(name)
	if !strings.HasSuffix(directory, ".dist-info") {
		return false
	}
	_, canonical := manifest.files[directory+"/RECORD"]
	return canonical
}
