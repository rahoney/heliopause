//go:build linux

// Package releaseactivation owns the narrow protected installation lifecycle.
// Signed identity, kernel caller identity and the old controller's quarantine
// approval are separate requirements. Local state grants no Policy approval.
package releaseactivation

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"

	"golang.org/x/mod/semver"
)

const (
	stateRoot          = "/var/lib/heliopause"
	registrationPath   = stateRoot + "/protected-installation.json"
	journalPath        = stateRoot + "/protected-update.json"
	runtimeLockPath    = stateRoot + "/runtime.lock"
	proofRoot          = stateRoot + "/proofs"
	registrationSchema = "helox.protected-installation/v1"
)

var (
	ErrUnmanaged  = errors.New("SELF_UPDATE_MANAGED_INSTALLATION_REQUIRED")
	ErrTransition = errors.New("SELF_UPDATE_TRANSITION_OR_RECOVERY_REQUIRED")
	ErrUntrusted  = errors.New("SELF_UPDATE_INSTALLATION_IDENTITY_INVALID")
)

type registration struct {
	Schema         string `json:"schema"`
	Management     string `json:"management"`
	Version        string `json:"version"`
	ManifestSHA256 string `json:"manifest_sha256"`
	ClientUID      uint32 `json:"client_uid"`
	ClientGID      uint32 `json:"client_gid"`
}

func parseRegistration(body []byte) (registration, error) {
	var record registration
	if len(body) == 0 || len(body) > 16<<10 || !utf8.Valid(body) {
		return record, ErrUntrusted
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return record, ErrUntrusted
	}
	seen := map[string]bool{}
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return record, ErrUntrusted
		}
		name, ok := key.(string)
		if !ok || seen[name] {
			return record, ErrUntrusted
		}
		seen[name] = true
		var destination any
		switch name {
		case "schema":
			destination = &record.Schema
		case "management":
			destination = &record.Management
		case "version":
			destination = &record.Version
		case "manifest_sha256":
			destination = &record.ManifestSHA256
		case "client_uid":
			destination = &record.ClientUID
		case "client_gid":
			destination = &record.ClientGID
		default:
			return record, ErrUntrusted
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil || bytes.Equal(bytes.TrimSpace(value), []byte("null")) || json.Unmarshal(value, destination) != nil {
			return record, ErrUntrusted
		}
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') || len(seen) != 6 {
		return record, ErrUntrusted
	}
	if _, err := decoder.Token(); err != io.EOF {
		return record, ErrUntrusted
	}
	if record.Schema != registrationSchema || record.Management != "NATIVE" || !exactVersion(record.Version) || !sha256Text(record.ManifestSHA256) || record.ClientUID == 0 || record.ClientUID > 1<<31-1 || record.ClientGID > 1<<31-1 {
		return record, ErrUntrusted
	}
	return record, nil
}

func exactVersion(value string) bool {
	core, _, _ := strings.Cut(value, "-")
	core, _, _ = strings.Cut(core, "+")
	return len(value) <= 128 && semver.IsValid(value) && strings.Count(core, ".") == 2
}

func sha256Text(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func protectedDirectory(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return ErrUntrusted
	}
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0022 != 0 {
			return ErrUntrusted
		}
		owner, ok := info.Sys().(*syscall.Stat_t)
		if !ok || owner.Uid != 0 {
			return ErrUntrusted
		}
		if current == filepath.Dir(current) {
			return nil
		}
	}
}

func openProtectedFile(path string) (*os.File, os.FileInfo, error) {
	if err := protectedDirectory(filepath.Dir(path)); err != nil {
		return nil, nil, err
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode().Perm()&0022 != 0 || before.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
		return nil, nil, ErrUntrusted
	}
	owner, ok := before.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Nlink != 1 {
		return nil, nil, ErrUntrusted
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, nil, ErrUntrusted
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) {
		_ = file.Close()
		return nil, nil, ErrUntrusted
	}
	return file, opened, nil
}

func readProtected(ctx context.Context, path string, limit int64) (body []byte, resultErr error) {
	if ctx == nil {
		return nil, ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, info, err := openProtectedFile(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if file.Close() != nil {
			resultErr = errors.Join(resultErr, ErrUntrusted)
		}
	}()
	if info.Size() <= 0 || info.Size() > limit {
		return nil, ErrUntrusted
	}
	body, err = io.ReadAll(io.LimitReader(protectedContextReader{ctx, file}, limit+1))
	if err != nil {
		return nil, errors.Join(ErrUntrusted, ctx.Err())
	}
	if int64(len(body)) != info.Size() || int64(len(body)) > limit {
		return nil, ErrUntrusted
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, current) {
		return nil, ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return body, nil
}

type protectedContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r protectedContextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}
