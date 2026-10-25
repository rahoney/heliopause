package cli

import (
	"errors"
	"io"
	"os"
	"syscall"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func readProjectPreflightControl(root *os.Root, name string, optional bool, limit int64, client string) (control domain.ProjectControlFile, resultErr error) {
	before, err := root.Lstat(name)
	if optional && errors.Is(err, os.ErrNotExist) {
		return domain.NewProjectControlFile(name, nil, false)
	}
	regular := func(info os.FileInfo) bool {
		if info == nil || !info.Mode().IsRegular() || info.Mode()&^os.ModePerm != 0 || info.Size() < 0 || info.Size() > limit {
			return false
		}
		s, ok := info.Sys().(*syscall.Stat_t)
		return ok && s.Nlink == 1
	}
	if err != nil || !regular(before) {
		return control, errors.Join(errors.New(client+" control must be bounded single-link regular content"), err)
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return control, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, f.Close())
		if resultErr != nil {
			control = domain.ProjectControlFile{}
		}
	}()
	opened, err := f.Stat()
	if err != nil || !regular(opened) || !os.SameFile(before, opened) {
		return control, errors.New(client + " control changed while opening")
	}
	body, readErr := io.ReadAll(io.LimitReader(f, limit+1))
	after, statErr := f.Stat()
	current, currentErr := root.Lstat(name)
	if readErr != nil || statErr != nil || currentErr != nil || !regular(after) || !regular(current) || !os.SameFile(before, after) || !os.SameFile(before, current) || after.Mode() != before.Mode() || current.Mode() != before.Mode() || int64(len(body)) != before.Size() || after.Size() != before.Size() || current.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) || !current.ModTime().Equal(before.ModTime()) {
		return control, errors.Join(errors.New(client+" control changed during preflight"), readErr, statErr, currentErr)
	}
	return domain.NewProjectControlFile(name, body, true)
}
