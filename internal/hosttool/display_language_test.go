package hosttool

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDisplayPlatformCommandIsFixedReadOnly(t *testing.T) {
	for _, test := range []struct {
		os     string
		wsl    bool
		source string
	}{{"darwin", false, "MACOS_UI"}, {"linux", true, "WINDOWS_UI"}, {"linux", false, ""}, {"windows", false, ""}} {
		plan, ok := displayLanguageCommand(test.os, test.wsl)
		if ok != (test.source != "") || plan.source != test.source {
			t.Fatal(test, plan)
		}
		if !ok {
			continue
		}
		if !filepath.IsAbs(plan.path) || !filepath.IsAbs(plan.directory) {
			t.Fatal(plan)
		}
		joined := strings.Join(plan.arguments, " ")
		if strings.Contains(joined, "Bypass") || strings.Contains(joined, "write ") {
			t.Fatal(plan)
		}
		if test.wsl && (!strings.Contains(joined, "-NoProfile") || !strings.Contains(joined, "CurrentUICulture.Name") || !strings.Contains(joined, "AutoLoadingPreference='None'")) {
			t.Fatal(plan)
		}
	}
}
func TestDisplayLanguageParsingBoundsAndTags(t *testing.T) {
	cases := []struct {
		body, source string
		valid        bool
		count        int
	}{
		{"ko-KR\r\n", "WINDOWS_UI", true, 1},
		{"(\n\"zh-Hant\", \"pt-BR\", ja,\n)", "MACOS_UI", true, 3},
		{"(en-US,ko-KR)", "MACOS_UI", true, 2},
		{"", "WINDOWS_UI", false, 0}, {"ko-KR\nen-US", "WINDOWS_UI", false, 0},
		{"ko-\x1b[31m", "WINDOWS_UI", false, 0}, {"ko;bad", "WINDOWS_UI", false, 0},
		{"(ko,,en)", "MACOS_UI", false, 0}, {"(\"ko)", "MACOS_UI", false, 0},
		{"()", "MACOS_UI", false, 0}, {"ko", "UNKNOWN", false, 0},
		{strings.Repeat("x", displayReadLimit+1), "WINDOWS_UI", false, 0},
		{"(" + strings.Repeat("en,", 33) + ")", "MACOS_UI", false, 0},
	}
	for _, c := range cases {
		tags, err := parseDisplayLanguages([]byte(c.body), c.source)
		if (err == nil) != c.valid || len(tags) != c.count {
			t.Fatalf("%q => %v %v", c.body, tags, err)
		}
	}
}
func TestDisplayCommandChild(t *testing.T) {
	if len(os.Args) == 0 {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case "haa-display-success":
		fmt.Print("ko-KR")
		os.Exit(0)
	case "haa-display-failure":
		fmt.Fprint(os.Stderr, "private backend diagnostic")
		os.Exit(3)
	case "haa-display-flood":
		_, _ = os.Stdout.Write(bytes.Repeat([]byte("a"), displayReadLimit*4))
		time.Sleep(time.Second)
		os.Exit(0)
	case "haa-display-stderr-flood":
		_, _ = os.Stderr.Write(bytes.Repeat([]byte("a"), displayReadLimit*4))
		time.Sleep(time.Second)
		os.Exit(0)
	case "haa-display-slow":
		time.Sleep(10 * time.Second)
		os.Exit(0)
	}
}
func TestDisplayCommandReapsAndBoundsOutput(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"success", "failure", "flood", "stderr-flood", "slow"} {
		t.Run(mode, func(t *testing.T) {
			budget := displayReadTimeout
			if mode == "slow" {
				budget = 100 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), budget)
			defer cancel()
			start := time.Now()
			body, err := runDisplayCommand(ctx, displayCommand{executable, t.TempDir(), "", []string{"-test.run=^TestDisplayCommandChild$", "--", "haa-display-" + mode}}, []string{"PATH=/usr/bin:/bin"})
			if mode == "success" {
				if err != nil || string(body) != "ko-KR" {
					t.Fatal(string(body), err)
				}
			} else {
				if err == nil || len(body) != 0 {
					t.Fatal("failed read exposed output", len(body), err)
				}
				if (mode == "flood" || mode == "stderr-flood") && errors.Is(err, context.DeadlineExceeded) {
					t.Fatal("overflow was masked by timeout", err)
				}
				if mode == "slow" && !errors.Is(err, context.DeadlineExceeded) {
					t.Fatal(err)
				}
				if strings.Contains(err.Error(), "private") {
					t.Fatal("raw backend error exposed")
				}
			}
			if time.Since(start) > displayReadTimeout+time.Second {
				t.Fatal("read exceeded bounded cancellation")
			}
		})
	}
}
func TestNativeDisplayLanguageRead(t *testing.T) {
	if runtime.GOOS != "darwin" && !displayWSL() {
		t.Skip("no native desktop/WSL preference on this host")
	}
	tags, source, err := PreferredDisplayLanguages(context.Background())
	if err != nil {
		// Headless hosts may have no global preference/disabled interop. This is a
		// qualified fallback, not a claim of a successful native preference read.
		t.Logf("native OS preference unavailable: source=%s fallback required", source)
		return
	}
	if len(tags) == 0 {
		t.Fatal("empty successful native read")
	}
	for _, tag := range tags {
		if !validDisplayTag(tag) {
			t.Fatal("invalid native tag")
		}
	}
	t.Logf("native OS preference source=%s tags=%v", source, tags)
}
