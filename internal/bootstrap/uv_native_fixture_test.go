package bootstrap_test

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// This fixture creates foreign installed DATA with the exact native UV binary.
// It never executes an artifact or a target interpreter on the host. Product
// enrollment subsequently performs its own public old-wheel authentication.
func uvNativeOriginalFixture(t *testing.T, ctx context.Context, wheel []byte, target string) {
	t.Helper()
	const label = "io.heliopause.test.uv-native-original"
	client := t.TempDir()
	ownerDigest := sha256.Sum256([]byte(client))
	owner := hex.EncodeToString(ownerDigest[:])
	docker := func(callCtx context.Context, input []byte, limit int, args ...string) ([]byte, error) {
		command := exec.CommandContext(callCtx, "/usr/bin/docker", append([]string{"--config", client, "--host", "unix:///run/docker.sock"}, args...)...)
		command.Stdin = bytes.NewReader(input)
		out, diagnostic := &uvFixtureOutput{limit: limit}, &uvFixtureOutput{limit: 16 << 10}
		command.Stdout, command.Stderr = out, diagnostic
		if err := command.Run(); err != nil {
			return nil, fmt.Errorf("fixture docker %v: %w: %s", args, err, diagnostic.body.String())
		}
		return out.body.Bytes(), nil
	}
	checked := func(input []byte, limit int, args ...string) []byte {
		t.Helper()
		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		body, err := docker(callCtx, input, limit, args...)
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	type containerIdentity struct {
		ID     string `json:"Id"`
		Image  string
		Config struct{ Labels map[string]string }
	}
	owned := map[string]string{}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for id, image := range owned {
			body, err := docker(cleanupCtx, nil, 64<<10, "inspect", id)
			var identities []containerIdentity
			if err != nil || json.Unmarshal(body, &identities) != nil || len(identities) != 1 || identities[0].ID != id || identities[0].Image != image || identities[0].Config.Labels[label] != owner {
				t.Errorf("native fixture cleanup identity unavailable: %s %v", id, err)
				continue
			}
			if _, err := docker(cleanupCtx, nil, 16<<10, "rm", "--force", id); err != nil {
				t.Error(err)
			}
		}
		body, err := docker(cleanupCtx, nil, 16<<10, "ps", "--all", "--quiet", "--filter", "label="+label+"="+owner)
		if err != nil || len(bytes.TrimSpace(body)) != 0 {
			t.Errorf("native fixture containers remain or query failed: %s %v", body, err)
		}
	})
	create := func(reference, config string, command []string, args ...string) string {
		t.Helper()
		arguments := append([]string{"create", "--pull", "never", "--platform", "linux/amd64", "--label", label + "=" + owner}, args...)
		arguments = append(arguments, reference)
		arguments = append(arguments, command...)
		id := strings.TrimSpace(string(checked(nil, 16<<10, arguments...)))
		decoded, err := hex.DecodeString(id)
		if err != nil || len(decoded) != 32 {
			t.Fatal("invalid fixture container identity")
		}
		owned[id] = "sha256:" + config
		var identities []containerIdentity
		body := checked(nil, 64<<10, "inspect", id)
		if json.Unmarshal(body, &identities) != nil || len(identities) != 1 || identities[0].ID != id || identities[0].Config.Labels[label] != owner {
			t.Fatal("fixture image/container identity mismatch")
		}
		if config == "" {
			owned[id] = identities[0].Image
		} else if identities[0].Image != owned[id] {
			t.Fatal("fixture locked image config mismatch")
		}
		return id
	}
	// Verify the locked UV supply image and copy its binary strictly as data.
	supply := create(runtimeidentity.UVImageReference, runtimeidentity.UVConfigSHA256, nil, "--network", "none", "--read-only", "--cap-drop", "ALL")
	archive := checked(nil, runtimeidentity.UVToolSize+(64<<10), "cp", supply+":"+runtimeidentity.UVToolPath, "-")
	reader := tar.NewReader(bytes.NewReader(archive))
	h, err := reader.Next()
	if err != nil || h.Name != "uv" || !h.FileInfo().Mode().IsRegular() || h.Mode != 0755 || h.Uid != 0 || h.Gid != 0 || h.Size != runtimeidentity.UVToolSize {
		t.Fatal("invalid locked UV supply tar", err)
	}
	native, err := io.ReadAll(io.LimitReader(reader, runtimeidentity.UVToolSize+1))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(native)
	if len(native) != runtimeidentity.UVToolSize || hex.EncodeToString(sum[:]) != runtimeidentity.UVToolSHA256 {
		t.Fatal("locked UV supply digest mismatch")
	}
	if _, err := reader.Next(); !errors.Is(err, io.EOF) {
		t.Fatal("unexpected UV supply tar entry", err)
	}
	// Explicit runc is only a fixture data generator; the product still uses
	// its normal authenticated gVisor observer and network-policy path.
	container := create(runtimeidentity.PythonImageReference, "", []string{"infinity"},
		"--runtime", "runc", "--network", "none", "--read-only", "--user", "1000:1000", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", "512m", "--cpus", "1",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=128m,uid=1000,gid=1000,mode=0700",
		"--tmpfs", "/haa-fixture:rw,nosuid,nodev,size=96m,uid=1000,gid=1000,mode=0700", "--entrypoint", "/bin/sleep")
	checked(nil, 16<<10, "start", container)
	// Only the verified UV binary is executable in this private tmpfs. The
	// authentic wheel and local registry HTML are copied as inert input data.
	const filename = "idna-3.10-py3-none-any.whl"
	wheelSum := sha256.Sum256(wheel)
	var input bytes.Buffer
	writer := tar.NewWriter(&input)
	for _, directory := range []string{"index/", "index/packages/", "index/simple/", "index/simple/idna/"} {
		if err := writer.WriteHeader(&tar.Header{Name: directory, Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}); err != nil {
			t.Fatal(err)
		}
	}
	for _, entry := range []struct {
		name string
		body []byte
		mode int64
	}{
		{"uv", native, 0555},
		{"index/packages/" + filename, wheel, 0444},
		{"index/simple/idna/index.html", []byte("<a href=\"../../packages/" + filename + "#sha256=" + hex.EncodeToString(wheelSum[:]) + "\">" + filename + "</a>"), 0444},
	} {
		if err := writer.WriteHeader(&tar.Header{Name: entry.name, Mode: entry.mode, Size: int64(len(entry.body)), Uid: 1000, Gid: 1000}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	checked(input.Bytes(), 16<<10, "cp", "--archive", "-", container+":/haa-fixture")
	execute := func(args ...string) []byte {
		t.Helper()
		arguments := []string{"exec", "--user", "1000:1000", container, "/usr/bin/env", "-i", "PATH=/usr/local/bin:/usr/bin:/bin", "HOME=/tmp", "LANG=C", "LC_ALL=C"}
		return checked(nil, 64<<10, append(arguments, args...)...)
	}
	checked(nil, 16<<10, "exec", "--detach", "--user", "1000:1000", container, "/usr/local/bin/python", "-I", "-B", "-m", "http.server", "8123", "--bind", "127.0.0.1", "--directory", "/haa-fixture/index")
	execute("/usr/local/bin/python", "-I", "-B", "-c", `import urllib.request,time
for i in range(20):
 try:
  with urllib.request.urlopen("http://127.0.0.1:8123/simple/idna/",timeout=1) as r: r.read(4096)
  break
 except OSError:
  if i==19: raise
  time.sleep(0.05)`)
	execute("/haa-fixture/uv", "venv", "--no-config", "--no-cache", "--no-python-downloads", "--no-managed-python", "--python", "/usr/local/bin/python", "/tmp/native-venv")
	execute("/haa-fixture/uv", "pip", "install", "--no-build", "--no-config", "--no-cache", "--no-python-downloads", "--no-managed-python", "--index-url", "http://127.0.0.1:8123/simple/", "--python", "/tmp/native-venv/bin/python", "idna==3.10")
	cfg := execute("/usr/local/bin/python", "-I", "-B", "-c", `import pathlib,sys; sys.stdout.buffer.write(pathlib.Path("/tmp/native-venv/pyvenv.cfg").read_bytes())`)
	version := strings.Split(runtimeidentity.PythonVersion, ".")
	if len(version) != 3 {
		t.Fatal("invalid locked Python version")
	}
	siteRelative := "lib/python" + version[0] + "." + version[1] + "/site-packages"
	site := filepath.Join(target, filepath.FromSlash(siteRelative))
	if err := os.MkdirAll(site, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(target, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "pyvenv.cfg"), cfg, 0600); err != nil {
		t.Fatal(err)
	}
	installed := checked(nil, 4<<20, "cp", container+":/tmp/native-venv/"+siteRelative+"/.", "-")
	reader = tar.NewReader(bytes.NewReader(installed))
	seen := map[string]bool{}
	for count := 0; ; count++ {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(strings.TrimPrefix(header.Name, "./"), "/")
		if name == "." && header.Typeflag == tar.TypeDir {
			continue
		}
		if count >= 256 || name == "" || path.Clean(name) != name || !filepath.IsLocal(name) || strings.ContainsAny(name, "\\\x00") || seen[name] || header.Uid != 1000 || header.Gid != 1000 || header.Mode & ^int64(0777) != 0 || header.Size < 0 || header.Size > 1<<20 {
			t.Fatal("unsafe native installed data", header.Name)
		}
		seen[name] = true
		destination := filepath.Join(site, filepath.FromSlash(name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destination, 0700); err != nil {
				t.Fatal(err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
				t.Fatal(err)
			}
			file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				t.Fatal(err)
			}
			_, copyErr := io.CopyN(file, reader, header.Size)
			closeErr := file.Close()
			if err := errors.Join(copyErr, closeErr); err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatal("unsupported native fixture file type", header.Name)
		}
	}
	marker, err := os.ReadFile(filepath.Join(site, "idna-3.10.dist-info", "INSTALLER"))
	if err != nil || string(marker) != "uv" {
		t.Fatal("actual UV installer format changed", err)
	}
	t.Logf("actual_uv_native_original version=3.10 files=%d uv=%s python=%s artifacts_executed=0", len(seen), runtimeidentity.UVVersion, runtimeidentity.PythonVersion)
}

type uvFixtureOutput struct {
	body  bytes.Buffer
	limit int
}

func (o *uvFixtureOutput) Write(body []byte) (int, error) {
	if len(body) > o.limit-o.body.Len() {
		return 0, errors.New("native fixture output exceeds bound")
	}
	return o.body.Write(body)
}
