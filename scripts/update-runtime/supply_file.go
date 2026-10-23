package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// readFrozenSupplyFile bounds ordinary untrusted proof inputs before cryptographic verification.
func readFrozenSupplyFile(directory, name string, limit int64) ([]byte, error) {
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() {
		return nil, errors.New("frozen supply directory is unavailable or a symlink")
	}
	location := filepath.Join(directory, name)
	before, err := os.Lstat(location)
	if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > limit {
		return nil, errors.New("frozen supply file is unavailable or exceeds bound")
	}
	file, err := os.Open(location)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) || after.Size() != before.Size() {
		return nil, errors.New("frozen supply file changed")
	}
	body, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(body)) != before.Size() {
		return nil, errors.New("frozen supply read failed or changed")
	}
	return body, nil
}
