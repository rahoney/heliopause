package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/blakesmith/ar"
	rpmutils "github.com/sassoftware/go-rpmutils"
)

const testEpoch int64 = 1792000000

type packageFixture struct{ assets, output, lock, license string }
type packageMember struct {
	body               []byte
	mode               int64
	owner, group, link string
	mtime              int64
}

func fixture(t *testing.T) packageFixture {
	t.Helper()
	root := t.TempDir()
	f := packageFixture{filepath.Join(root, "assets"), filepath.Join(root, "output"), filepath.Join(root, "runtimes.lock.json"), filepath.Join(root, "LICENSE")}
	for _, p := range []string{f.assets, f.output} {
		if err := os.Mkdir(p, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	inputs := map[string]string{filepath.Join(f.assets, "helox-linux-amd64"): "cli-native-payload", filepath.Join(f.assets, "haa_gvisor_observer-linux-amd64"): "observer-native-payload", filepath.Join(f.assets, "haa-network-policy-helper-linux-amd64"): "policy-native-payload", filepath.Join(f.assets, "helox-runtime-images.json"): `{"schema":"helox.runtime-image-manifest/v1"}`, f.lock: `{"schema_version":1}`, f.license: "Apache License Version 2.0"}
	inputs[filepath.Join(f.assets, "haa-release-installer-linux-amd64")] = "installer-native-payload"
	inputs[filepath.Join(f.assets, "pnpm-linux-amd64")] = "pnpm-native-payload"
	inputs[filepath.Join(f.assets, "pnpm-NOTICE.txt")] = "pnpm-notices-payload"
	inputs[filepath.Join(f.assets, "bun-linux-amd64")] = "bun-native-payload"
	inputs[filepath.Join(f.assets, "bun-NOTICE.txt")] = "bun-notices-payload"
	inputs[filepath.Join(f.assets, "pipx-runtime-linux-amd64.zip")] = "pipx-data-payload"
	inputs[filepath.Join(f.assets, "pipx-NOTICE.txt")] = "pipx-notices-payload"
	for p, b := range inputs {
		if err := os.WriteFile(p, []byte(b), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func TestPackagesContainBoundedPayloadAndAreReproducible(t *testing.T) {
	f := fixture(t)
	if err := buildPackages("v1.2.3", testEpoch, f.assets, f.output, f.lock, f.license); err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(t.TempDir(), "second")
	if err := os.Mkdir(second, 0o700); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(f.assets)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if err := os.Chtimes(filepath.Join(f.assets, e.Name()), time.Unix(testEpoch+86400, 0), time.Unix(testEpoch+86400, 0)); err != nil {
			t.Fatal(err)
		}
	}
	if err := buildPackages("v1.2.3", testEpoch, f.assets, second, f.lock, f.license); err != nil {
		t.Fatal(err)
	}
	for _, format := range packageFormats {
		first := readTest(t, filepath.Join(f.output, "helox-linux-amd64."+format))
		again := readTest(t, filepath.Join(second, "helox-linux-amd64."+format))
		if !bytes.Equal(first, again) {
			t.Fatalf("%s differs after input mtime and temporary path changes", format)
		}
		var members map[string]packageMember
		if format == "deb" {
			members = readDeb(t, first)
		} else {
			members = readRPM(t, first)
		}
		root := "usr/lib/heliopause/releases/v1.2.3/"
		for _, want := range []struct {
			name, body string
			mode       int64
		}{{"helox", "cli-native-payload", 0o555}, {"haa_gvisor_observer", "observer-native-payload", 0o555}, {"haa-network-policy-helper", "policy-native-payload", 0o500}, {"haa-release-installer", "installer-native-payload", 0o500}, {"pnpm", "pnpm-native-payload", 0o555}, {"pnpm-NOTICE.txt", "pnpm-notices-payload", 0o444}, {"bun", "bun-native-payload", 0o555}, {"bun-NOTICE.txt", "bun-notices-payload", 0o444}, {"pipx-runtime-linux-amd64.zip", "pipx-data-payload", 0o444}, {"pipx-NOTICE.txt", "pipx-notices-payload", 0o444}} {
			m, ok := members[root+want.name]
			if !ok || string(m.body) != want.body || m.mode&0o7777 != want.mode || m.owner != "root" || m.group != "root" || m.mtime != testEpoch {
				t.Fatalf("%s payload %s = %#v", format, want.name, m)
			}
		}
		if m := members["usr/bin/helox"]; m.link != "/"+root+"helox" {
			t.Fatalf("%s CLI link = %#v", format, m)
		}
		unit := string(members["usr/lib/systemd/system/helox-network-policy.service"].body)
		for _, required := range []string{"ConditionPathExists=/etc/heliopause/network-policy.json", "ExecStart=/" + root + "haa-network-policy-helper serve\n", "RuntimeDirectory=heliopause-network-policy\n", "NoNewPrivileges=yes", "Restart=no"} {
			if !strings.Contains(unit, required) {
				t.Fatalf("%s service lacks %q", format, required)
			}
		}
		for _, forbidden := range []string{"etc/heliopause/network-policy.json", "usr/libexec/heliopause/runsc", "usr/libexec/heliopause/haa_gvisor_observer"} {
			if _, ok := members[forbidden]; ok {
				t.Fatalf("package activates uncontrolled runtime/caller config: %s", forbidden)
			}
		}
		for name, m := range members {
			if strings.Contains(name, "../") || strings.HasPrefix(name, "/") || (m.link == "" && m.mode&0o6022 != 0) {
				t.Fatalf("unsafe member %q mode %o", name, m.mode)
			}
		}
	}
}

func TestPackageInputRejectionPreservesExistingOutputs(t *testing.T) {
	for _, name := range []string{"short-version", "path-version", "zero-time", "negative-time", "overflow-time", "relative-assets", "symlink-assets", "symlink-input", "missing-installer", "empty-input", "oversized-input", "existing-output"} {
		t.Run(name, func(t *testing.T) {
			f := fixture(t)
			version := "v1.2.3"
			epoch := testEpoch
			switch name {
			case "short-version":
				version = "v1.2"
			case "path-version":
				version = "v1.2.3/../../other"
			case "zero-time":
				epoch = 0
			case "negative-time":
				epoch = -1
			case "overflow-time":
				epoch = 1 << 32
			case "relative-assets":
				f.assets = "relative"
			case "symlink-assets":
				p := filepath.Join(t.TempDir(), "link")
				if err := os.Symlink(f.assets, p); err != nil {
					t.Fatal(err)
				}
				f.assets = p
			case "symlink-input":
				p := filepath.Join(f.assets, "helox-linux-amd64")
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(f.license, p); err != nil {
					t.Fatal(err)
				}
			case "empty-input":
				if err := os.WriteFile(f.license, nil, 0o600); err != nil {
					t.Fatal(err)
				}
			case "missing-installer":
				if err := os.Remove(filepath.Join(f.assets, "haa-release-installer-linux-amd64")); err != nil {
					t.Fatal(err)
				}
			case "oversized-input":
				file, err := os.OpenFile(f.license, os.O_WRONLY, 0)
				if err != nil {
					t.Fatal(err)
				}
				err = file.Truncate(maxInputBytes + 1)
				closeErr := file.Close()
				if err != nil || closeErr != nil {
					t.Fatal(err, closeErr)
				}
			case "existing-output":
				if err := os.WriteFile(filepath.Join(f.output, "helox-linux-amd64.rpm"), []byte("keep"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := buildPackages(version, epoch, f.assets, f.output, f.lock, f.license); err == nil {
				t.Fatal("invalid package input accepted")
			}
			entries, err := os.ReadDir(f.output)
			if err != nil {
				t.Fatal(err)
			}
			if name == "existing-output" {
				if len(entries) != 1 || string(readTest(t, filepath.Join(f.output, "helox-linux-amd64.rpm"))) != "keep" {
					t.Fatal("existing output changed")
				}
			} else if len(entries) != 0 {
				t.Fatal("invalid input produced partial output")
			}
		})
	}
}

func readTest(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func readDeb(t *testing.T, body []byte) map[string]packageMember {
	t.Helper()
	r := ar.NewReader(bytes.NewReader(body))
	parts := map[string][]byte{}
	for {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		parts[strings.TrimSuffix(h.Name, "/")] = data
	}
	if string(parts["debian-binary"]) != "2.0\n" || len(parts) != 3 {
		t.Fatal("not a bounded Debian 2.0 package")
	}
	control := readTar(t, parts["control.tar.gz"])
	metadata := string(control["control"].body)
	for _, required := range []string{"Package: helox\n", "Version: 1.2.3-1\n", "Architecture: amd64\n"} {
		if !strings.Contains(metadata, required) {
			t.Fatalf("deb metadata lacks %q: %s", required, metadata)
		}
	}
	post := string(control["postinst"].body)
	if !strings.Contains(post, "helox doctor") {
		t.Fatal("postinstall lacks doctor guidance")
	}
	for _, bad := range []string{"systemctl", "curl", "wget", "docker pull", "sudo"} {
		if strings.Contains(post, bad) {
			t.Fatalf("postinstall contains %s", bad)
		}
	}
	return readTar(t, parts["data.tar.gz"])
}

func readTar(t *testing.T, body []byte) map[string]packageMember {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	r := tar.NewReader(gz)
	result := map[string]packageMember{}
	for {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Uid != 0 || h.Gid != 0 {
			t.Fatal("non-root archive owner")
		}
		if h.Typeflag == tar.TypeDir {
			continue
		}
		data, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimPrefix(h.Name, "./")
		if _, ok := result[name]; ok {
			t.Fatal("duplicate tar member")
		}
		result[name] = packageMember{data, h.Mode, "root", "root", h.Linkname, h.ModTime.Unix()}
	}
	return result
}

func readRPM(t *testing.T, body []byte) map[string]packageMember {
	t.Helper()
	r, err := rpmutils.ReadRpm(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for tag, want := range map[int]string{rpmutils.NAME: "helox", rpmutils.VERSION: "1.2.3", rpmutils.RELEASE: "1", rpmutils.ARCH: "x86_64", rpmutils.BUILDHOST: "reproducible.helox"} {
		got, err := r.Header.GetString(tag)
		if err != nil || got != want {
			t.Fatalf("rpm tag %d = %q / %v", tag, got, err)
		}
	}
	post, err := r.Header.GetString(rpmutils.POSTIN)
	if err != nil || !strings.Contains(post, "helox doctor") {
		t.Fatal("RPM postinstall lacks doctor guidance", err)
	}
	for _, bad := range []string{"systemctl", "curl", "wget", "docker pull", "sudo"} {
		if strings.Contains(post, bad) {
			t.Fatalf("RPM postinstall contains %s", bad)
		}
	}
	reader, err := r.PayloadReaderExtended()
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]packageMember{}
	for {
		f, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if f.Mode()&0o170000 == 0o040000 {
			continue
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimPrefix(f.Name(), "/")
		if _, ok := result[name]; ok {
			t.Fatal("duplicate RPM member")
		}
		result[name] = packageMember{data, int64(f.Mode()), f.UserName(), f.GroupName(), f.Linkname(), int64(f.Mtime())}
	}
	return result
}
