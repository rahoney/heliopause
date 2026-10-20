//go:build linux

package releasebootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/rahoney/heliopause/internal/core/ports"
)

type ownedOutputFile struct {
	info   os.FileInfo
	closed bool
	digest string
}
type ownedOutput struct {
	path             string
	root             *os.Root
	identity, intake os.FileInfo
	files            map[string]ownedOutputFile
	closed           bool
	closeErr         error
}

func newOwnedOutput() (*ownedOutput, error) {
	name, err := os.MkdirTemp("", "helox-bootstrap-")
	if err != nil {
		return nil, err
	}
	canonical, err := filepath.EvalSymlinks(name)
	if err != nil {
		_ = os.Remove(name)
		return nil, err
	}
	info, err := os.Lstat(canonical)
	if err != nil || !privateDirectory(info) {
		_ = os.Remove(name)
		return nil, errors.New("bootstrap output is not private")
	}
	root, err := os.OpenRoot(canonical)
	if err != nil {
		_ = os.Remove(name)
		return nil, err
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		_ = root.Close()
		return nil, errors.New("bootstrap output changed")
	}
	output := &ownedOutput{path: canonical, root: root, identity: info, files: map[string]ownedOutputFile{}}
	if err := root.Mkdir("intake", 0700); err != nil {
		_ = output.close()
		_ = output.removeDirectory()
		return nil, err
	}
	output.intake, err = root.Lstat("intake")
	if err != nil || !privateDirectory(output.intake) {
		_ = root.Remove("intake")
		_ = output.close()
		_ = output.removeDirectory()
		return nil, errors.New("bootstrap intake is not private")
	}
	return output, nil
}

func privateDirectory(info os.FileInfo) bool {
	if info == nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0700 {
		return false
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	return ok && owner.Uid == uint32(os.Geteuid())
}
func ownedRegular(info os.FileInfo) bool {
	if info == nil || !info.Mode().IsRegular() || info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 || info.Mode().Perm()&^0600 != 0 {
		return false
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	return ok && owner.Uid == uint32(os.Geteuid()) && owner.Nlink == 1
}
func (o *ownedOutput) check() error {
	info, err := os.Lstat(o.path)
	if err != nil || !privateDirectory(info) || !os.SameFile(o.identity, info) {
		return errors.New("bootstrap output identity changed")
	}
	return nil
}

type exportWriter struct {
	file           *os.File
	ctx            context.Context
	current, total int64
}

func (w *exportWriter) Write(body []byte) (int, error) {
	n, err := w.file.Write(body)
	w.current += int64(n)
	if n > 0 {
		ports.ReportProgress(w.ctx, ports.Progress{Stage: "stage", Current: w.current, Total: w.total, Unit: "bytes"})
	}
	return n, err
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(body []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(body)
}

func (o *ownedOutput) copy(ctx context.Context, name string, reader io.Reader, size uint64, digest string) (resultErr error) {
	if filepath.Base(name) != name || name == "." || name == ".." || size == 0 || size > 512<<20 {
		return errors.New("invalid bootstrap output member")
	}
	if err := o.check(); err != nil {
		return err
	}
	file, err := o.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() {
		info, statErr := file.Stat()
		closeErr := file.Close()
		o.files[name] = ownedOutputFile{info: info, closed: closeErr == nil && statErr == nil, digest: digest}
		resultErr = errors.Join(resultErr, statErr, closeErr)
	}()
	hash := sha256.New()
	writer := &exportWriter{file: file, ctx: ctx, total: int64(size)}
	count, err := io.Copy(io.MultiWriter(writer, hash), io.LimitReader(contextReader{ctx, reader}, int64(size)+1))
	if err != nil {
		return err
	}
	if count != int64(size) || hex.EncodeToString(hash.Sum(nil)) != digest {
		return errors.New("exported bootstrap bytes differ from frozen identity")
	}
	if err := file.Chmod(0600); err != nil {
		return err
	}
	return file.Sync()
}

func (o *ownedOutput) read(name string, limit int64) (body []byte, resultErr error) {
	file, err := o.root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	info, err := file.Stat()
	owned, found := o.files[name]
	if err != nil || !found || !owned.closed || !ownedRegular(info) || !os.SameFile(owned.info, info) || info.Size() > limit {
		return nil, errors.New("bootstrap metadata changed")
	}
	body, err = io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(body)) > limit {
		return nil, errors.Join(errors.New("bootstrap metadata exceeds bound"), err)
	}
	return body, nil
}

func (o *ownedOutput) sync() (resultErr error) {
	if err := o.check(); err != nil {
		return err
	}
	file, err := o.root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(file.Sync(), file.Close())
}
func (o *ownedOutput) removeIntake() error {
	if err := o.check(); err != nil {
		return err
	}
	info, err := o.root.Lstat("intake")
	if err != nil || !privateDirectory(info) || !os.SameFile(o.intake, info) {
		return errors.New("bootstrap intake identity changed")
	}
	return o.root.Remove("intake") // Empty only; never recursively sweep unknown Runs.
}
func (o *ownedOutput) discardFiles() error {
	if err := o.check(); err != nil {
		return err
	}
	names := make([]string, 0, len(o.files))
	for name := range o.files {
		names = append(names, name)
	}
	sort.Strings(names)
	var resultErr error
	for _, name := range names {
		expected := o.files[name]
		info, err := o.root.Lstat(name)
		if err != nil || !expected.closed || !ownedRegular(info) || !os.SameFile(expected.info, info) || expected.info.Size() != info.Size() || expected.info.Mode() != info.Mode() || expected.info.ModTime() != info.ModTime() {
			resultErr = errors.Join(resultErr, errors.New("unknown bootstrap output retained"))
			continue
		}
		resultErr = errors.Join(resultErr, o.root.Remove(name))
	}
	return errors.Join(resultErr, o.sync())
}
func (o *ownedOutput) close() error {
	if !o.closed {
		o.closed = true
		o.closeErr = o.root.Close()
	}
	return o.closeErr
}
func (o *ownedOutput) removeDirectory() error {
	if err := o.check(); err != nil {
		return err
	}
	return os.Remove(o.path) // Fails on unknown remaining members.
}

func (o *ownedOutput) verifyAll(ctx context.Context) error {
	if err := o.check(); err != nil {
		return err
	}
	for name, expected := range o.files {
		file, err := o.root.Open(name)
		if err != nil {
			return err
		}
		info, err := file.Stat()
		if err != nil || !expected.closed || !ownedRegular(info) || !os.SameFile(expected.info, info) || expected.info.Size() != info.Size() || expected.info.Mode() != info.Mode() || expected.info.ModTime() != info.ModTime() {
			_ = file.Close()
			return errors.New("prepared output changed before handoff")
		}
		hash := sha256.New()
		count, readErr := io.Copy(hash, io.LimitReader(contextReader{ctx, file}, info.Size()+1))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil || count != info.Size() || hex.EncodeToString(hash.Sum(nil)) != expected.digest {
			return errors.Join(errors.New("prepared output differs from verified digest"), readErr, closeErr)
		}
	}
	return ctx.Err()
}
