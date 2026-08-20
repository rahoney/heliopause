package pypi

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
)

const (
	defaultSdistMaxCompressed   = 64 << 20
	defaultSdistMaxUncompressed = 256 << 20
	defaultSdistMaxFiles        = 10000
	defaultSdistMaxMetadata     = 1 << 20
)

var pep517EntryPoint = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*(?::[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)?$`)

// SdistLimits bounds static source-distribution inspection before any backend
// code is eligible for a later isolated build.
type SdistLimits struct {
	MaxCompressed, MaxUncompressed, MaxFiles, MaxMetadata int64
}

func DefaultSdistLimits() SdistLimits {
	return SdistLimits{defaultSdistMaxCompressed, defaultSdistMaxUncompressed, defaultSdistMaxFiles, defaultSdistMaxMetadata}
}

// SdistInspection is bounded static evidence used to decide whether a PEP 517
// build may be attempted in a separate gVisor session. It contains no source
// bytes, Host paths or backend output.
type SdistInspection struct {
	Project, Version, Filename                                    string
	DeclaredSHA256, ObservedSHA256                                string
	BuildBackend, BuildConfigSHA256                               string
	BuildRequirements, BuildRequirementSpecs, RuntimeRequirements []string
}

// InspectSdist validates a public PEP 517 .tar.gz source distribution without
// extracting it to the Host filesystem or invoking its build backend.
func InspectSdist(reader io.Reader, filename, declaredSHA256 string, limits SdistLimits) (SdistInspection, error) {
	if reader == nil || filename == "" || !validSHA256(declaredSHA256) || limits.MaxCompressed <= 0 || limits.MaxUncompressed <= 0 || limits.MaxFiles <= 0 || limits.MaxMetadata <= 0 {
		return SdistInspection{}, errors.New("sdist intake input is invalid")
	}
	project, version, err := parseSdistFilename(filename)
	if err != nil {
		return SdistInspection{}, err
	}
	compressed, readErr := io.ReadAll(io.LimitReader(reader, limits.MaxCompressed+1))
	if readErr != nil || int64(len(compressed)) > limits.MaxCompressed {
		return SdistInspection{}, errors.New("sdist compressed size exceeds bound")
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return SdistInspection{}, errors.New("sdist gzip structure is invalid")
	}
	defer gzipReader.Close()
	archive := tar.NewReader(io.LimitReader(gzipReader, limits.MaxUncompressed+1))
	seen := map[string]bool{}
	var root string
	var files, uncompressed int64
	metadata := map[string][]byte{}
	for {
		header, nextErr := archive.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return SdistInspection{}, errors.New("sdist archive is invalid")
		}
		if header.Format != tar.FormatPAX {
			return SdistInspection{}, errors.New("sdist must use POSIX.1-2001 PAX tar format")
		}
		files++
		if files > limits.MaxFiles || header.Size < 0 || header.Size > limits.MaxUncompressed || uncompressed > limits.MaxUncompressed-header.Size {
			return SdistInspection{}, errors.New("sdist archive exceeds bounds")
		}
		name, ok := sdistPath(header.Name)
		if !ok || seen[name] {
			return SdistInspection{}, errors.New("sdist path is unsafe or duplicate")
		}
		seen[name] = true
		parts := strings.Split(name, "/")
		if len(parts) < 2 {
			return SdistInspection{}, errors.New("sdist must have one top-level directory")
		}
		if root == "" {
			root = parts[0]
		}
		if parts[0] != root {
			return SdistInspection{}, errors.New("sdist has multiple top-level directories")
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if header.Size != 0 {
				return SdistInspection{}, errors.New("sdist directory is invalid")
			}
		case tar.TypeReg:
			uncompressed += header.Size
			if name == root+"/PKG-INFO" || name == root+"/pyproject.toml" {
				if header.Size > limits.MaxMetadata {
					return SdistInspection{}, errors.New("sdist metadata exceeds bound")
				}
				body, readErr := io.ReadAll(io.LimitReader(archive, limits.MaxMetadata+1))
				if readErr != nil || int64(len(body)) != header.Size {
					return SdistInspection{}, errors.New("sdist metadata is unreadable")
				}
				metadata[path.Base(name)] = body
			}
		default:
			return SdistInspection{}, errors.New("sdist contains unsupported file type")
		}
	}
	if root != project+"-"+version || len(metadata["PKG-INFO"]) == 0 || len(metadata["pyproject.toml"]) == 0 {
		return SdistInspection{}, errors.New("sdist metadata does not match filename")
	}
	sum := sha256.Sum256(compressed)
	observed := hex.EncodeToString(sum[:])
	if observed != declaredSHA256 {
		return SdistInspection{}, errors.New("sdist declared and observed SHA-256 differ")
	}
	pkgInfo := headerValues(metadata["PKG-INFO"], limits.MaxMetadata)
	if pkgInfo["name"] != project || pkgInfo["version"] != version {
		return SdistInspection{}, errors.New("sdist PKG-INFO does not match filename")
	}
	backend, requirements, err := parseBuildSystem(metadata["pyproject.toml"])
	if err != nil {
		return SdistInspection{}, err
	}
	configDigest := sha256.Sum256(metadata["pyproject.toml"])
	return SdistInspection{Project: project, Version: version, Filename: filename, DeclaredSHA256: declaredSHA256, ObservedSHA256: observed, BuildBackend: backend, BuildConfigSHA256: hex.EncodeToString(configDigest[:]), BuildRequirements: requirementNames(requirements), BuildRequirementSpecs: requirements, RuntimeRequirements: splitHeaders(pkgInfo["requires-dist"])}, nil
}

func parseSdistFilename(filename string) (string, string, error) {
	if !strings.HasSuffix(filename, ".tar.gz") {
		return "", "", errors.New("sdist filename is invalid")
	}
	parts := strings.Split(strings.TrimSuffix(filename, ".tar.gz"), "-")
	for split := 1; split < len(parts); split++ {
		project, projectErr := NormalizeProjectName(strings.Join(parts[:split], "-"))
		version, versionErr := NormalizeVersion(strings.Join(parts[split:], "-"))
		if projectErr == nil && versionErr == nil {
			return project, version, nil
		}
	}
	return "", "", errors.New("sdist filename is invalid")
}

func sdistPath(value string) (string, bool) {
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") {
		return "", false
	}
	clean := path.Clean(value)
	if clean != value || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return clean, true
}

// parseBuildSystem accepts only a deliberately small static PEP 517 surface.
// Dynamic TOML features, backend-path and non-string requirement entries are
// rejected so a later build cannot silently expand its execution authority.
func parseBuildSystem(body []byte) (string, []string, error) {
	section := ""
	backend := ""
	var requires []string
	for _, raw := range strings.Split(string(body), "\n") {
		line := strings.TrimSpace(strings.SplitN(raw, "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		if section != "build-system" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return "", nil, errors.New("pyproject build-system is invalid")
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch key {
		case "build-backend":
			if backend != "" || len(value) < 3 || value[0] != '"' || value[len(value)-1] != '"' {
				return "", nil, errors.New("pyproject build backend is invalid")
			}
			backend = value[1 : len(value)-1]
		case "requires":
			if requires != nil {
				return "", nil, errors.New("pyproject build requirements are invalid")
			}
			parsed, err := parseTOMLStringArray(value)
			if err != nil || len(parsed) == 0 {
				return "", nil, errors.New("pyproject build requirements are invalid")
			}
			requires = parsed
		case "backend-path":
			return "", nil, errors.New("in-tree build backend is unsupported")
		default:
			return "", nil, errors.New("pyproject build-system key is unsupported")
		}
	}
	if !pep517EntryPoint.MatchString(backend) || len(requires) == 0 {
		return "", nil, errors.New("pyproject PEP 517 build system is incomplete")
	}
	normalized := make([]string, 0, len(requires))
	seen := map[string]bool{}
	for _, requirement := range requires {
		name, err := parseStaticRequirement(requirement)
		if err != nil || seen[name] {
			return "", nil, errors.New("pyproject build requirement is unsupported")
		}
		seen[name] = true
		normalized = append(normalized, requirement)
	}
	sort.Strings(normalized)
	return backend, normalized, nil
}

func requirementNames(requirements []string) []string {
	names := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		name, err := parseStaticRequirement(requirement)
		if err != nil {
			return nil
		}
		names = append(names, name)
	}
	return names
}

func parseTOMLStringArray(value string) ([]string, error) {
	if len(value) < 2 || value[0] != '[' || value[len(value)-1] != ']' {
		return nil, errors.New("not an array")
	}
	inside := strings.TrimSpace(value[1 : len(value)-1])
	if inside == "" {
		return nil, nil
	}
	parts := strings.Split(inside, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 2 || part[0] != '"' || part[len(part)-1] != '"' || strings.Contains(part[1:len(part)-1], "\\") {
			return nil, errors.New("not a string array")
		}
		result = append(result, part[1:len(part)-1])
	}
	return result, nil
}

func parseStaticRequirement(value string) (string, error) {
	if value == "" || value != strings.TrimSpace(value) || strings.ContainsAny(value, ";@[]") {
		return "", errors.New("requirement is unsupported")
	}
	matches := requirementNamePrefix.FindStringSubmatch(value)
	if matches == nil {
		return "", errors.New("requirement is unsupported")
	}
	return NormalizeProjectName(matches[1])
}
