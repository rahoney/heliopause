package pypi

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	defaultWheelMaxCompressed   = 64 << 20
	defaultWheelMaxUncompressed = 256 << 20
	defaultWheelMaxFiles        = 10000
	defaultWheelMaxMetadata     = 1 << 20
)

// WheelLimits bounds archive parsing before any file content is trusted.
type WheelLimits struct {
	MaxCompressed, MaxUncompressed, MaxFiles, MaxMetadata int64
}

func DefaultWheelLimits() WheelLimits {
	return WheelLimits{defaultWheelMaxCompressed, defaultWheelMaxUncompressed, defaultWheelMaxFiles, defaultWheelMaxMetadata}
}

// WheelTarget is the locked interpreter/ABI/platform compatibility tuple.
type WheelTarget struct{ Python, ABI, Platform string }

// WheelFile is normalized RECORD evidence for one regular installed file.
type WheelFile struct {
	Path   string
	Size   int64
	SHA256 string
}

// WheelInspection contains only bounded, normalized static evidence.
type WheelInspection struct {
	Project, Version, Filename        string
	PythonTags, ABITags, PlatformTags []string
	Tags                              []string
	WheelVersion                      string
	DeclaredSHA256, ObservedSHA256    string
	Files                             []WheelFile
	RequiresPython                    string
	RequiresDist, ImportNames         []string
	EntryPoints, Scripts              []string
	NativeExtensions                  []string
	License, LicenseFile              string
}

// InspectWheel validates one selected wheel without extracting or executing it.
func InspectWheel(reader io.ReaderAt, size int64, filename, declaredSHA256 string, target WheelTarget, limits WheelLimits) (WheelInspection, error) {
	return inspectWheel(reader, size, filename, declaredSHA256, target, limits, false)
}

// InspectWheelForSource permits PyTorch's canonical PEP 440 local build
// suffixes while retaining the ordinary PyPI parser's stricter contract.
func InspectWheelForSource(reader io.ReaderAt, size int64, filename, declaredSHA256 string, target WheelTarget, limits WheelLimits, source domain.SourceID) (WheelInspection, error) {
	return inspectWheel(reader, size, filename, declaredSHA256, target, limits, IsPyTorchSource(source))
}

func inspectWheel(reader io.ReaderAt, size int64, filename, declaredSHA256 string, target WheelTarget, limits WheelLimits, allowLocalVersion bool) (WheelInspection, error) {
	if reader == nil || size <= 0 || filename == "" || !validSHA256(declaredSHA256) || limits.MaxCompressed <= 0 || limits.MaxUncompressed <= 0 || limits.MaxFiles <= 0 || limits.MaxMetadata <= 0 {
		return WheelInspection{}, errors.New("wheel intake input is invalid")
	}
	project, version, pyTags, abiTags, platformTags, err := parseWheelFilenameWithLocal(filename, allowLocalVersion)
	if err != nil || !wheelTagsCompatible(pyTags, abiTags, platformTags, target) {
		return WheelInspection{}, errors.New("wheel filename or target tags are invalid")
	}
	if size > limits.MaxCompressed {
		return WheelInspection{}, errors.New("wheel compressed size exceeds bound")
	}
	archive, err := zip.NewReader(reader, size)
	if err != nil || int64(len(archive.File)) > limits.MaxFiles {
		return WheelInspection{}, errors.New("wheel ZIP structure is invalid")
	}
	seen := make(map[string]*zip.File, len(archive.File))
	var uncompressed int64
	var metadataFiles = map[string][]byte{}
	metadataCount := map[string]int{}
	var regularFiles []string
	hash := sha256.New()
	if err := hashReaderAt(reader, size, hash); err != nil {
		return WheelInspection{}, errors.New("wheel digest could not be computed")
	}
	observed := hex.EncodeToString(hash.Sum(nil))
	if observed != declaredSHA256 {
		return WheelInspection{}, errors.New("wheel declared and observed SHA-256 differ")
	}
	for _, entry := range archive.File {
		name, ok := wheelPath(entry.Name)
		if !ok || seen[name] != nil {
			return WheelInspection{}, errors.New("wheel contains unsafe or duplicate path")
		}
		seen[name] = entry
		if entry.FileInfo().Mode()&os.ModeSymlink != 0 || entry.FileInfo().Mode().IsDir() && entry.UncompressedSize64 != 0 {
			return WheelInspection{}, errors.New("wheel contains unsupported file type")
		}
		if entry.UncompressedSize64 > uint64(limits.MaxUncompressed) || uncompressed > limits.MaxUncompressed-int64(entry.UncompressedSize64) {
			return WheelInspection{}, errors.New("wheel uncompressed size exceeds bound")
		}
		uncompressed += int64(entry.UncompressedSize64)
		if entry.FileInfo().Mode().IsRegular() {
			regularFiles = append(regularFiles, name)
		}
		if strings.HasSuffix(name, ".dist-info/METADATA") || strings.HasSuffix(name, ".dist-info/WHEEL") || strings.HasSuffix(name, ".dist-info/RECORD") {
			metadataCount[path.Base(name)]++
			if entry.UncompressedSize64 > uint64(limits.MaxMetadata) {
				return WheelInspection{}, errors.New("wheel metadata exceeds bound")
			}
			body, readErr := readZipEntry(entry, limits.MaxMetadata)
			if readErr != nil {
				return WheelInspection{}, errors.New("wheel metadata is unreadable")
			}
			metadataFiles[path.Base(name)] = body
		}
	}
	if metadataCount["METADATA"] != 1 || metadataCount["WHEEL"] != 1 || metadataCount["RECORD"] != 1 {
		return WheelInspection{}, errors.New("wheel requires one metadata set")
	}
	info, err := parseWheelMetadata(project, version, filename, metadataFiles["METADATA"], metadataFiles["WHEEL"], metadataFiles["RECORD"], regularFiles, seen, limits)
	if err != nil {
		return WheelInspection{}, err
	}
	info.PythonTags, info.ABITags, info.PlatformTags = pyTags, abiTags, platformTags
	if !wheelMetadataTagsMatch(info.Tags, pyTags, abiTags, platformTags) {
		return WheelInspection{}, errors.New("wheel WHEEL tags do not match filename")
	}
	info.ObservedSHA256 = observed
	info.DeclaredSHA256 = declaredSHA256
	for _, name := range regularFiles {
		if strings.HasPrefix(name, info.distInfo()+"/") {
			continue
		}
		if strings.Contains(name, ".data/scripts/") {
			info.Scripts = append(info.Scripts, name)
		}
		if strings.HasSuffix(name, ".so") || strings.Contains(name, ".so.") || strings.HasSuffix(name, ".pyd") {
			info.NativeExtensions = append(info.NativeExtensions, name)
		}
	}
	return info, nil
}

