package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// This official PyPI JSON repeats the wheel's public MD5 in two fields. The
// pinned scanner mistakes its prefix for a payment token. Retain the original
// bytes and every rule; neither a path nor a line alone grants an exemption.
const publicPygmentsMetadata = "tools/pipx/supply/pygments/metadata.json"
const publicPygmentsMetadataSHA256 = "330104a10adc8a587e3f98f4e207557125ca0c1d489e129334da5acb2b813a03"

type publicMetadataFinding struct {
	RuleID, File, SymlinkFile, Commit          string
	StartLine, EndLine, StartColumn, EndColumn int
}

func (c *checker) runGitleaksWithPublicMetadata(step, executable string, arguments ...string) error {
	directory, err := os.MkdirTemp("", "helox-gitleaks-report-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	report := filepath.Join(directory, "redacted.json")
	args := append(append([]string(nil), arguments...), "--report-format=json", "--report-path", report)
	output, scanErr := c.runCommandWithTimeout(step, 5*time.Minute, c.offlineEnvironment(), executable, args...)
	if scanErr == nil {
		return nil
	}
	var exitError *exec.ExitError
	if !errors.As(scanErr, &exitError) || exitError.ExitCode() != 1 || strings.Contains(strings.ToLower(output), "invalid configuration") {
		return scanErr
	}
	original := &checkFailure{class: findingFailure, step: step, detail: strings.TrimSpace(output)}
	body, err := readPublicMetadataBounded(report)
	if err != nil {
		return original
	}
	var findings []publicMetadataFinding
	if json.Unmarshal(body, &findings) != nil || len(findings) == 0 {
		return original
	}
	if err := c.verifyPublicMetadataFindings(findings); err != nil {
		return original
	}
	fmt.Fprintf(c.stdout, "[check] verified %d exact public wheel MD5 false positives; all scanner rules retained\n", len(findings))
	return nil
}

func readPublicMetadataBounded(filename string) ([]byte, error) {
	info, err := os.Lstat(filename)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return nil, errors.New("public metadata/report is not a bounded regular file")
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(body) > 1<<20 {
		return nil, errors.Join(errors.New("public metadata/report read failed"), readErr, closeErr)
	}
	return body, nil
}

func (c *checker) verifyPublicMetadataFindings(findings []publicMetadataFinding) error {
	if len(findings) == 0 {
		return errors.New("public metadata findings missing")
	}
	groups := map[string]map[int]bool{}
	for _, f := range findings {
		if f.RuleID != "square-access-token" || filepath.ToSlash(f.File) != publicPygmentsMetadata || f.SymlinkFile != "" || f.StartLine != 1 || f.EndLine != 1 || (f.StartColumn != 3155 && f.StartColumn != 3357) || f.EndColumn != f.StartColumn+32 {
			return errors.New("finding is outside exact public metadata")
		}
		if groups[f.Commit] == nil {
			groups[f.Commit] = map[int]bool{}
		}
		if groups[f.Commit][f.StartColumn] {
			return errors.New("duplicate public metadata finding")
		}
		groups[f.Commit][f.StartColumn] = true
	}
	for commit, positions := range groups {
		if len(positions) != 2 {
			return errors.New("public metadata findings incomplete")
		}
		var body []byte
		var err error
		if commit == "" {
			body, err = readPublicMetadataBounded(filepath.Join(c.root, filepath.FromSlash(publicPygmentsMetadata)))
		} else {
			if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(commit) {
				return errors.New("historical finding commit is invalid")
			}
			git, lookErr := exec.LookPath(executableName("git"))
			if lookErr != nil {
				return lookErr
			}
			var output string
			output, err = c.runCommandWithTimeout("Gitleaks exact public metadata blob", time.Minute, c.offlineEnvironment(), git, "show", commit+":"+publicPygmentsMetadata)
			body = []byte(output)
		}
		if err != nil {
			return err
		}
		digest := sha256.Sum256(body)
		if hex.EncodeToString(digest[:]) != publicPygmentsMetadataSHA256 {
			return errors.New("public metadata original bytes differ")
		}
	}
	return nil
}
