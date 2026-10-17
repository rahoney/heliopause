package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type sdkReceipt struct {
	PlanID  string      `json:"plan_id"`
	Archive archive     `json:"archive"`
	Files   []inputFile `json:"files"`
}

func prepareSDK(p plan, state string) (string, error) {
	if err := validateMetadata("go", p.Version, p.Metadata); err != nil {
		return "", err
	}
	a, err := hostArchive(p.Metadata)
	if err != nil {
		return "", err
	}
	sdk := filepath.Join(state, "sdk")
	goBin := filepath.Join(sdk, "go/bin/go")
	var receipt sdkReceipt
	if err := loadJSON(filepath.Join(state, "sdk-receipt.json"), &receipt); err == nil {
		if receipt.PlanID != p.ID || receipt.Archive != a {
			return "", errors.New("target SDK receipt mismatch")
		}
		files, err := sdkInventory(sdk)
		if err != nil {
			return "", err
		}
		bodyA, bodyB := fmt.Sprint(files), fmt.Sprint(receipt.Files)
		if bodyA != bodyB {
			return "", errors.New("target SDK contents changed")
		}
		return goBin, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if _, err := os.Lstat(sdk); err == nil {
		return "", errors.New("SDK directory exists without a completed receipt; preserve and investigate it")
	}
	archivePath := filepath.Join(state, "go-sdk.tar.gz")
	if err := verifyArchive(archivePath, a); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if err := downloadArchive(archivePath, a); err != nil {
			return "", err
		}
	}
	staging, err := os.MkdirTemp(state, "sdk-extract-")
	if err != nil {
		return "", err
	}
	// Failed extraction stays outside source for diagnosis. No ambient SDK is changed.
	if err := extractSDK(archivePath, staging); err != nil {
		return "", fmt.Errorf("target SDK extraction retained for diagnosis: %w", err)
	}
	if err := checkGo(filepath.Join(staging, "go/bin/go"), filepath.Join(state, "candidate")); err != nil {
		return "", err
	}
	files, err := sdkInventory(staging)
	if err != nil {
		return "", err
	}
	if err := os.Rename(staging, sdk); err != nil {
		return "", err
	}
	if err := saveJSON(filepath.Join(state, "sdk-receipt.json"), sdkReceipt{p.ID, a, files}); err != nil {
		return "", err
	}
	return goBin, nil
}
func verifyArchive(path string, a archive) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, a.Size+1))
	if err != nil {
		return err
	}
	if n != a.Size || hex.EncodeToString(h.Sum(nil)) != a.SHA256 {
		return errors.New("target SDK archive size/SHA256 mismatch (preserved)")
	}
	return nil
}
func downloadArchive(path string, a archive) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
	if err != nil {
		return err
	}
	client := officialHTTP()
	client.Timeout = 10 * time.Minute
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("official SDK download status %d", resp.StatusCode)
	}
	f, err := os.CreateTemp(filepath.Dir(path), "sdk-download-")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer f.Close()
	if _, err := io.Copy(f, io.LimitReader(resp.Body, a.Size+1)); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := verifyArchive(temp, a); err != nil {
		return err
	}
	return os.Rename(temp, path)
}
func extractSDK(path, root string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	seen := map[string]bool{}
	var expanded int64
	for count := 0; ; count++ {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if count >= 30000 {
			return errors.New("SDK archive file count bound exceeded")
		}
		name := strings.TrimSuffix(h.Name, "/")
		if filepath.IsAbs(name) || filepath.Clean(name) != name || !strings.HasPrefix(name, "go/") && name != "go" || strings.Contains(name, "\\") || seen[name] {
			return errors.New("unsafe/duplicate SDK archive path")
		}
		seen[name] = true
		full := filepath.Join(root, name)
		if !within(root, full) {
			return errors.New("SDK archive path escapes root")
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(full, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if h.Size < 0 || h.Size > 128<<20 {
				return errors.New("SDK archive member bound exceeded")
			}
			expanded += h.Size
			if expanded > 1<<30 {
				return errors.New("SDK expanded archive bound exceeded")
			}
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if h.Mode&0o111 != 0 {
				mode = 0o755
			}
			out, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(out, tr, h.Size)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return errors.New("SDK links and special files are refused")
		}
	}
	return nil
}
func sdkInventory(root string) ([]inputFile, error) {
	files := []inputFile{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		body, mode, err := readRegular(root, rel)
		if err != nil {
			return err
		}
		files = append(files, inputFile{rel, hash(body), uint32(mode.Perm())})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, err
}