func parseWheelMetadata(project, version, filename string, metadata, wheel, record []byte, regular []string, entries map[string]*zip.File, limits WheelLimits) (WheelInspection, error) {
	if len(metadata) == 0 || len(wheel) == 0 || len(record) == 0 {
		return WheelInspection{}, errors.New("wheel requires METADATA, WHEEL and RECORD")
	}
	meta := headerValues(metadata, limits.MaxMetadata)
	wheelHeaders := headerValues(wheel, limits.MaxMetadata)
	if meta["name"] != project || meta["version"] != version || wheelHeaders["wheel-version"] == "" || !supportedWheelVersion(wheelHeaders["wheel-version"]) {
		return WheelInspection{}, errors.New("wheel embedded metadata does not match filename")
	}
	distInfo := ""
	for name := range entries {
		if strings.HasSuffix(name, ".dist-info/METADATA") {
			distInfo = strings.TrimSuffix(name, "/METADATA")
			break
		}
	}
	if distInfo != project+"-"+version+".dist-info" {
		return WheelInspection{}, errors.New("wheel dist-info directory is invalid")
	}
	if _, ok := entries[distInfo+"/WHEEL"]; !ok {
		return WheelInspection{}, errors.New("wheel dist-info files are incomplete")
	}
	for _, name := range []string{distInfo + "/METADATA", distInfo + "/WHEEL", distInfo + "/RECORD"} {
		entry, ok := entries[name]
		if !ok {
			return WheelInspection{}, errors.New("wheel dist-info files are incomplete")
		}
		body, readErr := readZipEntry(entry, limits.MaxMetadata)
		if readErr != nil {
			return WheelInspection{}, errors.New("wheel metadata is unreadable")
		}
		var expected []byte
		switch path.Base(name) {
		case "METADATA":
			expected = metadata
		case "WHEEL":
			expected = wheel
		case "RECORD":
			expected = record
		}
		if !bytes.Equal(body, expected) {
			return WheelInspection{}, errors.New("wheel metadata set is inconsistent")
		}
	}
	files, err := validateRecord(record, regular, entries, limits.MaxMetadata)
	if err != nil {
		return WheelInspection{}, err
	}
	imports := splitHeaders(meta["import-name"])
	if len(imports) == 0 {
		imports = importNamesFromWheelFiles(files, distInfo)
	}
	return WheelInspection{Project: project, Version: version, Filename: filename, WheelVersion: wheelHeaders["wheel-version"], Tags: splitHeaders(wheelHeaders["tag"]), Files: files, RequiresPython: meta["requires-python"], RequiresDist: splitHeaders(meta["requires-dist"]), ImportNames: imports, EntryPoints: splitHeaders(meta["entry-points"]), License: meta["license"], LicenseFile: meta["license-file"]}, nil
}

