package sandbox

import (
	"archive/tar"
	"bytes"
	"testing"
)

func uvRunTarData(t *testing.T, name string, body []byte, declared int64, complete bool) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	if err := writer.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0644, Uid: 1000, Gid: 1000, Size: declared}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
	if complete {
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return output.Bytes()
}

func TestUVRunCensusRetainsCurrentSelectedConsoleBytesWithoutExecutingThem(t *testing.T) {
	body := []byte("UNAPPROVED DATA; never executed")
	stream := uvRunTarData(t, "uv-run-cache/archive-v0/0123456789abcdef/bin/probe", body, int64(len(body)), true)
	facts, err := scanUVRunCensus(bytes.NewReader(stream), "probe")
	if err != nil || len(facts) != 1 || !bytes.Equal(facts[0].Body, body) {
		t.Fatal("selected-command DATA capture", err)
	}
	other, err := scanUVRunCensus(bytes.NewReader(stream), "other")
	if err != nil || len(other) != 1 || len(other[0].Body) != 0 || other[0].SHA256 != facts[0].SHA256 {
		t.Fatal("unselected DATA capture changed complete census", err)
	}
}

func TestUVRunCensusRejectsUnsafeCommandTraversalOversizeAndTruncatedInput(t *testing.T) {
	for _, command := range []string{"", "../probe", "-probe", "probe\x00other"} {
		if _, err := scanUVRunCensus(bytes.NewReader(nil), command); err == nil {
			t.Fatal("unsafe command accepted", command)
		}
	}
	if _, err := scanUVRunCensus(nil, "probe"); err == nil {
		t.Fatal("nil stream accepted")
	}
	for _, name := range []string{"../escape", "/absolute", "uv-run-cache/../escape"} {
		body := uvRunTarData(t, name, []byte("x"), 1, true)
		if _, err := scanUVRunCensus(bytes.NewReader(body), "probe"); err == nil {
			t.Fatal("unsafe path accepted", name)
		}
	}
	large := uvRunTarData(t, "uv-run-cache/archive-v0/0123456789abcdef/bin/probe", make([]byte, (64<<10)+1), (64<<10)+1, true)
	if _, err := scanUVRunCensus(bytes.NewReader(large), "probe"); err == nil {
		t.Fatal("selected control bound waived")
	}
	truncated := uvRunTarData(t, "uv-run-cache/archive-v0/0123456789abcdef/bin/probe", []byte("short"), 100, false)
	if _, err := scanUVRunCensus(bytes.NewReader(truncated), "probe"); err == nil {
		t.Fatal("truncated source accepted")
	}
}
