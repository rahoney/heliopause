package terraformprovider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectCapturesLocalModulesAndParsesExactLock(t *testing.T) {
	root := t.TempDir()
	if e := os.Mkdir(filepath.Join(root, "child"), 0o700); e != nil {
		t.Fatal(e)
	}
	for name, body := range map[string]string{
		"main.tf":                "terraform {\n required_providers {\n random = { source = \"hashicorp/random\", version = \"~> 3.7.0\" }\n }\n}\nmodule \"child\" { source = \"./child\" }\nresource \"random_integer\" \"test\" {}\n",
		"child/versions.tf.json": `{"terraform":{"required_providers":{"github":{"source":"integrations/github","version":">= 6.0.0, < 7.0.0"}}}}`,
	} {
		if e := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); e != nil {
			t.Fatal(e)
		}
	}
	capture, e := CaptureProject(context.Background(), root)
	if e != nil {
		t.Fatal(e)
	}
	reqs, e := Requirements(capture.Controls)
	if e != nil || len(reqs) != 2 {
		t.Fatalf("complete requirements=%+v %v", reqs, e)
	}
	if capture.Members["child"] == nil || capture.Members["child/versions.tf.json"] == nil || capture.Members[LockControl] != nil {
		t.Fatal("local module/absent lock identity lost")
	}
	providers := []LockedProvider{{Address: "registry.terraform.io/hashicorp/random", Version: "3.7.2", Constraints: "~> 3.7.0", Hashes: []string{"zh:" + strings.Repeat("a", 64)}}}
	lock, e := EncodeLock(providers)
	if e != nil {
		t.Fatal(e)
	}
	parsed, e := ParseLock(lock)
	if e != nil || len(parsed) != 1 || parsed[0].Version != "3.7.2" {
		t.Fatalf("lock=%+v %v", parsed, e)
	}
	contents := PackageContents{ZH: "zh:" + strings.Repeat("a", 64), H1: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}
	if e := VerifyPackageLock(parsed[0], contents); e != nil {
		t.Fatal(e)
	}
	contents.ZH = "zh:" + strings.Repeat("b", 64)
	if e := VerifyPackageLock(parsed[0], contents); e == nil {
		t.Fatal("mismatched lock accepted")
	}
}

func TestTerraformControlAliasesRemoteAndAmbiguousSelection(t *testing.T) {
	base := "terraform {\n required_providers {\n random = { source = \"hashicorp/random\" }\n }\n}\n"
	for _, c := range []struct{ name, body string }{
		{"remote", base + "module \"bad\" { source = \"hashicorp/test/aws\" }\n"},
		{"escape", base + "module \"bad\" { source = \"../escape\" }\n"},
		{"duplicate", base + base},
		{"implicit", "resource \"random_integer\" \"test\" {}\n"},
		{"expression", "terraform {\n required_providers {\n random = { source = var.source }\n }\n}\n"},
		{"private", "terraform {\n required_providers {\n random = { source = \"private.example/hashicorp/random\" }\n }\n}\n"},
		{"deep", strings.Repeat("{", 65) + strings.Repeat("}", 65)},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			if e := os.WriteFile(filepath.Join(root, "main.tf"), []byte(c.body), 0o600); e != nil {
				t.Fatal(e)
			}
			if _, e := CaptureProject(context.Background(), root); e == nil {
				t.Fatal("unsupported configuration accepted")
			}
		})
	}
	for _, alias := range []string{"symlink", "hardlink", "directory", "override", "case"} {
		t.Run(alias, func(t *testing.T) {
			root := t.TempDir()
			p := filepath.Join(root, "main.tf")
			if e := os.WriteFile(p, []byte(base), 0o600); e != nil {
				t.Fatal(e)
			}
			var e error
			switch alias {
			case "symlink":
				e = os.Symlink("main.tf", filepath.Join(root, "other.tf"))
			case "hardlink":
				e = os.Link(p, filepath.Join(root, "other.tf"))
			case "directory":
				e = os.Mkdir(filepath.Join(root, "other.tf"), 0o700)
			case "override":
				e = os.WriteFile(filepath.Join(root, "override.tf"), []byte(base), 0o600)
			case "case":
				e = os.WriteFile(filepath.Join(root, "OTHER.TF"), []byte(base), 0o600)
			}
			if e != nil {
				t.Fatal(e)
			}
			if _, e = CaptureProject(context.Background(), root); e == nil {
				t.Fatal("aliased input accepted")
			}
		})
	}
	for _, lock := range []string{
		`provider "registry.terraform.io/hashicorp/random" { version = "3.7.2" }`,
		`provider "private.example/hashicorp/random" { version = "3.7.2" hashes = [] }`,
		"provider \"registry.terraform.io/hashicorp/random\" {\n version = \"3.7.2\"\n hashes = [\"zh:" + strings.Repeat("a", 64) + "\", \"zh:" + strings.Repeat("a", 64) + "\"]\n}",
		"provider \"registry.terraform.io/hashicorp/random\" {\n version = var.version\n hashes = [\"zh:" + strings.Repeat("a", 64) + "\"]\n}",
	} {
		if _, e := ParseLock([]byte(lock)); e == nil {
			t.Fatal("ambiguous lock accepted")
		}
	}
}

func TestTerraformVersionConstraints(t *testing.T) {
	for _, c := range []struct {
		version, constraint string
		want                bool
	}{
		{"3.7.2", "~> 3.7.0", true}, {"3.8.0", "~> 3.7.0", false}, {"3.99.2", "~> 3.7", true}, {"4.0.0", "~> 3.7", false}, {"6.6.0", ">= 6.0.0, < 7.0.0", true}, {"6.6.0", "!= 6.6.0", false}, {"6.6.0", "latest", false}, {"1.2.3-01", "", false},
	} {
		if got := MatchesConstraint(c.version, c.constraint); got != c.want {
			t.Errorf("%s %s=%v", c.version, c.constraint, got)
		}
	}
}
