package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
)

func TestExecutionOutputIsExplicitBoundedInertData(t *testing.T) {
	for _, payload := range [][]byte{
		[]byte("private-output"), []byte("\x1b]52;c;secret\x07\x1b[2J\r\nQuarantine: COMPLETED"),
		append([]byte("한국어\u202e\u2066\u0085\u2028"), 0, 255, 0x9b), bytes.Repeat([]byte{27}, 1<<20),
	} {
		hidden := executionOutput(payload, []byte("private-stderr"), false)
		body, err := json.Marshal(hidden)
		if err != nil || strings.Contains(string(body), "private-") || hidden.Stdout != "" || hidden.Stderr != "" || hidden.Encoding != "OMITTED" || hidden.StdoutBytes != len(payload) {
			t.Fatal("private output disclosed without explicit selection", err)
		}
		shown := executionOutput(payload, []byte("stderr\n"), true)
		decoded, err := strconv.Unquote(shown.Stdout)
		if err != nil || !bytes.Equal([]byte(decoded), payload) {
			t.Fatal("display encoding changed original bytes", err)
		}
		for _, r := range shown.Stdout {
			if r < 32 || r > 126 {
				t.Fatal("terminal control/non-ASCII escaped display missing")
			}
		}
		if len(shown.Stdout) > 6*len(payload)+2 {
			t.Fatal("escaped display exceeds input-derived bound")
		}
		report := executionReport{Schema: "display-test-only", ProgramOutput: shown}
		for _, machine := range []bool{false, true} {
			var out bytes.Buffer
			if err := writeExecutionReport(&out, report, machine); err != nil {
				t.Fatal(err)
			}
			if bytes.ContainsAny(out.Bytes(), "\x1b\x07\r") || strings.Contains(out.String(), "\nQuarantine: COMPLETED") {
				t.Fatal("program stream controlled HAA display")
			}
			if machine && !json.Valid(out.Bytes()) {
				t.Fatal("program stream corrupted JSON")
			}
		}
	}
}

type executionFailWriter struct{}

func (executionFailWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestExecutionReportPreservesWriterFailure(t *testing.T) {
	for _, machine := range []bool{true, false} {
		if err := writeExecutionReport(executionFailWriter{}, executionReport{}, machine); !errors.Is(err, io.ErrClosedPipe) {
			t.Fatal("writer cause lost", err)
		}
		if err := writeExecutionReport(nil, executionReport{}, machine); err == nil {
			t.Fatal("nil writer accepted")
		}
	}
}
