package sandbox

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
)

func TestBoundaryContainerCommandInstallsImmutableHelperBeforeServingExecs(t *testing.T) {
	command := boundaryContainerCommand()
	for _, required := range []string{
		"tmp=/haa-runtime/.haa-boundary.tmp",
		"chown 0:0 \"$tmp\"",
		"chmod 0555 \"$tmp\"",
		"mv \"$tmp\" " + boundaryHelperPath,
		"exec " + boundarySetprivPath + " " + boundaryDemotionArguments + " /bin/sleep infinity",
	} {
		if !strings.Contains(command, required) {
			t.Fatalf("initializer missing %q: %q", required, command)
		}
	}
	if strings.Contains(command, "docker cp") || strings.Contains(command, "--user "+boundaryBootstrapUser) {
		t.Fatalf("initializer has an invalid helper installation surface: %q", command)
	}
}

func isCPPIdentStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func isCPPIdentChar(b byte) bool {
	return isCPPIdentStart(b) || (b >= '0' && b <= '9')
}

type cppToken struct {
	typ   int
	val   string
	delim string
	body  string
}

const (
	cppTokIdent = 1
	cppTokPunct = 2
	cppTokRaw   = 3
)

func tokenizeCPPSource(source []byte) ([]cppToken, error) {
	var tokens []cppToken
	i := 0
	n := len(source)
	for i < n {
		b := source[i]
		if b == ' ' || b == '\t' || b == '\r' || b == '\n' {
			i++
			continue
		}
		// Line comment: skip through newline
		if b == '/' && i+1 < n && source[i+1] == '/' {
			i += 2
			for i < n && source[i] != '\n' {
				i++
			}
			continue
		}
		// Block comment: skip through */
		if b == '/' && i+1 < n && source[i+1] == '*' {
			i += 2
			idx := bytes.Index(source[i:], []byte("*/"))
			if idx == -1 {
				return nil, errors.New("unterminated block comment in C++ source")
			}
			i += idx + 2
			continue
		}
		// Character literal
		if b == '\'' {
			i++
			for i < n {
				if source[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				if source[i] == '\'' {
					i++
					break
				}
				i++
			}
			continue
		}
		// Raw string literal: R"delim(...)delim"
		if b == 'R' && i+1 < n && source[i+1] == '"' {
			start := i + 2
			paren := bytes.IndexByte(source[start:], '(')
			if paren >= 0 && paren <= 16 {
				delim := string(source[start : start+paren])
				if !strings.ContainsAny(delim, " \t\r\n\\)") {
					bodyStart := start + paren + 1
					closeSeq := []byte(")" + delim + "\"")
					endIdx := bytes.Index(source[bodyStart:], closeSeq)
					if endIdx >= 0 {
						body := string(source[bodyStart : bodyStart+endIdx])
						tokens = append(tokens, cppToken{
							typ:   cppTokRaw,
							val:   string(source[i : bodyStart+endIdx+len(closeSeq)]),
							delim: delim,
							body:  body,
						})
						i = bodyStart + endIdx + len(closeSeq)
						continue
					}
				}
			}
		}
		// Ordinary string literal: skip
		if b == '"' {
			i++
			for i < n {
				if source[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				if source[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}
		// Identifier or keyword
		if isCPPIdentStart(b) {
			start := i
			for i < n && isCPPIdentChar(source[i]) {
				i++
			}
			tokens = append(tokens, cppToken{typ: cppTokIdent, val: string(source[start:i])})
			continue
		}
		// Punctuation / operator
		tokens = append(tokens, cppToken{typ: cppTokPunct, val: string(b)})
		i++
	}
	return tokens, nil
}

func extractOCIBootstrapCommand(source []byte) (string, error) {
	tokens, err := tokenizeCPPSource(source)
	if err != nil {
		return "", fmt.Errorf("tokenize C++ source: %w", err)
	}

	var candidates []string
	for k := 0; k+7 < len(tokens); k++ {
		if tokens[k].typ == cppTokIdent && tokens[k].val == "constexpr" &&
			tokens[k+1].typ == cppTokIdent && tokens[k+1].val == "char" &&
			tokens[k+2].typ == cppTokIdent && tokens[k+2].val == "kOCIBootstrapCommand" &&
			tokens[k+3].typ == cppTokPunct && tokens[k+3].val == "[" &&
			tokens[k+4].typ == cppTokPunct && tokens[k+4].val == "]" &&
			tokens[k+5].typ == cppTokPunct && tokens[k+5].val == "=" &&
			tokens[k+6].typ == cppTokRaw && tokens[k+6].delim == "HAA" &&
			tokens[k+7].typ == cppTokPunct && tokens[k+7].val == ";" {
			candidates = append(candidates, tokens[k+6].body)
		}
	}

	if len(candidates) == 0 {
		return "", errors.New("no valid kOCIBootstrapCommand declaration found in observer source")
	}
	if len(candidates) > 1 {
		return "", fmt.Errorf("multiple (%d) kOCIBootstrapCommand declarations found in observer source", len(candidates))
	}
	return candidates[0], nil
}

func TestObserverOCIBootstrapCommandMatchesBoundaryContainerCommand(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate boundary helper test source")
	}
	source, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "..", "..", "tools", "gvisor-observer", "observer.cc"))
	if err != nil {
		t.Fatalf("read observer source: %v", err)
	}
	observerCommand, err := extractOCIBootstrapCommand(source)
	if err != nil {
		t.Fatalf("extract observer OCI bootstrap command: %v", err)
	}
	if observerCommand != boundaryContainerCommand() {
		t.Fatalf("observer OCI bootstrap command drifted from Go boundaryContainerCommand()")
	}
}

func TestExtractOCIBootstrapCommand_UniqueDeclaration(t *testing.T) {
	snippet := []byte(`
// Comment header
constexpr char kOCIBootstrapCommand[] = R"HAA(exact payload test)HAA";
`)
	extracted, err := extractOCIBootstrapCommand(snippet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extracted != "exact payload test" {
		t.Fatalf("extracted %q, want %q", extracted, "exact payload test")
	}
}

func TestExtractOCIBootstrapCommand_IgnoresLineComment(t *testing.T) {
	commentOnly := []byte(`// constexpr char kOCIBootstrapCommand[] = R"HAA(fake in line comment)HAA";`)
	if _, err := extractOCIBootstrapCommand(commentOnly); err == nil {
		t.Fatal("expected error when declaration is only in a line comment, got nil")
	}

	withComment := []byte(`
// constexpr char kOCIBootstrapCommand[] = R"HAA(fake in line comment)HAA";
constexpr char kOCIBootstrapCommand[] = R"HAA(real payload)HAA";
`)
	extracted, err := extractOCIBootstrapCommand(withComment)
	if err != nil {
		t.Fatalf("unexpected error with preceding line comment: %v", err)
	}
	if extracted != "real payload" {
		t.Fatalf("extracted %q, want %q", extracted, "real payload")
	}
}

func TestExtractOCIBootstrapCommand_IgnoresBlockComment(t *testing.T) {
	commentOnly := []byte(`/* constexpr char kOCIBootstrapCommand[] = R"HAA(fake in block comment)HAA"; */`)
	if _, err := extractOCIBootstrapCommand(commentOnly); err == nil {
		t.Fatal("expected error when declaration is only in a block comment, got nil")
	}

	withComment := []byte(`
/*
constexpr char kOCIBootstrapCommand[] = R"HAA(fake in block comment)HAA";
*/
constexpr char kOCIBootstrapCommand[] = R"HAA(real payload)HAA";
`)
	extracted, err := extractOCIBootstrapCommand(withComment)
	if err != nil {
		t.Fatalf("unexpected error with preceding block comment: %v", err)
	}
	if extracted != "real payload" {
		t.Fatalf("extracted %q, want %q", extracted, "real payload")
	}
}

func TestExtractOCIBootstrapCommand_IgnoresUnrelatedLiterals(t *testing.T) {
	inStringLit := []byte(`const char* s = "constexpr char kOCIBootstrapCommand[] = R\"HAA(fake in string)HAA\";";`)
	if _, err := extractOCIBootstrapCommand(inStringLit); err == nil {
		t.Fatal("expected error when declaration is inside a string literal, got nil")
	}

	inRawLit := []byte(`const char* s = R"UNRELATED(
constexpr char kOCIBootstrapCommand[] = R"HAA(fake in raw literal)HAA";
)UNRELATED";`)
	if _, err := extractOCIBootstrapCommand(inRawLit); err == nil {
		t.Fatal("expected error when declaration is inside an unrelated raw string literal, got nil")
	}

	similarIdent := []byte(`constexpr char kOCIBootstrapCommand_other[] = R"HAA(payload)HAA";`)
	if _, err := extractOCIBootstrapCommand(similarIdent); err == nil {
		t.Fatal("expected error for similar identifier name, got nil")
	}
}

func TestExtractOCIBootstrapCommand_MultipleCandidatesFails(t *testing.T) {
	duplicate := []byte(`
constexpr char kOCIBootstrapCommand[] = R"HAA(first)HAA";
constexpr char kOCIBootstrapCommand[] = R"HAA(second)HAA";
`)
	_, err := extractOCIBootstrapCommand(duplicate)
	if err == nil {
		t.Fatal("expected error on multiple candidate definitions, got nil")
	}
	if !strings.Contains(err.Error(), "multiple (2)") {
		t.Fatalf("expected error mentioning multiple candidates, got %v", err)
	}
}

func TestExtractOCIBootstrapCommand_MissingCandidateFails(t *testing.T) {
	missing := []byte(`int main() { return 0; }`)
	_, err := extractOCIBootstrapCommand(missing)
	if err == nil {
		t.Fatal("expected error on missing candidate definition, got nil")
	}
}

func TestObserverOCIBootstrapCommandDriftDetected(t *testing.T) {
	driftedSnippet := []byte(fmt.Sprintf("constexpr char kOCIBootstrapCommand[] = R\"HAA(%s-drift)HAA\";", boundaryContainerCommand()))
	extracted, err := extractOCIBootstrapCommand(driftedSnippet)
	if err != nil {
		t.Fatalf("unexpected extract error: %v", err)
	}
	if extracted == boundaryContainerCommand() {
		t.Fatal("expected payload byte difference to be detected as drift")
	}
}

func TestBoundaryExecUsesFixedRootBootstrapBeforeDemotingTarget(t *testing.T) {
	for _, test := range []struct {
		mode   string
		origin string
	}{
		{boundaryLaunchMode, boundaryOriginLaunchMode},
		{boundaryPythonHandoffMode, boundaryOriginPythonHandoffMode},
		{boundaryELFHandoffMode, boundaryOriginELFHandoffMode},
	} {
		arguments := boundaryExecArguments("0123456789abcdef", test.mode, "/bin/true")
		want := []string{"exec", "--user", boundaryBootstrapUser, "0123456789abcdef", boundaryHelperPath, test.origin, "/bin/true"}
		if !sameStrings(arguments, want) {
			t.Fatalf("boundary exec for %q = %#v, want %#v", test.mode, arguments, want)
		}
	}
	if !strings.Contains(boundaryHelper, "exec "+boundarySetprivPath+" "+boundaryDemotionArguments) {
		t.Fatalf("boundary helper does not irreversibly demote requested targets: %q", boundaryHelper)
	}
	for _, required := range []string{
		"exec " + boundaryHelperPath + " " + boundaryLaunchMode,
		"exec " + boundaryHelperPath + " " + boundaryPythonHandoffMode,
		"exec " + boundaryHelperPath + " " + boundaryELFHandoffMode,
	} {
		if !strings.Contains(boundaryHelper, required) {
			t.Fatalf("boundary helper does not self-exec canonical mode %q: %q", required, boundaryHelper)
		}
	}
	for _, required := range []string{
		`admission "${1-}"`,
		`token="$1"; shift; exec ` + boundaryHelperPath,
		`--launch|--handoff-python|--handoff-elf) shift; admission "${1-}"; shift; demote "$@"`,
	} {
		if !strings.Contains(boundaryHelper, required) {
			t.Fatalf("boundary helper does not validate then strip admission capability %q: %q", required, boundaryHelper)
		}
	}
	if strings.Contains(boundaryReadinessScript, "NoNewPrivs") {
		t.Fatalf("readiness relies on unsupported /proc NoNewPrivs: %q", boundaryReadinessScript)
	}
	if !strings.Contains(boundaryHelper, "  -c) shift; already_demoted; exec /bin/sh -c") ||
		strings.Contains(boundaryHelper, "--origin-c") {
		t.Fatalf("npm script-shell is not a direct in-container handoff: %q", boundaryHelper)
	}
	for _, required := range []string{
		"done < /proc/self/status",
		`Uid) [ "$#" -eq 4 ]`,
		`Gid) [ "$#" -eq 4 ]`,
		`Groups) [ "$#" -eq 0 ]`,
		`CapInh) [ "$#" -eq 1 ]`,
		`CapPrm) [ "$#" -eq 1 ]`,
		`CapEff) [ "$#" -eq 1 ]`,
		`CapBnd) [ "$#" -eq 1 ]`,
		`CapAmb) [ "$#" -eq 1 ]`,
		`1:1:1:1:1:1:1:1`,
	} {
		if !strings.Contains(boundaryHelper, required) {
			t.Fatalf("already-demoted npm handoff validation missing %q: %q", required, boundaryHelper)
		}
	}
	if strings.Contains(boundaryHelper, "-c) shift; demote") {
		t.Fatalf("already-demoted npm handoff still invokes privileged demotion: %q", boundaryHelper)
	}
}

func TestAwaitBoundaryHelperUsesOnlyWrappedUserExecAndFailsClosed(t *testing.T) {
	runner := &recordingRunner{errors: []error{errors.New("not ready")}}
	if err := awaitBoundaryHelper(context.Background(), runner, "0123456789abcdef"); err != nil {
		t.Fatalf("awaitBoundaryHelper() = %v", err)
	}
	if len(runner.calls) != 2 || !sameStrings(runner.calls[0].arguments, boundaryReadinessArguments("0123456789abcdef")) || !sameStrings(runner.calls[1].arguments, boundaryReadinessArguments("0123456789abcdef")) {
		t.Fatalf("helper readiness calls = %#v", runner.calls)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := awaitBoundaryHelper(ctx, &recordingRunner{errors: []error{errors.New("not ready")}}, "0123456789abcdef"); err == nil {
		t.Fatal("awaitBoundaryHelper() accepted unavailable helper")
	}
}
