//go:build realcorpus

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPipxActualPackagesPreserveSelectedData(t *testing.T) {
	root := os.Getenv("HELOX_PIPX_RELEASE_ACTUAL_DIR")
	if root == "" {
		t.Skip("explicit complete local producer candidate required")
	}
	// Local unsigned fixture version matches the existing independent package
	// readers; it is not a tag, published release or package installation.
	versionRoot := "usr/lib/heliopause/releases/v1.2.3/"
	for _, format := range packageFormats {
		body := readTest(t, filepath.Join(root, "helox-linux-amd64."+format))
		var members map[string]packageMember
		if format == "deb" {
			members = readDeb(t, body)
		} else {
			members = readRPM(t, body)
		}
		for _, name := range []string{"pipx-runtime-linux-amd64.zip", "pipx-NOTICE.txt"} {
			expected := readTest(t, filepath.Join(root, name))
			member, ok := members[versionRoot+name]
			if !ok || !bytes.Equal(member.body, expected) || member.mode&07777 != 0444 || member.owner != "root" || member.group != "root" {
				t.Fatal("selected DATA payload/mode/owner mismatch", format, name)
			}
		}
		notice, ok := members["usr/share/doc/helox/pipx-NOTICE.txt"]
		if !ok || !bytes.Equal(notice.body, readTest(t, filepath.Join(root, "pipx-NOTICE.txt"))) || notice.mode&07777 != 0444 {
			t.Fatal("packaged notice differs", format)
		}
		for _, name := range []string{"etc/heliopause/network-policy.json", "usr/libexec/heliopause/haa_gvisor_observer"} {
			if _, ok := members[name]; ok {
				t.Fatal("package activates protected Host", format, name)
			}
		}
		t.Logf("actual %s: selected ZIP/NOTICE bytes and root0444 payload/documentation matched; no package installation", format)
	}
}
