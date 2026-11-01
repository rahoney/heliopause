package pypi

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var requirementsBareAssignment = regexp.MustCompile(`(^|[^<>=!~])=([^=]|$)`)

var requirementsEncoding = regexp.MustCompile(`coding[:=]\s*([-A-Za-z0-9_.]+)`)

// RequirementsFile retains bounded logical lines and physical source locations. It is
// not a dependency selection, a PEP 440 solver, or installation approval.
type RequirementsFile struct {
	lines []RequirementsLine
}

// RequirementsLine preserves the physical source location after continuation
// and comment processing. Include names are relative to this file's directory;
// the filesystem owner must constrain and freeze the actual referenced file.
type RequirementsLine struct {
	Number     int    `json:"number"`
	Text       string `json:"text"`
	Include    string `json:"include,omitempty"`
	Constraint bool   `json:"constraint"`
}

func ParseRequirementsFile(body []byte) (RequirementsFile, error) {
	if len(body) > UVSyncSourceByteLimit || !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 {
		return RequirementsFile{}, errors.New("requirements source exceeds supported text bounds")
	}
	if !bytes.HasPrefix(body, []byte("\xef\xbb\xbf")) {
		physical := strings.SplitN(string(body), "\n", 3)
		for index, line := range physical {
			if index >= 2 || !strings.HasPrefix(line, "#") {
				continue
			}
			match := requirementsEncoding.FindStringSubmatch(line)
			if len(match) != 0 && strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(match[1]), "-", ""), "_", "") != "utf8" {
				return RequirementsFile{}, errors.New("requirements source encoding must be UTF-8")
			}
		}
	}
	result := RequirementsFile{}
	var pending strings.Builder
	first := 0
	for index, physical := range append(strings.Split(strings.TrimPrefix(string(body), "\ufeff"), "\n"), "") {
		physical = strings.TrimSuffix(physical, "\r")
		if strings.ContainsRune(physical, '\r') {
			return RequirementsFile{}, fmt.Errorf("requirements line %d has an unsupported line ending", index+1)
		}
		if first == 0 {
			first = index + 1
		}
		// Like pinned pip, comment-only lines do not continue onto the next
		// physical line. Comments are removed after continuation processing.
		continued := strings.HasSuffix(physical, "\\") && !strings.HasPrefix(strings.TrimSpace(physical), "#")
		if continued {
			physical = strings.Trim(physical, "\\")
		}
		if strings.HasPrefix(strings.TrimSpace(physical), "#") {
			pending.WriteByte(' ')
		}
		pending.WriteString(physical)
		if continued {
			continue
		}
		text := pending.String()
		var previous rune
		for at, r := range text {
			if r == '#' && (at == 0 || unicode.IsSpace(previous)) {
				text = text[:at]
				break
			}
			previous = r
		}
		text = strings.TrimSpace(text)
		if text != "" {
			if strings.Contains(text, "${") {
				return RequirementsFile{}, fmt.Errorf("requirements line %d cannot expand environment variables", first)
			}
			line := RequirementsLine{Number: first, Text: text}
			for _, option := range []struct {
				name       string
				constraint bool
			}{
				{"--requirement", false}, {"--constraint", true}, {"-r", false}, {"-c", true},
			} {
				if !strings.HasPrefix(text, option.name) {
					continue
				}
				argument := strings.TrimPrefix(text, option.name)
				if len(option.name) > 2 && argument != "" && argument[0] != '=' && !unicode.IsSpace(rune(argument[0])) {
					continue
				}
				argument = strings.TrimSpace(strings.TrimPrefix(argument, "="))
				if len(argument) >= 2 && (argument[0] == '\'' || argument[0] == '"') && argument[len(argument)-1] == argument[0] {
					argument = argument[1 : len(argument)-1]
				} else if strings.IndexFunc(argument, unicode.IsSpace) >= 0 {
					return RequirementsFile{}, fmt.Errorf("requirements include at line %d needs one relative filename", first)
				}
				if argument == "" || len(argument) > 4096 || path.IsAbs(argument) || strings.ContainsAny(argument, "\\:\"'\t\r\n") || strings.HasPrefix(argument, "-") {
					return RequirementsFile{}, fmt.Errorf("requirements include at line %d is unsupported", first)
				}
				line.Include, line.Constraint = argument, option.constraint
				break
			}
			if line.Include == "" && text != "--require-hashes" {
				var requirement []string
				options := false
				for _, token := range strings.Fields(text) {
					if strings.HasPrefix(token, "-") {
						options = true
						if token != "--hash" && !strings.HasPrefix(token, "--hash=") {
							return RequirementsFile{}, fmt.Errorf("requirements option at line %d is unsupported", first)
						}
					}
					if !options {
						requirement = append(requirement, token)
					}
				}
				base, _, _ := strings.Cut(strings.Join(requirement, " "), ";")
				if base == "" || requirementsBareAssignment.MatchString(base) || strings.ContainsAny(base, "@:/\\") || !(base[0] >= 'a' && base[0] <= 'z' || base[0] >= 'A' && base[0] <= 'Z' || base[0] >= '0' && base[0] <= '9') {
					return RequirementsFile{}, fmt.Errorf("requirements source at line %d is unsupported", first)
				}
			}
			if len(result.lines) >= maxPyPIReportEntries {
				return RequirementsFile{}, errors.New("requirements source exceeds logical line bounds")
			}
			result.lines = append(result.lines, line)
		}
		pending.Reset()
		first = 0
	}
	return result, nil
}

func (r RequirementsFile) Lines() []RequirementsLine {
	return append([]RequirementsLine(nil), r.lines...)
}
