package pypi

import (
	"bytes"
	"strings"
	"testing"
)

func TestRequirementsFilePreservesNativeLogicalLinesAndLocations(t *testing.T) {
	body := []byte("\xef\xbb\xbf# header\\\r\nrequests [security] >= 2.8.1, \\\r\n == 2.8.* ; python_version < \"3.15\" # comment\r\n-r \"nested/base requirements.txt\"\r\n--constraint=constraints.txt\r\nfoo\\\n# stop continuation\\\nbar\u00a0# unicode whitespace\nend\\")
	original := bytes.Clone(body)
	file, err := ParseRequirementsFile(body)
	if err != nil {
		t.Fatal(err)
	}
	want := []RequirementsLine{
		{Number: 2, Text: `requests [security] >= 2.8.1,  == 2.8.* ; python_version < "3.15"`},
		{Number: 4, Text: `-r "nested/base requirements.txt"`, Include: "nested/base requirements.txt"},
		{Number: 5, Text: "--constraint=constraints.txt", Include: "constraints.txt", Constraint: true},
		{Number: 6, Text: "foo"},
		{Number: 8, Text: "bar"},
		{Number: 9, Text: "end"},
	}
	got := file.Lines()
	if len(got) != len(want) {
		t.Fatalf("logical lines=%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d=%+v, want %+v", i, got[i], want[i])
		}
	}
	if !bytes.Equal(original, body) {
		t.Fatal("parser changed original bytes")
	}
	body[0] = '!'
	got[0].Text = "changed"
	if file.Lines()[0] != want[0] {
		t.Fatal("frozen source mutated")
	}
}

func TestRequirementsFileRefusesUnsafeIncludesAndEnvironmentExpansion(t *testing.T) {
	for _, text := range []string{
		"# coding: latin-1\nalpha\n", "API_KEY=synthetic", "export TOKEN=synthetic", "--index-url https://example.invalid", "foo --trusted-host evil.invalid", "foo @ https://example.invalid/a.whl", "./local.whl", "--editable ./x", "--hash=sha256:abc",
		"-r /etc/passwd", "-r https://example.invalid/file", "-r file:secret",
		"-r ../foo\\bar", "-r one two", "-r", "--requirement=", "-r --index-url=secret",
		"-r \"unterminated", "foo==${VERSION}", "foo\x00bar", "foo\rbar", string([]byte{0xff}),
	} {
		t.Run(text, func(t *testing.T) {
			if _, err := ParseRequirementsFile([]byte(text)); err == nil {
				t.Fatal("unsafe source accepted")
			}
		})
	}
	if _, err := ParseRequirementsFile([]byte(strings.Repeat("x", UVSyncSourceByteLimit+1))); err == nil {
		t.Fatal("oversize source accepted")
	}
	for _, text := range []string{"", "# comment\n", "-r ../shared.txt", "--requirement sub/child.txt", "-cconstraints.txt", "# coding: utf_8\nalpha\n", "--require-hashes", "foo==1 --hash=sha256:abc", `foo; python_version < "3.15"`} {
		if _, err := ParseRequirementsFile([]byte(text)); err != nil {
			t.Fatal("bounded source refused", text, err)
		}
	}
}
