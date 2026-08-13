package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var markdownLink = regexp.MustCompile(`!?\[[^\]]*\]\(([^)]*)\)`)
var markdownReference = regexp.MustCompile(`^ {0,3}\[[^\]]+\]:\s*(.+)$`)

func checkMarkdownTree(root string) error {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return &checkFailure{class: executionFailure, step: "documentation inventory", detail: "resolve source tree", cause: err}
	}
	root = resolvedRoot

	var markdownFiles []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			if entry.Type()&os.ModeSymlink != 0 {
				return &checkFailure{class: findingFailure, step: "documentation", detail: fmt.Sprintf("Markdown file is a symbolic link: %s", displayPath(root, path))}
			}
			markdownFiles = append(markdownFiles, path)
		}
		return nil
	})
	if err != nil {
		if failure, ok := err.(*checkFailure); ok {
			return failure
		}
		return &checkFailure{class: executionFailure, step: "documentation inventory", cause: err}
	}
	sort.Strings(markdownFiles)

	var findings []string
	for _, path := range markdownFiles {
		fileFindings, err := checkMarkdownFile(root, path)
		if err != nil {
			return err
		}
		findings = append(findings, fileFindings...)
	}
	if len(findings) != 0 {
		return &checkFailure{class: findingFailure, step: "documentation", detail: strings.Join(findings, "\n")}
	}
	return nil
}

func checkMarkdownFile(root, path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, &checkFailure{class: executionFailure, step: "documentation", detail: displayPath(root, path), cause: err}
	}
	defer file.Close()

	var findings []string
	var fence byte
	var fenceLength int
	var fenceLine int
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), outputLimit)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Text()
		marker, length, closing := fenceMarker(line, fence, fenceLength)
		if fence == 0 && marker != 0 {
			fence, fenceLength, fenceLine = marker, length, lineNumber
			continue
		}
		if fence != 0 {
			if closing {
				fence, fenceLength, fenceLine = 0, 0, 0
			}
			continue
		}

		matches := markdownLink.FindAllStringSubmatch(line, -1)
		if reference := markdownReference.FindStringSubmatch(line); reference != nil {
			matches = append(matches, []string{reference[0], reference[1]})
		}
		for _, match := range matches {
			target := markdownDestination(match[1])
			if target == "" || strings.HasPrefix(target, "#") {
				continue
			}
			finding := validateLocalLink(root, filepath.Dir(path), target)
			if finding != "" {
				findings = append(findings, fmt.Sprintf("%s:%d: %s", displayPath(root, path), lineNumber, finding))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, &checkFailure{class: executionFailure, step: "documentation", detail: displayPath(root, path), cause: err}
	}
	if fence != 0 {
		findings = append(findings, fmt.Sprintf("%s:%d: unclosed fenced code block", displayPath(root, path), fenceLine))
	}
	return findings, nil
}

func fenceMarker(line string, open byte, openLength int) (byte, int, bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || len(trimmed) < 3 {
		return 0, 0, false
	}
	marker := trimmed[0]
	if marker != '`' && marker != '~' {
		return 0, 0, false
	}
	length := 0
	for length < len(trimmed) && trimmed[length] == marker {
		length++
	}
	if length < 3 {
		return 0, 0, false
	}
	if open == 0 {
		return marker, length, false
	}
	if marker == open && length >= openLength && strings.TrimSpace(trimmed[length:]) == "" {
		return marker, length, true
	}
	return marker, length, false
}

func markdownDestination(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "<") {
		if end := strings.IndexByte(trimmed, '>'); end > 0 {
			return trimmed[1:end]
		}
	}
	if fields := strings.Fields(trimmed); len(fields) != 0 {
		return fields[0]
	}
	return ""
}

func validateLocalLink(root, sourceDirectory, target string) string {
	if looksLikeWindowsPath(target) {
		return fmt.Sprintf("absolute local link is not allowed: %q", target)
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return fmt.Sprintf("malformed link target %q", target)
	}
	if parsed.Scheme != "" {
		if strings.EqualFold(parsed.Scheme, "file") {
			return fmt.Sprintf("absolute local link is not allowed: %q", target)
		}
		return ""
	}
	if filepath.IsAbs(parsed.Path) || strings.HasPrefix(parsed.Path, "\\\\") {
		return fmt.Sprintf("absolute local link is not allowed: %q", target)
	}
	if parsed.Path == "" {
		return ""
	}
	decodedPath, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return fmt.Sprintf("malformed escaped link target %q", target)
	}
	resolved := filepath.Clean(filepath.Join(sourceDirectory, filepath.FromSlash(decodedPath)))
	if !isWithinRoot(root, resolved) {
		return fmt.Sprintf("local link leaves the source tree: %q", target)
	}
	_, err = os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("local link target does not exist: %q", target)
		}
		return fmt.Sprintf("cannot inspect local link target %q: %v", target, err)
	}
	realTarget, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		return fmt.Sprintf("cannot resolve local link target %q: %v", target, err)
	}
	if !isWithinRoot(root, realTarget) {
		return fmt.Sprintf("local link resolves outside the source tree: %q", target)
	}
	return ""
}

func looksLikeWindowsPath(target string) bool {
	return len(target) >= 3 && ((target[0] >= 'A' && target[0] <= 'Z') || (target[0] >= 'a' && target[0] <= 'z')) && target[1] == ':' && (target[2] == '\\' || target[2] == '/')
}

func displayPath(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relative)
}
