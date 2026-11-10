package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSemgrepBuildOutputRejectsOverflow(t *testing.T) {
	var data bytes.Buffer
	bounded := &semgrepBuildOutput{&data, 3}
	if _, err := io.Copy(bounded, strings.NewReader("abc")); err != nil {
		t.Fatal(err)
	}
	if n, err := bounded.Write([]byte("d")); err == nil || n != 0 || data.String() != "abc" {
		t.Fatal("overflow changed bounded output", n, err)
	}
}

func TestSemgrepBuildSDKValidatesFrozenDescriptor(t *testing.T) {
	valid := `{"base_image":"docker.io/library/alpine@sha256:` + strings.Repeat("a", 64) + `","architecture":"amd64","opam_version":"2.5.2","note":"fixture","packages":[{"name":"bash-5.3.3-r1.apk","bytes":3,"sha256":"` + hash([]byte("apk")) + `"}]}`
	for name, body := range map[string]string{
		"valid":               valid,
		"unknown field":       strings.Replace(valid, `"note":`, `"unexpected":`, 1),
		"duplicate field":     strings.Replace(valid, `"architecture":"amd64"`, `"architecture":"amd64","architecture":"amd64"`, 1),
		"remote mutable base": strings.Replace(valid, "@sha256:"+strings.Repeat("a", 64), ":latest", 1),
		"escape":              strings.Replace(valid, "bash-5.3.3-r1.apk", "../bash.apk", 1),
		"oversized APK":       strings.Replace(valid, `"bytes":3`, `"bytes":134217729`, 1),
		"wrong architecture":  strings.Replace(valid, "amd64", "arm64", 1),
		"trailing data":       valid + ` {}`,
	} {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.Mkdir(filepath.Join(directory, "build"), 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(directory, "build/sdk-inputs.json")
			if err := os.WriteFile(path, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(directory)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			selected := semgrepSourcePackageInput{Files: []semgrepSourcePackageFile{{"build/sdk-inputs.json", hash([]byte(body)), int64(len(body)), 0444}}}
			sdk, err := readSemgrepBuildSDK(root, selected)
			if name == "valid" {
				if err != nil || len(sdk.Packages) != 1 {
					t.Fatal(sdk, err)
				}
			} else if err == nil {
				t.Fatal("invalid SDK accepted")
			}
			if name == "valid" {
				selected.Files[0].SHA256 = strings.Repeat("0", 64)
				if _, err := readSemgrepBuildSDK(root, selected); err == nil {
					t.Fatal("changed SDK bytes accepted")
				}
			}
		})
	}
}

func TestSemgrepBuildRejectsUnsafePathsBeforeHostUse(t *testing.T) {
	for _, args := range [][]string{{}, {"--input", "relative", "--output", "/tmp/semgrep-core-linux-amd64"}, {"--expected-sha256", "bad"}} {
		var out bytes.Buffer
		if err := rebuildSemgrep(args, &out); err == nil || out.Len() != 0 {
			t.Fatal("invalid build reached output", args, err)
		}
	}
}
