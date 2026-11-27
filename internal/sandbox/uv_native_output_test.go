package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func uvOutputFiles() map[string][]byte {
	return map[string][]byte{
		"pylock.toml":                       []byte("lock-version='1.0'\n"),
		"site/probe-1.0.dist-info/METADATA": []byte("Name: probe\nVersion: 1.0\n"),
		"site/probe-1.0.dist-info/WHEEL":    []byte("Wheel-Version: 1.0\nTag: py3-none-any\n"),
		"site/probe/__init__.py":            []byte("raise RuntimeError('must never execute')\n"),
	}
}

func TestUVNativeOutputReadsControlsWithoutExecutingCode(t *testing.T) {
	data, err := scanUVNativeOutput(bytes.NewReader(testClosureTar(t, uvOutputFiles(), 1000)))
	if err != nil || len(data.lock) == 0 || len(data.controls) != 1 || data.controls[0].DistInfo != "probe-1.0.dist-info" {
		t.Fatalf("bounded native output rejected: %v", err)
	}
	for name, mutate := range map[string]func(map[string][]byte){
		"missing lock":      func(f map[string][]byte) { delete(f, "pylock.toml") },
		"missing metadata":  func(f map[string][]byte) { delete(f, "site/probe-1.0.dist-info/METADATA") },
		"missing wheel":     func(f map[string][]byte) { delete(f, "site/probe-1.0.dist-info/WHEEL") },
		"empty control":     func(f map[string][]byte) { f["site/probe-1.0.dist-info/WHEEL"] = nil },
		"foreign root file": func(f map[string][]byte) { f["surprise"] = []byte("x") },
		"absolute path":     func(f map[string][]byte) { f["/site/escape"] = []byte("x") },
		"traversal":         func(f map[string][]byte) { f["site/../escape"] = []byte("x") },
		"backslash":         func(f map[string][]byte) { f["site/a\\b"] = []byte("x") },
		"oversize lock":     func(f map[string][]byte) { f["pylock.toml"] = bytes.Repeat([]byte("x"), uvNativeLockLimit+1) },
		"oversize metadata": func(f map[string][]byte) {
			f["site/probe-1.0.dist-info/METADATA"] = bytes.Repeat([]byte("x"), (1<<20)+1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			files := uvOutputFiles()
			mutate(files)
			got, err := scanUVNativeOutput(bytes.NewReader(testClosureTar(t, files, 1000)))
			if err == nil || len(got.lock) != 0 || len(got.controls) != 0 {
				t.Fatal("invalid output retained native selection data")
			}
		})
	}
}

func TestUVNativeOutputRejectsTarStructureAndTruncation(t *testing.T) {
	valid := testClosureTar(t, uvOutputFiles(), 1000)
	for name, body := range map[string][]byte{
		"wrong owner":      testClosureTar(t, uvOutputFiles(), 0),
		"truncated header": valid[:100],
		"truncated file":   valid[:513],
		"trailing archive": append(append([]byte(nil), valid...), valid...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := scanUVNativeOutput(bytes.NewReader(body)); err == nil {
				t.Fatal("invalid tar accepted")
			}
		})
	}
	for _, header := range []tar.Header{
		{Name: "site/link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd", Mode: 0o644, Uid: 1000, Gid: 1000},
		{Name: "site/hard", Typeflag: tar.TypeLink, Linkname: "pylock.toml", Mode: 0o644, Uid: 1000, Gid: 1000},
		{Name: "site/world", Typeflag: tar.TypeReg, Mode: 0o666, Uid: 1000, Gid: 1000},
		{Name: "site/setuid", Typeflag: tar.TypeReg, Mode: 0o4644, Uid: 1000, Gid: 1000},
		{Name: "site/huge", Typeflag: tar.TypeReg, Size: 1 << 30, Mode: 0o644, Uid: 1000, Gid: 1000},
	} {
		var buffer bytes.Buffer
		writer := tar.NewWriter(&buffer)
		if err := writer.WriteHeader(&header); err != nil {
			t.Fatal(err)
		}
		// A declared excessive size must fail before reading its missing body.
		if _, err := scanUVNativeOutput(bytes.NewReader(buffer.Bytes())); err == nil {
			t.Fatalf("unsupported header accepted: %s", header.Name)
		}
	}
	var duplicate bytes.Buffer
	writer := tar.NewWriter(&duplicate)
	for range 2 {
		if err := writer.WriteHeader(&tar.Header{Name: "site", Typeflag: tar.TypeDir, Mode: 0o700, Uid: 1000, Gid: 1000}); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := scanUVNativeOutput(&duplicate); err == nil {
		t.Fatal("duplicate tar path accepted")
	}
}

type uvOutputRunner struct {
	body   []byte
	err    error
	call   []string
	cancel context.CancelFunc
}

func (r *uvOutputRunner) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("buffered command must not be used")
}

func (r *uvOutputRunner) RunOutput(_ context.Context, output io.Writer, binary string, args ...string) error {
	r.call = append([]string{binary}, args...)
	_, err := output.Write(r.body)
	if r.cancel != nil {
		r.cancel()
	}
	return errors.Join(err, r.err)
}