// importNamesFromWheelFiles is a bounded static fallback for widely deployed
// wheels that predate the optional Import-Name metadata. Only a valid Python
// identifier observed as a top-level module or package is returned; the
// dynamic backend never receives package-manager or caller-supplied names.
func importNamesFromWheelFiles(files []WheelFile, distInfo string) []string {
	seen := map[string]struct{}{}
	for _, file := range files {
		name := file.Path
		if strings.HasPrefix(name, distInfo+"/") || strings.Contains(name, ".data/") {
			continue
		}
		parts := strings.Split(name, "/")
		if len(parts) == 1 && strings.HasSuffix(parts[0], ".py") {
			module := strings.TrimSuffix(parts[0], ".py")
			if module != "__init__" && validPythonImportComponent(module) {
				seen[module] = struct{}{}
			}
			continue
		}
		if len(parts) > 1 && strings.HasSuffix(name, ".py") && validPythonImportComponent(parts[0]) {
			seen[parts[0]] = struct{}{}
		}
	}
	imports := make([]string, 0, len(seen))
	for name := range seen {
		imports = append(imports, name)
	}
	sort.Strings(imports)
	return imports
}

func validPythonImportComponent(value string) bool {
	if value == "" {
		return false
	}
	for index, character := range value {
		if character == '_' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || index > 0 && character >= '0' && character <= '9' {
			continue
		}
		return false
	}
	return true
}

