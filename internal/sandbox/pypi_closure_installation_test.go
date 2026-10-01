package sandbox

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func testClosureTar(t *testing.T, files map[string][]byte, uid int) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := tar.NewWriter(&buffer)
	for name, body := range files {
		if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Uid: uid, Gid: uid, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestClosureInstallationScannerBindsRecordAndGeneratedFiles(t *testing.T) {
	body := []byte("x = 1\n")
	digest := sha256.Sum256(body)
	manifest := closureManifest{
		identity: "frozen", artifactCount: 1,
		files: map[string]closureFile{
			"demo.py":                   {path: "demo.py", size: int64(len(body)), sha256: hex.EncodeToString(digest[:]), scheme: artifactpypi.SchemeSite},
			"demo-1.0.dist-info/RECORD": {path: "demo-1.0.dist-info/RECORD", scheme: artifactpypi.SchemeDistInfo},
		},
	}
	valid := map[string][]byte{
		"demo.py":                      body,
		"demo-1.0.dist-info/RECORD":    []byte("demo.py,..."),
		"demo-1.0.dist-info/INSTALLER": []byte("pip\n"),
	}
	installed, err := scanClosureTar(bytes.NewReader(testClosureTar(t, valid, 1000)), manifest, 1024)
	if err != nil || installed.identity == "" || len(installed.files) != 3 {
		t.Fatalf("valid installed closure rejected: %v", err)
	}
	bad := map[string][]byte{}
	for name, value := range valid {
		bad[name] = value
	}
	bad["demo.py"] = []byte("x = 2\n")
	if _, err := scanClosureTar(bytes.NewReader(testClosureTar(t, bad, 1000)), manifest, 1024); err == nil {
		t.Fatal("RECORD hash mismatch accepted")
	}
	bad["demo.py"] = body
	bad["surprise.py"] = body
	if _, err := scanClosureTar(bytes.NewReader(testClosureTar(t, bad, 1000)), manifest, 1024); err == nil {
		t.Fatal("unowned installer output accepted")
	}
	if _, err := scanClosureTar(bytes.NewReader(testClosureTar(t, valid, 0)), manifest, 1024); err == nil {
		t.Fatal("wrong installed ownership accepted")
	}
}

func TestClosureInstallationAccountsOnlyDeclaredGeneratedWrapper(t *testing.T) {
	manifest := closureManifest{
		identity: "frozen", artifactCount: 1,
		files:            map[string]closureFile{"demo-1.0.dist-info/RECORD": {path: "demo-1.0.dist-info/RECORD", scheme: artifactpypi.SchemeDistInfo}},
		generatedScripts: map[string]string{"bin/demo": "demo.cli"},
	}
	files := map[string][]byte{
		"demo-1.0.dist-info/RECORD": []byte("record"),
		"bin/demo":                  []byte("#!/usr/bin/python\nfrom demo.cli import main\n"),
	}
	if _, err := scanClosureTar(bytes.NewReader(testClosureTar(t, files, 1000)), manifest, 1024); err != nil {
		t.Fatal(err)
	}
	delete(files, "bin/demo")
	if _, err := scanClosureTar(bytes.NewReader(testClosureTar(t, files, 1000)), manifest, 1024); err == nil {
		t.Fatal("missing declared wrapper accepted")
	}
	files["bin/other"] = []byte("unexpected")
	if _, err := scanClosureTar(bytes.NewReader(testClosureTar(t, files, 1000)), manifest, 1024); err == nil {
		t.Fatal("undeclared wrapper accepted")
	}
}
