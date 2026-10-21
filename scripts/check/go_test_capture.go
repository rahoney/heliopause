package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const maximumGoTestEventLine = 64 << 10
const maximumGoTestActive = 128

type goTestActivity struct {
	packageName, test string
	started           time.Time
	paused            bool
}

type goTestDuration struct {
	packageName, test, status string
	milliseconds              int64
}

// Diagnostic-only streaming state from the pinned Go tool. Keep source-level
// root function names, never dynamic subtest inputs or Output as authority.
type goTestCapture struct {
	mu                               sync.Mutex
	output                           boundedBuffer
	line                             []byte
	dropping, unavailable, truncated bool
	active                           map[string]goTestActivity
	started, finished                uint64
	firstFailed                      string
	offlineMissing                   bool
	stderrTail                       string
	slow                             []goTestDuration
	now                              func() time.Time
}

func newGoTestCapture(now func() time.Time) *goTestCapture {
	return &goTestCapture{output: boundedBuffer{limit: outputLimit}, active: make(map[string]goTestActivity), now: now}
}

func (c *goTestCapture) Write(data []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := len(data)
	for len(data) != 0 {
		end := bytes.IndexByte(data, '\n')
		part := data
		if end >= 0 {
			part = data[:end]
		}
		if c.dropping {
			_, _ = c.output.Write(part)
		} else {
			if len(part) > maximumGoTestEventLine-len(c.line) {
				_, _ = c.output.Write(c.line)
				_, _ = c.output.Write(part)
				c.line = nil
				c.dropping = true
				c.unavailable = true
				c.truncated = true
			} else {
				c.line = append(c.line, part...)
			}
		}
		if end < 0 {
			break
		}
		if !c.dropping {
			c.consume(c.line)
		} else {
			_, _ = c.output.Write([]byte{'\n'})
		}
		c.line = c.line[:0]
		c.dropping = false
		data = data[end+1:]
	}
	return n, nil
}

type goTestStderr struct{ capture *goTestCapture }

func (w goTestStderr) Write(data []byte) (int, error) {
	w.capture.mu.Lock()
	defer w.capture.mu.Unlock()
	// Only the Go tool's stderr/build diagnostics identify a missing offline
	// prerequisite. Test Output may contain the same text as ordinary fixture data.
	const marker = "GOPROXY=off"
	c := w.capture
	if !c.offlineMissing {
		if strings.Contains(c.stderrTail+string(data), marker) {
			c.offlineMissing = true
		}
		if len(data) >= len(marker)-1 {
			c.stderrTail = string(data[len(data)-(len(marker)-1):])
		} else {
			c.stderrTail += string(data)
			if len(c.stderrTail) > len(marker)-1 {
				c.stderrTail = c.stderrTail[len(c.stderrTail)-(len(marker)-1):]
			}
		}
	}
	return w.capture.output.Write(data)
}

func (c *goTestCapture) consume(body []byte) {
	var event struct{ Action, Package, Test, Output string }
	if json.Unmarshal(body, &event) != nil || event.Action == "" {
		c.unavailable = true
		_, _ = c.output.Write(body)
		_, _ = c.output.Write([]byte{'\n'})
		return
	}
	if event.Output != "" && !positiveGoTestProgress(event.Output) {
		_, _ = c.output.Write([]byte(event.Output))
	}
	if event.Action == "build-output" && strings.Contains(event.Output, "GOPROXY=off") {
		c.offlineMissing = true
	}
	if !validGoTestPackage(event.Package) || event.Test != "" && !validGoRootTest(event.Test) {
		return
	}
	key := event.Package + "\x00" + event.Test
	switch event.Action {
	case "start", "run":
		if _, exists := c.active[key]; exists {
			return
		}
		if len(c.active) >= maximumGoTestActive {
			c.truncated = true
			return
		}
		c.active[key] = goTestActivity{packageName: event.Package, test: event.Test, started: c.now()}
		if event.Test == "" {
			c.started++
		}
	case "pause", "cont":
		if activity, ok := c.active[key]; ok {
			activity.paused = event.Action == "pause"
			c.active[key] = activity
		}
	case "pass", "fail", "skip":
		if event.Test != "" {
			if activity, ok := c.active[key]; ok {
				elapsed := c.now().Sub(activity.started).Milliseconds()
				if elapsed >= 1000 {
					c.slow = append(c.slow, goTestDuration{event.Package, event.Test, event.Action, elapsed})
					sort.Slice(c.slow, func(i, j int) bool {
						a, b := c.slow[i], c.slow[j]
						if a.milliseconds != b.milliseconds {
							return a.milliseconds > b.milliseconds
						}
						return a.packageName+"/"+a.test < b.packageName+"/"+b.test
					})
					if len(c.slow) > 8 {
						c.slow = c.slow[:8]
					}
				}
			}
		}
		if event.Action == "fail" && c.firstFailed == "" {
			c.firstFailed = event.Package + "/" + event.Test
		}
		if event.Test == "" {
			if _, ok := c.active[key]; ok {
				c.finished++
			}
			for id, activity := range c.active {
				if activity.packageName == event.Package {
					delete(c.active, id)
				}
			}
		} else {
			delete(c.active, key)
		}
	}
}

func positiveGoTestProgress(output string) bool {
	line := strings.TrimSpace(output)
	if strings.ContainsAny(line, "\r\n") {
		return false
	}
	return line == "PASS" || strings.HasPrefix(line, "=== RUN ") || strings.HasPrefix(line, "=== PAUSE ") ||
		strings.HasPrefix(line, "=== CONT ") || strings.HasPrefix(line, "--- PASS:") || strings.HasPrefix(line, "--- SKIP:")
}

func validGoTestPackage(name string) bool {
	const module = "github.com/rahoney/heliopause"
	if len(name) == 0 || len(name) > 192 || name != module && !strings.HasPrefix(name, module+"/") {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("./_-", r)) {
			return false
		}
	}
	return true
}

func validGoRootTest(name string) bool {
	if len(name) > 128 || !(strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Fuzz") || strings.HasPrefix(name, "Example")) {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_') {
			return false
		}
	}
	return true
}

func (c *goTestCapture) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.line) != 0 {
		c.consume(c.line)
		c.line = nil
	}
	var result strings.Builder
	result.WriteString(c.output.String())
	fmt.Fprintf(&result, "\n[go-test progress started_packages=%d finished_packages=%d active=%d unavailable=%t truncated=%t]", c.started, c.finished, len(c.active), c.unavailable, c.truncated)
	if c.firstFailed != "" {
		fmt.Fprintf(&result, "\nfirst_failed=%s", c.firstFailed)
	}
	keys := make([]string, 0, len(c.active))
	for key := range c.active {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for i, key := range keys {
		if i == 16 {
			result.WriteString("\n[active entries truncated]")
			break
		}
		activity := c.active[key]
		state := "running"
		if activity.paused {
			state = "paused"
		}
		elapsed := c.now().Sub(activity.started).Milliseconds()
		if elapsed < 0 {
			elapsed = 0
		}
		fmt.Fprintf(&result, "\nactive_package=%s test=%s state=%s elapsed_ms=%d", activity.packageName, activity.test, state, elapsed)
	}
	for _, completed := range c.slow {
		fmt.Fprintf(&result, "\nslow_completed_package=%s test=%s status=%s elapsed_ms=%d", completed.packageName, completed.test, completed.status, completed.milliseconds)
	}
	return result.String()
}