func validateRecord(body []byte, regular []string, entries map[string]*zip.File, limit int64) ([]WheelFile, error) {
	r := csv.NewReader(bytes.NewReader(body))
	r.FieldsPerRecord = 3
	recorded := map[string]WheelFile{}
	for {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || row[0] == "" {
			return nil, errors.New("wheel RECORD is invalid")
		}
		name, ok := wheelPath(row[0])
		if !ok || recorded[name].Path != "" {
			return nil, errors.New("wheel RECORD path is invalid")
		}
		if row[1] == "" && row[2] == "" {
			recorded[name] = WheelFile{Path: name}
			continue
		}
		if !strings.HasPrefix(row[1], "sha256=") {
			return nil, errors.New("wheel RECORD digest is invalid")
		}
		digest, err := base64.RawURLEncoding.DecodeString(row[1][len("sha256="):])
		if err != nil || len(digest) != sha256.Size {
			return nil, errors.New("wheel RECORD digest is invalid")
		}
		n, err := strconv.ParseInt(row[2], 10, 64)
		if err != nil || n < 0 || int64(len(recorded)) > limit {
			return nil, errors.New("wheel RECORD size is invalid")
		}
		recorded[name] = WheelFile{Path: name, Size: n, SHA256: hex.EncodeToString(digest)}
	}
	for _, name := range regular {
		if name == "" {
			continue
		}
		file, ok := recorded[name]
		if !ok || file.Path == "" {
			return nil, errors.New("wheel contains unrecorded file")
		}
		if strings.HasSuffix(name, ".dist-info/RECORD") && file.Size == 0 && file.SHA256 == "" {
			continue
		}
		if name != path.Base(name) && strings.HasSuffix(name, "/RECORD") {
			continue
		}
		entry := entries[name]
		body, err := readZipEntry(entry, limit)
		if err != nil || int64(len(body)) != file.Size {
			return nil, errors.New("wheel RECORD size mismatch")
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != file.SHA256 && name != path.Base(name) {
			return nil, errors.New("wheel RECORD digest mismatch")
		}
	}
	if len(recorded) != len(regular) {
		return nil, errors.New("wheel RECORD contains unknown file")
	}
	files := make([]WheelFile, 0, len(recorded))
	for _, file := range recorded {
		if file.Path != "" {
			files = append(files, file)
		}
	}
	return files, nil
}

func (i WheelInspection) distInfo() string {
	return i.Project + "-" + strings.ReplaceAll(i.Version, ".", ".") + ".dist-info"
}
func supportedWheelVersion(v string) bool {
	return v == "1.0" || v == "1.1" || v == "1.2" || v == "2.0"
}
func validSHA256(v string) bool {
	if len(v) != 64 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil
}
func wheelPath(name string) (string, bool) {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return "", false
	}
	clean := path.Clean(name)
	if clean != name || clean == "." || strings.HasPrefix(clean, "../") || clean == ".." || strings.HasSuffix(clean, "/") {
		return "", false
	}
	return clean, true
}
func readZipEntry(file *zip.File, limit int64) ([]byte, error) {
	if file == nil || file.UncompressedSize64 > uint64(limit) {
		return nil, errors.New("entry exceeds bound")
	}
	r, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, limit+1))
}
func headerValues(body []byte, limit int64) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(string(body), "\n") {
		if line == "" {
			continue
		}
		at := strings.IndexByte(line, ':')
		if at <= 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:at]))
		value := strings.TrimSpace(line[at+1:])
		if key == "requires-dist" || key == "import-name" || key == "entry-points" || key == "tag" {
			out[key] += value + "\n"
		} else {
			out[key] = value
		}
	}
	return out
}
func splitHeaders(value string) []string {
	var out []string
	for _, v := range strings.Split(value, "\n") {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}
func parseWheelFilename(filename string) (string, string, []string, []string, []string, error) {
	return parseWheelFilenameWithLocal(filename, false)
}

func parseWheelFilenameWithLocal(filename string, allowLocalVersion bool) (string, string, []string, []string, []string, error) {
	if !strings.HasSuffix(filename, ".whl") {
		return "", "", nil, nil, nil, errors.New("not a wheel")
	}
	parts := strings.Split(strings.TrimSuffix(filename, ".whl"), "-")
	if len(parts) < 5 {
		return "", "", nil, nil, nil, errors.New("wheel filename is invalid")
	}
	py, abi, platform := strings.Split(parts[len(parts)-3], "."), strings.Split(parts[len(parts)-2], "."), strings.Split(parts[len(parts)-1], ".")
	for split := 1; split < len(parts)-3; split++ {
		project, err := NormalizeProjectName(strings.Join(parts[:split], "-"))
		version, versionErr := normalizeVersionForProfile(parts[split], allowLocalVersion)
		if err == nil && versionErr == nil {
			return project, version, py, abi, platform, nil
		}
	}
	return "", "", nil, nil, nil, errors.New("wheel name/version is invalid")
}

// ParseWheelFilename returns the normalized identity and declared compatibility
// tags of one wheel filename without inspecting archive bytes.
func ParseWheelFilename(filename string) (string, string, []string, []string, []string, error) {
	return parseWheelFilename(filename)
}

func ParseWheelFilenameForSource(filename string, source domain.SourceID) (string, string, []string, []string, []string, error) {
	return parseWheelFilenameWithLocal(filename, IsPyTorchSource(source))
}
func wheelTagsCompatible(py, abi, platform []string, target WheelTarget) bool {
	if target.Python == "" || target.ABI == "" || target.Platform == "" {
		return false
	}
	has := func(values []string, want string, kind string) bool {
		for _, value := range values {
			if value == "any" || value == want || kind == "python" && value == "py3" && strings.HasPrefix(want, "cp3") || kind == "abi" && value == "none" || kind == "platform" && manylinuxTagCompatible(value, want) {
				return true
			}
		}
		return false
	}
	return has(py, target.Python, "python") && has(abi, target.ABI, "abi") && has(platform, target.Platform, "platform")
}

var manylinuxPlatformPattern = regexp.MustCompile(`^manylinux_([0-9]+)_([0-9]+)_(.+)$`)

// manylinuxTagCompatible permits a target with equal-or-newer glibc on the
// same architecture to run a wheel built for an older manylinux baseline.
func manylinuxTagCompatible(wheel, target string) bool {
	wheelMatch := manylinuxPlatformPattern.FindStringSubmatch(wheel)
	targetMatch := manylinuxPlatformPattern.FindStringSubmatch(target)
	if len(wheelMatch) != 4 || len(targetMatch) != 4 || wheelMatch[3] != targetMatch[3] {
		return false
	}
	wheelMajor, err := strconv.Atoi(wheelMatch[1])
	if err != nil {
		return false
	}
	wheelMinor, err := strconv.Atoi(wheelMatch[2])
	if err != nil {
		return false
	}
	targetMajor, err := strconv.Atoi(targetMatch[1])
	if err != nil {
		return false
	}
	targetMinor, err := strconv.Atoi(targetMatch[2])
	if err != nil {
		return false
	}
	return targetMajor > wheelMajor || targetMajor == wheelMajor && targetMinor >= wheelMinor
}
func wheelMetadataTagsMatch(tags, py, abi, platform []string) bool {
	if len(tags) == 0 {
		return false
	}
	want := strings.Join(py, ".") + "-" + strings.Join(abi, ".") + "-" + strings.Join(platform, ".")
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}
func hashReaderAt(reader io.ReaderAt, size int64, hash io.Writer) error {
	buf := make([]byte, 64<<10)
	for offset := int64(0); offset < size; {
		n, err := reader.ReadAt(buf[:minInt64(int64(len(buf)), size-offset)], offset)
		if n > 0 {
			if _, werr := hash.Write(buf[:n]); werr != nil {
				return werr
			}
			offset += int64(n)
		}
		if errors.Is(err, io.EOF) && offset == size {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}
func minInt64(a, b int64) int {
	if a < b {
		return int(a)
	}
	return int(b)
}
