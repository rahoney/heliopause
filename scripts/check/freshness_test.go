package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func validVersionSupportRecord() supportRecordFile {
	components := []string{
		"Go", "Docker", "containerd", "gVisor", "Bazel", "Node", "npm", "Python", "pip",
		"PyTorch CPU", "PyTorch cu126", "PyTorch cu130", "PyTorch cu132", "Staticcheck", "gosec", "govulncheck", "Gitleaks",
	}
	records := make([]supportRecord, 0, len(components))
	for _, component := range components {
		records = append(records, supportRecord{
			Component: component, PinnedVersion: "1.0.0", SupportRole: "test", OfficialSource: "https://example.test/source",
			CheckedAt: "2026-09-17", Latest: "1.0.0", Status: "SUPPORTED", Decision: "KEEP", Reason: "test record",
		})
	}
	return supportRecordFile{SchemaVersion: 1, Records: records}
}

func TestValidateVersionSupport(t *testing.T) {
	now := time.Date(2026, time.September, 17, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		mutate      func(*supportRecordFile)
		strict      bool
		wantError   bool
		wantWarning bool
	}{
		{name: "valid record"},
		{name: "missing field", mutate: func(file *supportRecordFile) { file.Records[0].PinnedVersion = "" }, wantError: true},
		{name: "malformed date", mutate: func(file *supportRecordFile) { file.Records[0].CheckedAt = "2026-99-99" }, wantError: true},
		{name: "stale warning", mutate: func(file *supportRecordFile) { file.Records[0].CheckedAt = "2026-06-18" }, wantWarning: true},
		{name: "strict stale failure", mutate: func(file *supportRecordFile) { file.Records[0].CheckedAt = "2026-06-18" }, strict: true, wantError: true, wantWarning: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file := validVersionSupportRecord()
			if test.mutate != nil {
				test.mutate(&file)
			}
			var output bytes.Buffer
			err := validateVersionSupport(file, now, test.strict, &output)
			if (err != nil) != test.wantError {
				t.Fatalf("validateVersionSupport() error = %v, want error %t", err, test.wantError)
			}
			if strings.Contains(output.String(), "WARNING:") != test.wantWarning {
				t.Fatalf("warning output = %q, want warning %t", output.String(), test.wantWarning)
			}
		})
	}
}