func TestUVNativeOutputRequiresProducerCompletion(t *testing.T) {
	body := testClosureTar(t, uvOutputFiles(), 1000)
	id := strings.Repeat("a", 64)
	for _, outcome := range []string{"complete", "command failure", "cancel", "scanner refusal"} {
		t.Run(outcome, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			runner := &uvOutputRunner{body: body}
			cause := errors.New("producer failed after output")
			switch outcome {
			case "command failure":
				runner.err = cause
			case "cancel":
				runner.cancel = cancel
			case "scanner refusal":
				runner.body = bytes.Repeat([]byte("x"), 8192)
			}
			data, err := readUVNativeOutput(ctx, runner, id)
			if outcome == "complete" {
				if err != nil || len(data.controls) != 1 {
					t.Fatal("completed producer rejected", err)
				}
			} else if err == nil || len(data.lock) != 0 || len(data.controls) != 0 {
				t.Fatal("failed producer retained output")
			}
			if outcome == "command failure" && !errors.Is(err, cause) {
				t.Fatal("producer cause was lost")
			}
			if strings.Join(runner.call, " ") != "docker cp "+id+":"+pythonSitePath+"/. -" {
				t.Fatal("output path changed", runner.call)
			}
		})
	}
}

func TestUVNativeOutputDiscardsOnlyExactEmptyAuxiliaryLock(t *testing.T) {
	for _, test := range []struct {
		name     string
		mode     int64
		body     string
		kind     byte
		uid      int
		accepted bool
	}{
		{"site/.lock", 0o666, "", tar.TypeReg, 1000, true},
		{"site/.lock", 0o666, "x", tar.TypeReg, 1000, false},
		{"site/.lock.other", 0o666, "", tar.TypeReg, 1000, false},
		{"site/other", 0o666, "", tar.TypeReg, 1000, false},
		{"site/.lock", 0o777, "", tar.TypeReg, 1000, false},
		{"site/.lock", 0o666, "", tar.TypeSymlink, 1000, false},
		{"site/.lock", 0o666, "", tar.TypeReg, 0, false},
	} {
		var buffer bytes.Buffer
		writer := tar.NewWriter(&buffer)
		reader := tar.NewReader(bytes.NewReader(testClosureTar(t, uvOutputFiles(), 1000)))
		for {
			header, err := reader.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := writer.WriteHeader(header); err != nil {
				t.Fatal(err)
			}
			if _, err := io.Copy(writer, reader); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.WriteHeader(&tar.Header{Name: test.name, Mode: test.mode, Size: int64(len(test.body)), Typeflag: test.kind, Linkname: "ignored", Uid: test.uid, Gid: test.uid}); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(writer, test.body); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		data, err := scanUVNativeOutput(&buffer)
		if (err == nil) != test.accepted {
			t.Fatalf("auxiliary lock acceptance differs: %+v: %v", test, err)
		}
		if test.accepted && len(data.controls) != 1 {
			t.Fatal("auxiliary lock changed control selection")
		}
	}
}

func TestUVNativeLockReaderRequiresOneBoundedOwnedFileAndProducerCompletion(t *testing.T) {
	valid := testClosureTar(t, map[string][]byte{"pylock.toml": []byte("lock-version='1.0'\n")}, 1000)
	for name, body := range map[string][]byte{
		"complete":         valid,
		"wrong owner":      testClosureTar(t, map[string][]byte{"pylock.toml": []byte("lock")}, 0),
		"wrong path":       testClosureTar(t, map[string][]byte{"site/pylock.toml": []byte("lock")}, 1000),
		"extra entry":      testClosureTar(t, map[string][]byte{"pylock.toml": []byte("lock"), "extra": []byte("x")}, 1000),
		"empty lock":       testClosureTar(t, map[string][]byte{"pylock.toml": nil}, 1000),
		"oversize lock":    testClosureTar(t, map[string][]byte{"pylock.toml": bytes.Repeat([]byte("x"), uvNativeLockLimit+1)}, 1000),
		"truncated lock":   valid[:513],
		"trailing archive": append(append([]byte(nil), valid...), valid...),
	} {
		t.Run(name, func(t *testing.T) {
			runner := &uvOutputRunner{body: body}
			lock, err := readUVNativeLock(context.Background(), runner, strings.Repeat("a", 64))
			if (err == nil) != (name == "complete") {
				t.Fatal("lock transport acceptance differs", err)
			}
			if err != nil && len(lock) != 0 {
				t.Fatal("failed lock transport retained bytes")
			}
			if strings.Join(runner.call, " ") != "docker cp "+strings.Repeat("a", 64)+":"+pythonSitePath+"/pylock.toml -" {
				t.Fatal("lock transport used wrong path")
			}
		})
	}
	cause := errors.New("producer failed after lock bytes")
	runner := &uvOutputRunner{body: valid, err: cause}
	if lock, err := readUVNativeLock(context.Background(), runner, strings.Repeat("a", 64)); !errors.Is(err, cause) || len(lock) != 0 {
		t.Fatal("lock bytes replaced producer completion", err)
	}
	for _, header := range []tar.Header{
		{Name: "pylock.toml", Mode: 0o666, Uid: 1000, Gid: 1000, Size: 1, Typeflag: tar.TypeReg},
		{Name: "pylock.toml", Mode: 0o644, Uid: 1000, Gid: 1000, Typeflag: tar.TypeSymlink, Linkname: "elsewhere"},
	} {
		var buffer bytes.Buffer
		writer := tar.NewWriter(&buffer)
		if err := writer.WriteHeader(&header); err != nil {
			t.Fatal(err)
		}
		if _, err := scanUVNativeLock(bytes.NewReader(buffer.Bytes())); err == nil {
			t.Fatal("unsafe lock header accepted")
		}
	}
}
