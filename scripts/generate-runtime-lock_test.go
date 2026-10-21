package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeLockAndGeneratedIdentityAreCurrent(t *testing.T) {
	t.Parallel()
	lock, err := readLock(filepath.Join("runtimes.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(filepath.Join("..", outputPath))
	if err != nil {
		t.Fatal(err)
	}
	qualification, err := readPyTorchVersions("version-support.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, render(lock, qualification)) {
		t.Fatal("generated runtime identity differs from the canonical lock")
	}
}

func TestRuntimeLockRejectsImageVersionAndABIDrift(t *testing.T) {
	base, err := readLock("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*runtimeLock){
		"Node floating tag":  func(lock *runtimeLock) { lock.NodeImage.Reference = "node:latest@sha256:" + fmt.Sprintf("%064d", 0) },
		"Python tag version": func(lock *runtimeLock) { lock.PythonImage.PythonVersion = "3.13.0" },
		"interpreter tag":    func(lock *runtimeLock) { lock.PythonImage.Target.Interpreter = "cp313" },
		"ABI tag":            func(lock *runtimeLock) { lock.PythonImage.Target.ABI = "cp313" },
		"Python platform":    func(lock *runtimeLock) { lock.PythonImage.Target.Platform = "manylinux_2_36_aarch64" },
	} {
		t.Run(name, func(t *testing.T) {
			lock := base
			mutate(&lock)
			if err := validate(lock); err == nil {
				t.Fatal("mismatched runtime target accepted")
			}
		})
	}
}

func TestPyTorchVersionOwnersRejectMissingDuplicateAndProfileSubstitution(t *testing.T) {
	for name, records := range map[string]string{
		"missing":         `[]`,
		"duplicate":       `[{"component":"PyTorch CPU","pinned_version":"2.14.0+cpu"},{"component":"PyTorch CPU","pinned_version":"2.14.0+cpu"}]`,
		"wrong profile":   `[{"component":"PyTorch cu126","pinned_version":"2.14.0+cu130"}]`,
		"unknown profile": `[{"component":"PyTorch cu999","pinned_version":"2.14.0+cu999"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "support.json")
			if err := os.WriteFile(path, []byte(`{"schema_version":1,"records":`+records+`}`), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := readPyTorchVersions(path); err == nil {
				t.Fatal("invalid qualification version owner accepted")
			}
		})
	}
}

func TestRuntimeLockRejectsUnknownFieldAndMissingPlatform(t *testing.T) {
	t.Parallel()
	current, err := os.ReadFile("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	unknown := filepath.Join(directory, "unknown.json")
	if err := os.WriteFile(unknown, append(current[:len(current)-2], []byte(",\n  \"unexpected\": true\n}\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readLock(unknown); err == nil {
		t.Fatal("unknown runtime lock field was accepted")
	}

	lock, err := readLock("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	lock.GVisor.RuntimeBundle.Architecture = "aarch64"
	if err := validate(lock); err == nil {
		t.Fatal("unexpected runtime bundle architecture was accepted")
	}
}

func TestRuntimeLockRejectsUnboundUVSupply(t *testing.T) {
	base, err := readLock("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*runtimeLock){
		"old schema": func(l *runtimeLock) { l.SchemaVersion = 1 },
		"foreign publisher": func(l *runtimeLock) {
			l.UVImage.Reference = strings.Replace(l.UVImage.Reference, "astral-sh", "foreign", 1)
		},
		"different version":       func(l *runtimeLock) { l.UVImage.Version = "0.0.1" },
		"missing source":          func(l *runtimeLock) { l.UVImage.Provenance.SourceCommit = "" },
		"missing bundle":          func(l *runtimeLock) { l.UVImage.Provenance.BundleSHA256 = "" },
		"missing index":           func(l *runtimeLock) { l.UVImage.Provenance.IndexSHA256 = "" },
		"wrong architecture":      func(l *runtimeLock) { l.UVImage.Architecture = "arm64" },
		"foreign path":            func(l *runtimeLock) { l.UVImage.Tool.Path = "/other" },
		"oversize":                func(l *runtimeLock) { l.UVImage.Tool.Size = 64<<20 + 1 },
		"missing native identity": func(l *runtimeLock) { l.UVImage.Tool.SHA256 = "" },
	} {
		t.Run(name, func(t *testing.T) {
			l := base
			mutate(&l)
			if err := validate(l); err == nil {
				t.Fatal("unbound UV supply accepted")
			}
		})
	}
}

func TestRuntimeLockRejectsInvalidPatchIdentity(t *testing.T) {
	t.Parallel()
	baseLock, err := readLock("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}

	invalidCases := []struct {
		name   string
		path   string
		sha256 string
	}{
		{name: "missing path", path: "", sha256: baseLock.GVisor.Patch.SHA256},
		{name: "path outside tools/gvisor", path: "scripts/patch.patch", sha256: baseLock.GVisor.Patch.SHA256},
		{name: "directory traversal path", path: "tools/gvisor/../../etc/test.patch", sha256: baseLock.GVisor.Patch.SHA256},
		{name: "non-patch extension", path: "tools/gvisor/patch.txt", sha256: baseLock.GVisor.Patch.SHA256},
		{name: "missing sha256", path: baseLock.GVisor.Patch.Path, sha256: ""},
		{name: "short sha256", path: baseLock.GVisor.Patch.Path, sha256: "abc"},
		{name: "sha512 instead of sha256", path: baseLock.GVisor.Patch.Path, sha256: "4463ce276e207f5a516a08ec627a768a19cf7bed0094d522b0810bee3424585caa8d344e093204012b974f5c508ab2362dcb0d7236f0c1992fccc426beeb7ffc"},
		{name: "invalid characters in sha256", path: baseLock.GVisor.Patch.Path, sha256: "z98f802d74a6ee42e4090957373ec30c432b64b7106a589f94bfdd1f384f8162"},
	}

	for _, tc := range invalidCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			badLock := baseLock
			badLock.GVisor.Patch.Path = tc.path
			badLock.GVisor.Patch.SHA256 = tc.sha256
			if err := validate(badLock); err == nil {
				t.Fatalf("expected error for case %q, but got nil", tc.name)
			}
		})
	}
}

func TestRuntimeLockRejectsInvalidBazelModuleLockIdentity(t *testing.T) {
	t.Parallel()
	lock, err := readLock("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	lock.GVisor.Build.BazelModuleLockSHA256 = "not-a-sha256"
	if err := validate(lock); err == nil {
		t.Fatal("invalid gVisor Bazel module lock identity was accepted")
	}
}

func TestRuntimeLockRejectsInvalidBuilderIdentity(t *testing.T) {
	t.Parallel()
	base, err := readLock("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		edit func(*runtimeLock)
	}{
		{"missing digest", func(l *runtimeLock) { l.GVisor.Build.Builder.ImageDigest = "" }},
		{"non-SHA-256 digest", func(l *runtimeLock) {
			l.GVisor.Build.Builder.ImageDigest = "sha512:" + l.GVisor.Build.Builder.ImageDigest[7:]
		}},
		{"mutable tag only", func(l *runtimeLock) { l.GVisor.Build.Builder.ImageDigest = l.GVisor.Build.Builder.ImageTag }},
		{"wrong architecture", func(l *runtimeLock) { l.GVisor.Build.Builder.Architecture = "arm64" }},
		{"unrelated repository", func(l *runtimeLock) { l.GVisor.Build.Builder.ImageRepository = "example.com/builder" }},
		{"invalid tag", func(l *runtimeLock) { l.GVisor.Build.Builder.ImageTag = "latest" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			lock := base
			test.edit(&lock)
			if err := validate(lock); err == nil {
				t.Fatal("invalid builder identity was accepted")
			}
		})
	}
}

func TestRuntimeLockRejectsUnpinnedOrSubstitutedGoToolchain(t *testing.T) {
	base, err := readLock("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		edit func(*runtimeLock)
	}{
		{"mutable", func(l *runtimeLock) { l.GoImage.Reference = "golang:1.26.8-bookworm" }},
		{"repository", func(l *runtimeLock) { l.GoImage.Reference = "example.com/" + l.GoImage.Reference }},
		{"architecture", func(l *runtimeLock) { l.GoImage.Architecture = "arm64" }},
		{"version", func(l *runtimeLock) { l.GoImage.GoVersion = "1.26.7" }},
		{"digest", func(l *runtimeLock) { l.GoImage.Reference = "golang:1.26.8-bookworm@sha256:invalid" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			lock := base
			test.edit(&lock)
			if err := validate(lock); err == nil {
				t.Fatal("untrusted Go runtime pin accepted")
			}
		})
	}
}

func TestRuntimeLockRejectsUnpinnedOrSubstitutedCargoToolchain(t *testing.T) {
	base, err := readLock("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		edit func(*runtimeLock)
	}{
		{"mutable", func(l *runtimeLock) { l.RustImage.Reference = "rust:1.99.0-bookworm" }},
		{"repository", func(l *runtimeLock) { l.RustImage.Reference = "example.com/" + l.RustImage.Reference }},
		{"architecture", func(l *runtimeLock) { l.RustImage.Architecture = "arm64" }},
		{"version", func(l *runtimeLock) { l.RustImage.RustVersion = "1.98.0" }},
		{"target", func(l *runtimeLock) { l.RustImage.Target = "aarch64-unknown-linux-gnu" }},
		{"digest", func(l *runtimeLock) { l.RustImage.Reference = "rust:1.99.0-bookworm@sha256:invalid" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			lock := base
			test.edit(&lock)
			if err := validate(lock); err == nil {
				t.Fatal("untrusted Cargo runtime pin accepted")
			}
		})
	}
}

func TestRuntimeLockObserverBuildIdentityMatchesCanonicalCommit(t *testing.T) {
	t.Parallel()
	lock, err := readLock("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	buildPath := filepath.Join("..", observerBuildPath)
	buildContent, err := os.ReadFile(buildPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyObserverBuildCommit(buildContent, lock.GVisor.Commit); err != nil {
		t.Fatalf("observer BUILD identity mismatch against canonical runtime lock: %v", err)
	}
}

func TestVerifyObserverBuildCommitDetectsDrift(t *testing.T) {
	t.Parallel()
	canonical := "7c6199801fd233d6d55309af4645d4746a077de7"
	stale := "5ceb9a5fd5750d6c73dd166441f28306039300d0"

	validBuild := []byte(fmt.Sprintf(`
cc_binary(
    name = "haa_gvisor_observer",
    srcs = ["observer.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"%s\\\""],
)
cc_binary(
    name = "haa_gvisor_observer_latch_test",
    srcs = ["observer_latch_test.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"%s\\\""],
)
`, canonical, canonical))

	if err := verifyObserverBuildCommit(validBuild, canonical); err != nil {
		t.Fatalf("expected valid build to pass, got: %v", err)
	}

	for _, tc := range []struct {
		name         string
		buildContent []byte
		commit       string
	}{
		{
			name: "stale haa_gvisor_observer target",
			buildContent: []byte(fmt.Sprintf(`
cc_binary(
    name = "haa_gvisor_observer",
    srcs = ["observer.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"%s\\\""],
)
cc_binary(
    name = "haa_gvisor_observer_latch_test",
    srcs = ["observer_latch_test.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"%s\\\""],
)
`, stale, canonical)),
			commit: canonical,
		},
		{
			name: "stale haa_gvisor_observer_latch_test target",
			buildContent: []byte(fmt.Sprintf(`
cc_binary(
    name = "haa_gvisor_observer",
    srcs = ["observer.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"%s\\\""],
)
cc_binary(
    name = "haa_gvisor_observer_latch_test",
    srcs = ["observer_latch_test.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"%s\\\""],
)
`, canonical, stale)),
			commit: canonical,
		},
		{
			name: "commented haa_gvisor_observer with valid commit but active target missing define",
			buildContent: []byte(fmt.Sprintf(`
# cc_binary(
#     name = "haa_gvisor_observer",
#     defines = ["HAA_GVISOR_COMMIT=\"%s\""],
# )
cc_binary(
    name = "haa_gvisor_observer",
    srcs = ["observer.cc"],
)
cc_binary(
    name = "haa_gvisor_observer_latch_test",
    srcs = ["observer_latch_test.cc"],
    defines = ["HAA_GVISOR_COMMIT=\"%s\""],
)
`, canonical, canonical)),
			commit: canonical,
		},
		{
			name: "commented haa_gvisor_observer_latch_test with valid commit but active target missing define",
			buildContent: []byte(fmt.Sprintf(`
cc_binary(
    name = "haa_gvisor_observer",
    srcs = ["observer.cc"],
    defines = ["HAA_GVISOR_COMMIT=\"%s\""],
)
# cc_binary(
#     name = "haa_gvisor_observer_latch_test",
#     defines = ["HAA_GVISOR_COMMIT=\"%s\""],
# )
cc_binary(
    name = "haa_gvisor_observer_latch_test",
    srcs = ["observer_latch_test.cc"],
)
`, canonical, canonical)),
			commit: canonical,
		},
		{
			name: "commented haa_gvisor_observer with valid commit and active target missing entirely",
			buildContent: []byte(fmt.Sprintf(`
# cc_binary(
#     name = "haa_gvisor_observer",
#     defines = ["HAA_GVISOR_COMMIT=\"%s\""],
# )
cc_binary(
    name = "haa_gvisor_observer_latch_test",
    srcs = ["observer_latch_test.cc"],
    defines = ["HAA_GVISOR_COMMIT=\"%s\""],
)
`, canonical, canonical)),
			commit: canonical,
		},
		{
			name: "commented haa_gvisor_observer_latch_test with valid commit and active target missing entirely",
			buildContent: []byte(fmt.Sprintf(`
cc_binary(
    name = "haa_gvisor_observer",
    srcs = ["observer.cc"],
    defines = ["HAA_GVISOR_COMMIT=\"%s\""],
)
# cc_binary(
#     name = "haa_gvisor_observer_latch_test",
#     defines = ["HAA_GVISOR_COMMIT=\"%s\""],
# )
`, canonical, canonical)),
			commit: canonical,
		},
		{
			name: "active target with duplicate defines",
			buildContent: []byte(fmt.Sprintf(`
cc_binary(
    name = "haa_gvisor_observer",
    defines = [
        "HAA_GVISOR_COMMIT=\"%s\"",
        "HAA_GVISOR_COMMIT=\"%s\"",
    ],
)
cc_binary(
    name = "haa_gvisor_observer_latch_test",
    defines = ["HAA_GVISOR_COMMIT=\"%s\""],
)
`, canonical, canonical, canonical)),
			commit: canonical,
		},
		{
			name: "duplicate active targets with same name",
			buildContent: []byte(fmt.Sprintf(`
cc_binary(
    name = "haa_gvisor_observer",
    defines = ["HAA_GVISOR_COMMIT=\"%s\""],
)
cc_binary(
    name = "haa_gvisor_observer",
    defines = ["HAA_GVISOR_COMMIT=\"%s\""],
)
cc_binary(
    name = "haa_gvisor_observer_latch_test",
    defines = ["HAA_GVISOR_COMMIT=\"%s\""],
)
`, canonical, canonical, canonical)),
			commit: canonical,
		},
		{
			name: "missing define in haa_gvisor_observer",
			buildContent: []byte(fmt.Sprintf(`
cc_binary(
    name = "haa_gvisor_observer",
    srcs = ["observer.cc"],
)
cc_binary(
    name = "haa_gvisor_observer_latch_test",
    srcs = ["observer_latch_test.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"%s\\\""],
)
`, canonical)),
			commit: canonical,
		},
		{
			name: "missing target haa_gvisor_observer_latch_test",
			buildContent: []byte(fmt.Sprintf(`
cc_binary(
    name = "haa_gvisor_observer",
    srcs = ["observer.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"%s\\\""],
)
`, canonical)),
			commit: canonical,
		},
		{
			name: "malformed define in target",
			buildContent: []byte(fmt.Sprintf(`
cc_binary(
    name = "haa_gvisor_observer",
    defines = ["HAA_GVISOR_COMMIT=not-a-valid-40-hex-hash"],
)
cc_binary(
    name = "haa_gvisor_observer_latch_test",
    defines = ["HAA_GVISOR_COMMIT=\"%s\""],
)
`, canonical)),
			commit: canonical,
		},
		{
			name: "extra mismatched define in third target",
			buildContent: []byte(fmt.Sprintf(`
cc_binary(
    name = "haa_gvisor_observer",
    srcs = ["observer.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"%s\\\""],
)
cc_binary(
    name = "haa_gvisor_observer_latch_test",
    srcs = ["observer_latch_test.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"%s\\\""],
)
cc_binary(
    name = "extra_target",
    defines = ["HAA_GVISOR_COMMIT=\\\"%s\\\""],
)
`, canonical, canonical, stale)),
			commit: canonical,
		},
		{
			name:         "invalid short commit length",
			buildContent: validBuild,
			commit:       "7c6199801fd2",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := verifyObserverBuildCommit(tc.buildContent, tc.commit); err == nil {
				t.Fatalf("expected error for case %q, but got nil", tc.name)
			}
		})
	}
}

func TestVerifyObserverBuildCommitValidWithCommentsAndStrings(t *testing.T) {
	t.Parallel()
	canonical := "7c6199801fd233d6d55309af4645d4746a077de7"

	build := []byte(fmt.Sprintf(`
# Top level comment with cc_binary( name = "fake" )
cc_binary( # comment with parens () and quotes ""
    name = "haa_gvisor_observer", # comment
    srcs = ["observer(1).cc"],
    defines = ["HAA_GVISOR_COMMIT=\"%s\""],
)
cc_library(
    name = "other_rule",
    textual_hdrs = ["observer.cc"],
)
cc_binary(
    name = 'haa_gvisor_observer_latch_test',
    defines = ['HAA_GVISOR_COMMIT="%s"'], # trailing comment
)
`, canonical, canonical))

	if err := verifyObserverBuildCommit(build, canonical); err != nil {
		t.Fatalf("expected build with comments and string parens to pass, got: %v", err)
	}
}

func TestRuntimeLockDigestUsesCompleteBoundedInput(t *testing.T) {
	body, err := os.ReadFile("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := readLock("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if lock.lockSHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("compiled lock digest differs from source bytes")
	}
	path := filepath.Join(t.TempDir(), "lock.json")
	changed := append(append([]byte(nil), body...), '\n')
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	other, err := readLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if other.lockSHA256 == lock.lockSHA256 {
		t.Fatal("raw lock byte change was ignored")
	}
	oversized := append(append([]byte(nil), body...), bytes.Repeat([]byte(" "), 64*1024+1-len(body))...)
	if err := os.WriteFile(path, oversized, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readLock(path); err == nil {
		t.Fatal("oversized lock prefix accepted")
	}
}
