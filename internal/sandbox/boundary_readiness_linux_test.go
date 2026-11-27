package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBoundaryReadinessChecksBothKernelStatusesWithTwoReaders(t *testing.T) {
	const good = "Name:\tsleep\nUid:\t1000\t1000\t1000\t1000\nGid:\t1000\t1000\t1000\t1000\nGroups:\t\nCapInh:\t0000000000000000\nCapPrm:\t0000000000000000\nCapEff:\t0000000000000000\nCapBnd:\t0000000000000000\nCapAmb:\t0000000000000000\n"
	root := t.TempDir()
	parent := filepath.Join(root, "init-status")
	self := filepath.Join(root, "self-status")
	emptyPATH := filepath.Join(root, "no-executables")
	if err := os.Mkdir(emptyPATH, 0700); err != nil {
		t.Fatal(err)
	}
	reads := filepath.Join(root, "reader-count")
	// Permit only the existing grep reader and count actual process invocations.
	wrapper := "#!/bin/sh\nprintf 'read\\n' >> \"$HAA_READINESS_READS\"\nexec /usr/bin/grep \"$@\"\n"
	if err := os.WriteFile(filepath.Join(emptyPATH, "grep"), []byte(wrapper), 0700); err != nil {
		t.Fatal(err)
	}
	// Only the two fixed kernel paths become fixture files. The evaluated shell
	// and credential/capability checks are the production readiness command.
	script := strings.ReplaceAll(boundaryReadinessScript, "/proc/1/status", parent)
	script = strings.ReplaceAll(script, "/proc/self/status", self)
	run := func(t *testing.T, parentBody, selfBody string, want bool) {
		t.Helper()
		if err := os.WriteFile(reads, nil, 0600); err != nil {
			t.Fatal(err)
		}
		for path, body := range map[string]string{parent: parentBody, self: selfBody} {
			if err := os.WriteFile(path, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "/bin/sh", "-ceu", script)
		cmd.Env = []string{"PATH=" + emptyPATH, "LC_ALL=C", "HAA_READINESS_READS=" + reads}
		out, err := cmd.CombinedOutput()
		countBody, readErr := os.ReadFile(reads)
		if readErr != nil {
			t.Fatal(readErr)
		}
		count := strings.Count(string(countBody), "read\n")
		if count > 2 || (want && count != 2) {
			t.Fatalf("readiness readers=%d want at most2/exact2 on success", count)
		}
		if (err == nil) != want {
			t.Fatalf("readiness success=%t want=%t error=%v output=%q", err == nil, want, err, out)
		}
	}
	run(t, good, good, true)
	for _, target := range []string{"init", "self"} {
		t.Run(target, func(t *testing.T) {
			check := func(bad string) {
				t.Helper()
				if target == "init" {
					run(t, bad, good, false)
				} else {
					run(t, good, bad, false)
				}
			}
			for _, field := range []string{"Uid", "Gid"} {
				for slot := 0; slot < 4; slot++ {
					values := []string{"1000", "1000", "1000", "1000"}
					values[slot] = "0"
					check(strings.Replace(good, field+":\t1000\t1000\t1000\t1000", field+":\t"+strings.Join(values, "\t"), 1))
				}
			}
			check(strings.Replace(good, "Groups:\t\n", "Groups:\t0 1000\n", 1))
			for _, field := range []string{"CapInh", "CapPrm", "CapEff", "CapBnd", "CapAmb"} {
				check(strings.Replace(good, field+":\t0000000000000000", field+":\t0000000000000001", 1))
			}
			for _, field := range []string{"Uid", "Gid", "Groups", "CapInh", "CapPrm", "CapEff", "CapBnd", "CapAmb"} {
				var lines []string
				for _, line := range strings.Split(good, "\n") {
					if !strings.HasPrefix(line, field+":") {
						lines = append(lines, line)
					}
				}
				check(strings.Join(lines, "\n"))
			}
			check("")
		})
	}
	for _, missing := range []string{parent, self} {
		for _, path := range []string{parent, self} {
			if err := os.WriteFile(path, []byte(good), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Remove(missing); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		cmd := exec.CommandContext(ctx, "/bin/sh", "-ceu", script)
		cmd.Env = []string{"PATH=" + emptyPATH, "HAA_READINESS_READS=" + reads}
		err := cmd.Run()
		cancel()
		if err == nil {
			t.Fatal("missing kernel status admitted readiness")
		}
	}
}
