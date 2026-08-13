package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (c *checker) checkFormat() error {
	files, err := ownedGoFiles(c.root)
	if err != nil {
		return err
	}
	output, err := c.runCommand("format check", c.gofmt, append([]string{"-l"}, files...)...)
	if err != nil {
		return err
	}
	if unformatted := strings.TrimSpace(output); unformatted != "" {
		return &checkFailure{class: findingFailure, step: "format check", detail: unformatted}
	}
	return nil
}

func (c *checker) applyFormat() error {
	files, err := ownedGoFiles(c.root)
	if err != nil {
		return err
	}
	_, err = c.runCommand("format", c.gofmt, append([]string{"-w"}, files...)...)
	return err
}

func ownedGoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && excludedSourceDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) != ".go" {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return &checkFailure{class: findingFailure, step: "source inventory", detail: fmt.Sprintf("Go source is a symbolic link: %s", path)}
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		var failure *checkFailure
		if errors.As(err, &failure) {
			return nil, failure
		}
		return nil, &checkFailure{class: executionFailure, step: "source inventory", cause: err}
	}
	if len(files) == 0 {
		return nil, &checkFailure{class: unavailable, step: "source inventory", detail: "no repository-owned Go files found"}
	}
	sort.Strings(files)
	return files, nil
}

func excludedSourceDirectory(name string) bool {
	switch name {
	case ".git", "vendor", "testdata":
		return true
	default:
		return false
	}
}
