package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
	if !bytes.Equal(generated, render(lock)) {
		t.Fatal("generated runtime identity differs from the canonical lock")
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
			name: "extra mismatched define",
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
