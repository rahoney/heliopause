package projectbuild

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
)

import "github.com/rahoney/heliopause/internal/core/domain"

const (
	MaxBuildOutputFiles              = 128
	MaxBuildOutputBytes        int64 = 200 << 20
	MaxBuildOutputFileBytes    int64 = 64 << 20
	MaxBuildOutputArchiveBytes       = MaxBuildOutputBytes + (MaxBuildOutputFiles*2+2)*512
)

var errBuildOutput = errors.New("project build output is invalid, incomplete or changed")

// BuildOutputFile is data inventory, never execution or approval authority.
type BuildOutputFile struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// CapturedBuildOutput retains exact directory ownership until the backend has
// reconciled observation and cleanup. Failure removes only that owned inode.
type CapturedBuildOutput struct {
	parent, run *os.Root
	name        string
	info        os.FileInfo
	artifact    domain.AcquiredArtifact
}

func (o *CapturedBuildOutput) Artifact() domain.AcquiredArtifact { return o.artifact }

func (o *CapturedBuildOutput) Close(keep bool) error {
	if o == nil {
		return nil
	}
	var failure error
	if o.run != nil {
		failure = o.run.Close()
		o.run = nil
	}
	if o.parent != nil {
		if !keep || failure != nil {
			current, err := o.parent.Lstat(o.name)
			if err != nil || o.info == nil || !current.IsDir() || !os.SameFile(current, o.info) {
				failure = errors.Join(failure, errBuildOutput)
			} else {
				failure = errors.Join(failure, o.parent.RemoveAll(o.name))
			}
		}
		failure = errors.Join(failure, o.parent.Close())
		o.parent = nil
	}
	if failure != nil {
		return errBuildOutput
	}
	return nil
}

// CaptureBuildOutput reconstructs bounded flat compiler output as private data.
// Untrusted TAR names/metadata confer no permissions or publication authority.
func CaptureBuildOutput(ctx context.Context, intake string, inputs domain.ProjectBuildInputs, stream io.Reader) (_ *CapturedBuildOutput, resultErr error) {
	if ctx == nil || !inputs.Valid() || stream == nil {
		return nil, errBuildOutput
	}
	parent, err := openBuildOutputParent(intake)
	if err != nil {
		return nil, err
	}
	o := &CapturedBuildOutput{parent: parent, name: inputs.RunID().String()}
	created := false
	defer func() {
		if resultErr != nil {
			if !created {
				resultErr = errors.Join(resultErr, parent.Close())
			} else {
				resultErr = errors.Join(resultErr, o.Close(false))
			}
			resultErr = errors.Join(errBuildOutput, ctx.Err())
		}
	}()
	if err := parent.Mkdir(o.name, 0o700); err != nil {
		return nil, errBuildOutput
	}
	created = true
	o.info, err = parent.Lstat(o.name)
	if err != nil || !o.info.IsDir() || o.info.Mode().Perm() != 0o700 {
		return nil, errBuildOutput
	}
	o.run, err = parent.OpenRoot(o.name)
	if err != nil {
		return nil, errBuildOutput
	}
	opened, err := o.run.Stat(".")
	if err != nil || !os.SameFile(o.info, opened) {
		return nil, errBuildOutput
	}
	file, err := o.run.OpenFile(inputs.Kind()+"-output.tar", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return nil, errBuildOutput
	}
	hash := sha256.New()
	bounded := &buildOutputBoundedWriter{writer: io.MultiWriter(file, hash), remaining: MaxBuildOutputArchiveBytes}
	writer := tar.NewWriter(bounded)
	_, scanErr := scanBuildOutput(ctx, stream, false, func(name string, size int64, content io.Reader) error {
		if err := writer.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o400, Size: size, Format: tar.FormatUSTAR}); err != nil {
			return err
		}
		n, err := io.Copy(writer, content)
		if err != nil || n != size {
			return errBuildOutput
		}
		return nil
	})
	tarErr := writer.Close()
	syncErr := file.Sync()
	closeErr := file.Close()
	if errors.Join(scanErr, tarErr, syncErr, closeErr) != nil {
		return nil, errBuildOutput
	}
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(hash.Sum(nil)))
	if err != nil {
		return nil, err
	}
	identity, err := domain.NewResolvedArtifactIdentity(inputs.Source().Identity().Source(), inputs.Source().Identity().Name(), inputs.RunID().String(), inputs.Kind()+"-output")
	if err != nil {
		return nil, err
	}
	o.artifact, err = domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+o.name+":"+inputs.Kind()+"-output", uint64(MaxBuildOutputArchiveBytes-bounded.remaining), "sha256:"+digest.String())
	if err != nil {
		return nil, err
	}
	return o, nil
}

