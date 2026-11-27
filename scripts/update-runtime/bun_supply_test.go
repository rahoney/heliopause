package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBunFrozenNoticesBindSourceAndRemainPassiveData(t *testing.T) {
	source := strings.Repeat("a", 40)
	body := []byte("Bun upstream source: https://github.com/oven-sh/bun/tree/" + source + "\n\nreviewed upstream license text\n")
	if !validBunNotices(source, body) {
		t.Fatal("valid source-bound notices refused")
	}
	for _, invalid := range [][]byte{nil, []byte("unbound license"), append(append([]byte(nil), body...), 0), append(append([]byte(nil), body...), 27), []byte(strings.Repeat("x", (64<<10)+1))} {
		if validBunNotices(source, invalid) {
			t.Fatal("unbounded or active control bytes accepted")
		}
	}
	if validBunNotices(strings.Repeat("b", 40), body) {
		t.Fatal("notices from another selected source accepted")
	}
}

func TestBunNoopDoesNotAcquireSupply(t *testing.T) {
	root, state := fixture(t)
	version, err := selectedVersion(root, "bun")
	if err != nil {
		t.Fatal(err)
	}
	called := false
	p, err := makePlanResolving(root, state, "bun", version, "", func(component, version string) (metadata, error) {
		called = true
		return metadata{}, errors.New("must not acquire during no-op")
	})
	if err != nil || called || len(p.Changes) != 0 || p.Metadata.Bun != nil {
		t.Fatalf("no-op altered or acquired supply: %+v %v", p, err)
	}
	if err := applyPlan(p, state, &bytes.Buffer{}, nil); err != nil {
		t.Fatal(err)
	}
	if err := applyPlan(p, state, &bytes.Buffer{}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestBunSupplyFlagsAreScopedAndExplicit(t *testing.T) {
	for _, name := range []string{"same-version-proof", "foreign-component", "implicit-network"} {
		t.Run(name, func(t *testing.T) {
			root, state := fixture(t)
			version, err := selectedVersion(root, "bun")
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"plan", "--root", root, "--state", state, "--component", "bun", "--version", version, "--bun-proof", filepath.Join(state, "unavailable")}
			switch name {
			case "foreign-component":
				args[6] = "node"
			case "implicit-network":
				args = []string{"plan", "--root", root, "--state", state, "--component", "bun", "--version", "999.0.0"}
			}
			var output bytes.Buffer
			if err := run(args, &output); err == nil || output.Len() != 0 {
				t.Fatalf("unsupported supply produced a plan: %v %s", err, output.String())
			}
		})
	}
}

func TestBunSelectedOwnerRejectsAmbiguousFields(t *testing.T) {
	for _, name := range []string{"duplicate", "case-alias", "unknown"} {
		t.Run(name, func(t *testing.T) {
			root, _ := fixture(t)
			path := filepath.Join(root, runtimeFile)
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "duplicate":
				body = bytes.Replace(body, []byte(`"bun_version":`), []byte(`"bun_version":"0.0.1","bun_version":`), 1)
			case "case-alias":
				body = bytes.Replace(body, []byte(`"bun_version":`), []byte(`"Bun_Version":`), 1)
			case "unknown":
				body = bytes.Replace(body, []byte(`"bun_version":`), []byte(`"unexpected":true,"bun_version":`), 1)
			}
			writeTest(t, path, body)
			if _, err := selectedVersion(root, "bun"); err == nil {
				t.Fatal("ambiguous Bun owner accepted")
			}
		})
	}
}
