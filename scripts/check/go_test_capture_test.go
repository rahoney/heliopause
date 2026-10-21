package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGoTestCaptureRetainsParallelActivityAndFirstFailure(t *testing.T) {
	now := time.Unix(100, 0)
	c := newGoTestCapture(func() time.Time { return now })
	const pkg = "github.com/rahoney/heliopause/internal/promotion"
	write := func(action, test, output string) {
		body, _ := json.Marshal(map[string]string{"Action": action, "Package": pkg, "Test": test, "Output": output})
		body = append(body, '\n')
		for _, part := range [][]byte{body[:7], body[7:]} {
			if n, err := c.Write(part); err != nil || n != len(part) {
				t.Fatal(n, err)
			}
		}
	}
	write("start", "", "")
	write("run", "TestSlow", "")
	write("pause", "TestSlow", "")
	write("run", "TestOther", "")
	write("output", "TestOther", "=== RUN   TestOther\n")
	write("output", "TestOther", "FIRST_FAILURE_DETAIL\n")
	write("fail", "TestOther", "")
	write("run", "TestOther/dynamic-secret-input", "")
	now = now.Add(7 * time.Second)
	text := c.String()
	for _, expected := range []string{"FIRST_FAILURE_DETAIL", "first_failed=" + pkg + "/TestOther", "test=TestSlow state=paused elapsed_ms=7000", "started_packages=1 finished_packages=0 active=2"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing %q in %s", expected, text)
		}
	}
	if strings.Contains(text, "dynamic-secret-input") || strings.Contains(text, "=== RUN") {
		t.Fatal("dynamic input/positive verbosity retained", text)
	}
	write("cont", "TestSlow", "")
	write("pass", "TestSlow", "")
	write("fail", "", "")
	text = c.String()
	if !strings.Contains(text, "finished_packages=1 active=0") || !strings.Contains(text, "first_failed="+pkg+"/TestOther") {
		t.Fatal("package completion lost first failure or active cleanup", text)
	}
}

func TestGoTestCaptureBoundsAndMalformedOutput(t *testing.T) {
	c := newGoTestCapture(time.Now)
	_, _ = c.Write([]byte("PLAIN_FIRST_ERROR\n"))
	_, _ = c.Write([]byte(strings.Repeat("x", maximumGoTestEventLine+1)))
	_, _ = c.Write([]byte("\n"))
	for i := 0; i < maximumGoTestActive+5; i++ {
		body, _ := json.Marshal(map[string]string{"Action": "run", "Package": "github.com/rahoney/heliopause/internal/promotion", "Test": fmt.Sprintf("TestCase%d", i)})
		_, _ = c.Write(append(body, '\n'))
	}
	text := c.String()
	if !strings.Contains(text, "PLAIN_FIRST_ERROR") || !strings.Contains(text, "unavailable=true truncated=true") || len(c.active) != maximumGoTestActive || len(c.line) > maximumGoTestEventLine || strings.Count(text, "active_package=") != 16 {
		t.Fatal("bounded diagnostic state disagrees", text)
	}
	for _, name := range []string{"/tmp/private", "github.com/rahoney/heliopause-foreign/pkg", "github.com/rahoney/heliopause/x\nsecret"} {
		if validGoTestPackage(name) {
			t.Fatal("unsafe package label accepted")
		}
	}
	if positiveGoTestProgress("=== RUN TestCase\nACTUAL_ERROR\n") {
		t.Fatal("multi-line first error hidden as positive progress")
	}
}

func TestGoTestCaptureConcurrentStderrAndStdout(t *testing.T) {
	c := newGoTestCapture(time.Now)
	var group sync.WaitGroup
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func(stream int) {
			defer group.Done()
			for n := 0; n < 100; n++ {
				if stream == 0 {
					_, _ = c.Write([]byte("{\"Action\":\"output\",\"Output\":\"STDOUT_MARKER\\n\"}\n"))
				} else {
					_, _ = (goTestStderr{c}).Write([]byte("STDERR_MARKER\n"))
				}
			}
		}(i)
	}
	group.Wait()
	text := c.String()
	if strings.Count(text, "STDOUT_MARKER") != 100 || strings.Count(text, "STDERR_MARKER") != 100 {
		t.Fatal("concurrent output dropped")
	}
}

func TestGoTestCaptureOversizeRetainsFirstError(t *testing.T) {
	c := newGoTestCapture(time.Now)
	first := []byte("{\"Action\":\"output\",\"Output\":\"FIRST_LARGE_FAILURE_")
	_, _ = c.Write(first)
	_, _ = c.Write([]byte(strings.Repeat("x", maximumGoTestEventLine)))
	_, _ = c.Write([]byte(strings.Repeat("y", outputLimit)))
	_, _ = c.Write([]byte("\"}\n"))
	text := c.String()
	if !strings.Contains(text, "FIRST_LARGE_FAILURE_") || !strings.Contains(text, "unavailable=true truncated=true") || len(c.line) > maximumGoTestEventLine || len(text) > outputLimit+1024 {
		t.Fatalf("oversize output lost the first error or escaped bounds: size=%d", len(text))
	}
}

func TestGoTestCaptureOfflinePrerequisiteUsesToolDiagnostics(t *testing.T) {
	c := newGoTestCapture(time.Now)
	_, _ = c.Write([]byte("{\"Action\":\"output\",\"Output\":\"passing fixture contains GOPROXY=off\\n\"}\n"))
	if c.offlineMissing {
		t.Fatal("test payload classified as missing prerequisite")
	}
	_, _ = (goTestStderr{c}).Write([]byte("module lookup disabled by GOPRO"))
	_, _ = (goTestStderr{c}).Write([]byte("XY=off\n"))
	if !c.offlineMissing {
		t.Fatal("split Go tool diagnostic lost")
	}
	c = newGoTestCapture(time.Now)
	_, _ = c.Write([]byte("{\"Action\":\"build-output\",\"Output\":\"module lookup disabled by GOPROXY=off\\n\"}\n"))
	if !c.offlineMissing {
		t.Fatal("Go build diagnostic lost")
	}
}

func TestGoTestCaptureRetainsOnlyBoundedSlowRootCompletions(t *testing.T) {
	now := time.Unix(100, 0)
	c := newGoTestCapture(func() time.Time { return now })
	write := func(action, name string) {
		body, _ := json.Marshal(map[string]string{"Action": action, "Package": "github.com/rahoney/heliopause/internal/promotion", "Test": name})
		_, _ = c.Write(append(body, '\n'))
	}
	for i := 0; i < 12; i++ {
		name := fmt.Sprintf("TestSlow%02d", i)
		write("run", name)
		now = now.Add(time.Duration(i) * time.Second)
		write("pass", name)
	}
	write("run", "TestTie")
	now = now.Add(11 * time.Second)
	write("fail", "TestTie")
	write("run", "TestShort")
	now = now.Add(500 * time.Millisecond)
	write("pass", "TestShort")
	text := c.String()
	if len(c.slow) != 8 || strings.Count(text, "slow_completed_package=") != 8 || c.slow[0].test != "TestSlow11" || c.slow[1].test != "TestTie" || !strings.Contains(text, "first_failed=") || strings.Contains(text, "test=TestShort") {
		t.Fatal("slow completion bounds/order/first failure disagrees", text)
	}
}