type buildOutputBoundedWriter struct {
	writer    io.Writer
	remaining int64
}

func (w *buildOutputBoundedWriter) Write(body []byte) (int, error) {
	if int64(len(body)) > w.remaining {
		return 0, errBuildOutput
	}
	n, err := w.writer.Write(body)
	w.remaining -= int64(n)
	return n, err
}

func openBuildOutputParent(intake string) (*os.Root, error) {
	if !filepath.IsAbs(intake) || filepath.Clean(intake) != intake || intake == "/" {
		return nil, errBuildOutput
	}
	before, err := os.Lstat(intake)
	if err != nil || !before.IsDir() || before.Mode() != os.ModeDir|0o700 {
		return nil, errBuildOutput
	}
	root, err := os.OpenRoot(intake)
	if err != nil {
		return nil, errBuildOutput
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		_ = root.Close()
		return nil, errBuildOutput
	}
	return root, nil
}

func buildOutputSingleLink(info os.FileInfo) bool {
	v := reflect.ValueOf(info.Sys())
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return false
	}
	n := v.Elem().FieldByName("Nlink")
	return n.IsValid() && n.CanUint() && n.Uint() == 1
}

// ReadBuildOutput verifies all consumed archive bytes using anchored handles.
// The consumer may stage data, but must discard it if this method fails late.
func ReadBuildOutput(ctx context.Context, intake string, artifact domain.AcquiredArtifact, variant string, consume func(string, int64, io.Reader) error) (files []BuildOutputFile, resultErr error) {
	parts := strings.Split(artifact.ContentHandle(), ":")
	declared, ok := artifact.DeclaredIntegrity()
	if (variant != "go-output" && variant != "cargo-output") || ctx == nil || artifact.Identity().Source().String() != "project-local" || artifact.Identity().Variant() != variant || !ok || declared != "sha256:"+artifact.Digest().String() || artifact.SizeBytes() == 0 || artifact.SizeBytes() > uint64(MaxBuildOutputArchiveBytes) || len(parts) != 3 || parts[0] != "intake" || parts[2] != variant || parts[1] != artifact.Identity().Version() {
		return nil, errBuildOutput
	}
	if _, err := domain.ParseRunID(parts[1]); err != nil {
		return nil, errBuildOutput
	}
	parent, err := openBuildOutputParent(intake)
	if err != nil {
		return nil, err
	}
	defer func() {
		if parent.Close() != nil {
			resultErr = errBuildOutput
		}
		if resultErr != nil {
			files = nil
		}
	}()
	beforeRun, err := parent.Lstat(parts[1])
	if err != nil || !beforeRun.IsDir() || beforeRun.Mode() != os.ModeDir|0o700 {
		return nil, errBuildOutput
	}
	run, err := parent.OpenRoot(parts[1])
	if err != nil {
		return nil, errBuildOutput
	}
	defer func() {
		if run.Close() != nil {
			resultErr = errBuildOutput
		}
	}()
	openedRun, err := run.Stat(".")
	if err != nil || !os.SameFile(beforeRun, openedRun) {
		return nil, errBuildOutput
	}
	before, err := run.Lstat(variant + ".tar")
	if err != nil || before.Mode() != 0o400 || !buildOutputSingleLink(before) || uint64(before.Size()) != artifact.SizeBytes() {
		return nil, errBuildOutput
	}
	file, err := run.OpenFile(variant+".tar", os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errBuildOutput
	}
	defer func() {
		if file.Close() != nil {
			resultErr = errBuildOutput
		}
	}()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) || !buildOutputSingleLink(opened) || opened.Mode() != before.Mode() || opened.Size() != before.Size() {
		return nil, errBuildOutput
	}
	hash := sha256.New()
	stream := &io.LimitedReader{R: io.TeeReader(file, hash), N: int64(artifact.SizeBytes()) + 1}
	files, err = scanBuildOutput(ctx, stream, true, consume)
	after, statErr := run.Lstat(variant + ".tar")
	currentRun, runErr := parent.Lstat(parts[1])
	if err != nil || stream.N != 1 || hex.EncodeToString(hash.Sum(nil)) != artifact.Digest().String() || statErr != nil || !os.SameFile(before, after) || after.Mode() != before.Mode() || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) || !buildOutputSingleLink(after) || runErr != nil || !os.SameFile(beforeRun, currentRun) {
		return nil, errBuildOutput
	}
	return files, nil
}

func scanBuildOutput(ctx context.Context, input io.Reader, canonical bool, consume func(string, int64, io.Reader) error) ([]BuildOutputFile, error) {
	limited := &io.LimitedReader{R: input, N: MaxBuildOutputArchiveBytes + 1}
	archive := tar.NewReader(limited)
	files := []BuildOutputFile{}
	seen := map[string]bool{}
	var total int64
	rootSeen := false
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errBuildOutput
		}
		name := strings.TrimPrefix(header.Name, "./")
		if !canonical && (name == "." || name == "") && header.Typeflag == tar.TypeDir && header.Size == 0 && !rootSeen && len(files) == 0 && header.Linkname == "" {
			rootSeen = true
			continue
		}
		if len(files) >= MaxBuildOutputFiles || !validBuildOutputName(name) || seen[strings.ToLower(name)] || header.Linkname != "" || (header.Typeflag != tar.TypeReg && header.Typeflag != 0) || header.Size < 0 || header.Size > MaxBuildOutputFileBytes || header.Size > MaxBuildOutputBytes-total || len(header.PAXRecords) != 0 {
			return nil, errBuildOutput
		}
		if canonical && (header.Name != name || header.Mode != 0o400 || header.Uid != 0 || header.Gid != 0 || header.Uname != "" || header.Gname != "") {
			return nil, errBuildOutput
		}
		if !canonical && (header.Mode&^int64(0o777) != 0 || header.Mode&0o022 != 0 || header.Uid != 1000 || header.Gid != 1000) {
			return nil, errBuildOutput
		}
		seen[strings.ToLower(name)] = true
		hash := sha256.New()
		content := &io.LimitedReader{R: io.TeeReader(archive, hash), N: header.Size}
		if consume != nil {
			if err := consume(name, header.Size, content); err != nil {
				return nil, errBuildOutput
			}
		}
		if _, err := io.Copy(io.Discard, content); err != nil || content.N != 0 {
			return nil, errBuildOutput
		}
		files = append(files, BuildOutputFile{name, header.Size, hex.EncodeToString(hash.Sum(nil))})
		total += header.Size
	}
	// TAR end/padding is still consumed and authenticated. Nonzero trailing data
	// or excessive padding is rejected, rather than hidden behind the first EOF.
	tail, err := io.ReadAll(io.LimitReader(limited, 4097))
	if err != nil || len(tail) > 4096 || limited.N <= 0 {
		return nil, errBuildOutput
	}
	for _, b := range tail {
		if b != 0 {
			return nil, errBuildOutput
		}
	}
	return files, nil
}

func validBuildOutputName(name string) bool {
	if name == "" || len(name) > 100 || strings.HasPrefix(name, ".") || strings.ContainsAny(name, "/\\:\x00\r\n\t ") {
		return false
	}
	for _, r := range name {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}
