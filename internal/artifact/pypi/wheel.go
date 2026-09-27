package pypi

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"debug/elf"
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
	"unicode"
	"unicode/utf8"

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

// WheelValidationStage is a bounded parser-stage classification exposed only
// so the PyPI inspection adapter can produce qualification diagnostics. It is
// never used to alter parser acceptance.
type WheelValidationStage string

const (
	WheelValidationFilename         WheelValidationStage = "FILENAME"
	WheelValidationCompatibility    WheelValidationStage = "COMPATIBILITY"
	WheelValidationZIP              WheelValidationStage = "ZIP"
	WheelValidationDigest           WheelValidationStage = "DIGEST"
	WheelValidationMetadataIdentity WheelValidationStage = "METADATA_IDENTITY"
	WheelValidationDistInfoIdentity WheelValidationStage = "DIST_INFO_IDENTITY"
	WheelValidationWheelTag         WheelValidationStage = "WHEEL_TAG"
	WheelValidationRecord           WheelValidationStage = "RECORD"
	WheelValidationFileType         WheelValidationStage = "FILE_TYPE"
	WheelValidationOther            WheelValidationStage = "OTHER"
	WheelValidationCompressed       WheelValidationStage = "PER_ARTIFACT_COMPRESSED"
	WheelValidationUncompressed     WheelValidationStage = "PER_ARTIFACT_UNCOMPRESSED"
	WheelValidationFileCount        WheelValidationStage = "FILE_COUNT"
	WheelValidationMetadata         WheelValidationStage = "METADATA"
	WheelValidationMetadataInvalid  WheelValidationStage = "METADATA_INVALID"
)

type wheelValidationError struct{ stage WheelValidationStage }

func (e wheelValidationError) Error() string { return "wheel validation failed" }

func wheelValidation(stage WheelValidationStage) error { return wheelValidationError{stage: stage} }

// WheelValidationStageOf returns only the bounded stage, never parser error
// text.
func WheelValidationStageOf(err error) (WheelValidationStage, bool) {
	validation, ok := err.(wheelValidationError)
	return validation.stage, ok
}

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
	// NoImportSurface is true only for a wheel proven to contain no dynamic Python
	// surface (such as a metadata-only wheel or native/data-only wheel).
	NoImportSurface      bool
	EntryPoints, Scripts []string
	NativeExtensions     []string
	License, LicenseFile string
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
		return WheelInspection{}, wheelValidation(WheelValidationOther)
	}
	project, version, pyTags, abiTags, platformTags, err := parseWheelFilenameWithLocal(filename, allowLocalVersion)
	if err != nil {
		return WheelInspection{}, wheelValidation(WheelValidationFilename)
	}
	if !wheelTagsCompatible(pyTags, abiTags, platformTags, target) {
		return WheelInspection{}, wheelValidation(WheelValidationCompatibility)
	}
	if size > limits.MaxCompressed {
		return WheelInspection{}, wheelValidation(WheelValidationCompressed)
	}
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return WheelInspection{}, wheelValidation(WheelValidationZIP)
	}
	if int64(len(archive.File)) > limits.MaxFiles {
		return WheelInspection{}, wheelValidation(WheelValidationFileCount)
	}
	seen := make(map[string]*zip.File, len(archive.File))
	var uncompressed int64
	var metadataFiles = map[string][]byte{}
	var regularFiles []string
	primaryDistInfo := wheelDistInfo(project, version)
	hash := sha256.New()
	if err := hashReaderAt(reader, size, hash); err != nil {
		return WheelInspection{}, wheelValidation(WheelValidationDigest)
	}
	observed := hex.EncodeToString(hash.Sum(nil))
	if observed != declaredSHA256 {
		return WheelInspection{}, wheelValidation(WheelValidationDigest)
	}
	for _, entry := range archive.File {
		name, directory, ok := wheelEntryPath(entry)
		if !ok || seen[name] != nil {
			return WheelInspection{}, wheelValidation(WheelValidationFileType)
		}
		seen[name] = entry
		if entry.UncompressedSize64 > uint64(limits.MaxUncompressed) || uncompressed > limits.MaxUncompressed-int64(entry.UncompressedSize64) {
			return WheelInspection{}, wheelValidation(WheelValidationUncompressed)
		}
		uncompressed += int64(entry.UncompressedSize64)
		if !directory {
			regularFiles = append(regularFiles, name)
		}
		if isPrimaryWheelMetadata(name, primaryDistInfo) {
			if entry.UncompressedSize64 > uint64(limits.MaxMetadata) {
				return WheelInspection{}, wheelValidation(WheelValidationMetadata)
			}
			body, readErr := readZipEntry(entry, limits.MaxMetadata)
			if readErr != nil {
				return WheelInspection{}, wheelValidation(WheelValidationMetadata)
			}
			metadataFiles[path.Base(name)] = body
		}
	}
	if len(metadataFiles) != 3 {
		return WheelInspection{}, wheelValidation(WheelValidationMetadataInvalid)
	}
	info, err := parseWheelMetadata(project, version, filename, metadataFiles["METADATA"], metadataFiles["WHEEL"], metadataFiles["RECORD"], regularFiles, seen, limits)
	if err != nil {
		return WheelInspection{}, err
	}
	info.PythonTags, info.ABITags, info.PlatformTags = pyTags, abiTags, platformTags
	if !wheelMetadataTagsMatch(info.Tags, pyTags, abiTags, platformTags) {
		return WheelInspection{}, wheelValidation(WheelValidationWheelTag)
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
		return WheelInspection{}, wheelValidation(WheelValidationMetadataInvalid)
	}
	meta := headerValues(metadata, limits.MaxMetadata)
	wheelHeaders := headerValues(wheel, limits.MaxMetadata)
	metadataProject, nameErr := NormalizeProjectName(meta["name"])
	if nameErr != nil || metadataProject != project || meta["version"] != version {
		return WheelInspection{}, wheelValidation(WheelValidationMetadataIdentity)
	}
	if wheelHeaders["wheel-version"] == "" || !supportedWheelVersion(wheelHeaders["wheel-version"]) {
		return WheelInspection{}, wheelValidation(WheelValidationMetadataInvalid)
	}
	distInfo := wheelDistInfo(project, version)
	for _, name := range []string{distInfo + "/METADATA", distInfo + "/WHEEL", distInfo + "/RECORD"} {
		entry, ok := entries[name]
		if !ok {
			return WheelInspection{}, wheelValidation(WheelValidationDistInfoIdentity)
		}
		body, readErr := readZipEntry(entry, limits.MaxMetadata)
		if readErr != nil {
			return WheelInspection{}, wheelValidation(WheelValidationMetadataInvalid)
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
			return WheelInspection{}, wheelValidation(WheelValidationDistInfoIdentity)
		}
	}
	files, err := validateRecord(record, regular, entries, distInfo)
	if err != nil {
		return WheelInspection{}, wheelValidation(WheelValidationRecord)
	}
	declared, err := declaredWheelImports(splitHeaders(meta["import-name"]), splitHeaders(meta["import-namespace"]))
	if err != nil {
		return WheelInspection{}, wheelValidation(WheelValidationMetadataInvalid)
	}
	inferred := importNamesFromWheelFiles(files, distInfo, entries, limits)
	imports := unionImportSurfaces(declared, inferred)
	var entryPoints []string
	if metaEP := splitHeaders(meta["entry-points"]); len(metaEP) > 0 {
		entryPoints = append(entryPoints, metaEP...)
	}
	for _, file := range files {
		if strings.HasSuffix(file.Path, ".dist-info/entry_points.txt") && file.Path != distInfo+"/entry_points.txt" {
			return WheelInspection{}, wheelValidation(WheelValidationDistInfoIdentity)
		}
	}
	if epEntry, ok := entries[distInfo+"/entry_points.txt"]; ok {
		if epEntry.UncompressedSize64 > uint64(limits.MaxMetadata) {
			return WheelInspection{}, wheelValidation(WheelValidationMetadata)
		}
		epData, readErr := readZipEntry(epEntry, limits.MaxMetadata)
		if readErr != nil {
			return WheelInspection{}, wheelValidation(WheelValidationMetadata)
		}
		parsedEP, parseErr := parseEntryPointsTxt(epData)
		if parseErr != nil {
			return WheelInspection{}, wheelValidation(WheelValidationMetadataInvalid)
		}
		entryPoints = append(entryPoints, parsedEP...)
	}
	sort.Strings(entryPoints)
	noImportSurface := len(imports) == 0 && provenNoDynamicPythonSurface(files, distInfo, declared, entryPoints, entries, limits)
	return WheelInspection{
		Project:         project,
		Version:         version,
		Filename:        filename,
		WheelVersion:    wheelHeaders["wheel-version"],
		Tags:            splitHeaders(wheelHeaders["tag"]),
		Files:           files,
		RequiresPython:  meta["requires-python"],
		RequiresDist:    splitHeaders(meta["requires-dist"]),
		ImportNames:     imports,
		NoImportSurface: noImportSurface,
		EntryPoints:     entryPoints,
		License:         meta["license"],
		LicenseFile:     meta["license-file"],
	}, nil
}

func unionImportSurfaces(lists ...[]string) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, list := range lists {
		for _, item := range list {
			if !validPythonImportName(item) {
				continue
			}
			if _, exists := seen[item]; !exists {
				seen[item] = struct{}{}
				result = append(result, item)
			}
		}
	}
	sort.Strings(result)
	return result
}

func declaredWheelImports(names, namespaces []string) ([]string, error) {
	seen := map[string]bool{}
	result := make([]string, 0, len(names)+len(namespaces))
	for _, value := range append(names, namespaces...) {
		parts := strings.Split(value, ";")
		name := strings.TrimSpace(parts[0])
		if len(parts) > 2 || len(parts) == 2 && strings.TrimSpace(parts[1]) != "private" || !validPythonImportName(name) || seen[name] {
			return nil, errors.New("wheel import metadata is invalid")
		}
		seen[name] = true
		result = append(result, name)
	}
	return result, nil
}

func validPythonImportName(name string) bool {
	for _, part := range strings.Split(name, ".") {
		if !validPythonImportComponent(part) {
			return false
		}
	}
	return true
}

func provenNoDynamicPythonSurface(files []WheelFile, distInfo string, declaredImports, entryPoints []string, entries map[string]*zip.File, limits WheelLimits) bool {
	if len(declaredImports) > 0 || len(entryPoints) > 0 {
		return false
	}
	distInfoSeen := map[string]bool{}
	for _, file := range files {
		if strings.HasPrefix(file.Path, distInfo+"/") {
			distInfoSeen[file.Path] = true
		}
	}
	if !distInfoSeen[distInfo+"/METADATA"] || !distInfoSeen[distInfo+"/WHEEL"] || !distInfoSeen[distInfo+"/RECORD"] {
		return false
	}
	var payloadFiles []WheelFile
	for _, file := range files {
		if strings.HasPrefix(file.Path, distInfo+"/") {
			name := file.Path
			if isPythonExecutableSurface(name) {
				return false
			}
			continue
		}
		payloadFiles = append(payloadFiles, file)
	}
	if len(payloadFiles) == 0 {
		return true
	}
	hasRecognizedPayload := false
	for _, file := range payloadFiles {
		name := file.Path
		if isPythonExecutableSurface(name) {
			return false
		}
		if !strings.Contains(name, "/") {
			return false
		}
		if isBareSO(name) {
			entry := entries[name]
			if entry == nil {
				return false
			}
			class, _, err := classifyBareSO(entry, limits.MaxUncompressed)
			if err != nil || class != soProvenNative {
				return false
			}
			hasRecognizedPayload = true
			continue
		}
		if !isRecognizedNativeOrDataPayload(name, distInfo) {
			return false
		}
		hasRecognizedPayload = true
	}
	return hasRecognizedPayload
}

func isPythonExecutableSurface(name string) bool {
	if strings.HasSuffix(name, ".py") || strings.HasSuffix(name, ".pyw") ||
		strings.HasSuffix(name, ".pyc") || strings.HasSuffix(name, ".pyo") ||
		strings.HasSuffix(name, ".pyd") || strings.HasSuffix(name, ".pth") {
		return true
	}
	if _, ok := pythonExtensionImportName(name); ok {
		return true
	}
	if strings.Contains(name, ".data/scripts/") || strings.HasPrefix(name, "bin/") {
		return true
	}
	if strings.HasSuffix(name, ".egg-link") {
		return true
	}
	return false
}

func isRecognizedNativeOrDataPayload(name, distInfo string) bool {
	base := path.Base(name)
	upper := strings.ToUpper(base)
	for _, prefix := range []string{"LICENSE", "NOTICE", "README", "COPYING", "AUTHORS", "PATENTS"} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	if isVersionedNativeSO(base) ||
		strings.HasSuffix(name, ".a") || strings.HasSuffix(name, ".dylib") ||
		strings.HasSuffix(name, ".dll") {
		return true
	}
	for _, ext := range []string{".h", ".hpp", ".hxx", ".cuh", ".inc", ".h.in", ".c", ".cpp", ".cc"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	for _, ext := range []string{".pc", ".pc.in", ".cmake", ".cmake.in"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	for _, ext := range []string{".txt", ".md", ".rst", ".json", ".yaml", ".yml", ".xml", ".csv", ".toml", ".ini", ".cfg", ".dat", ".bin", ".pdf", ".html", ".css"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	dataRoot := strings.TrimSuffix(distInfo, ".dist-info") + ".data/"
	if strings.HasPrefix(name, dataRoot+"headers/") || strings.HasPrefix(name, dataRoot+"data/") {
		return true
	}
	return false
}

func isVersionedNativeSO(base string) bool {
	index := strings.LastIndex(base, ".so.")
	if index <= 0 || index+4 >= len(base) {
		return false
	}
	for _, part := range strings.Split(base[index+4:], ".") {
		if part == "" {
			return false
		}
		for _, character := range part {
			if character < '0' || character > '9' {
				return false
			}
		}
	}
	return true
}

// importNamesFromWheelFiles is a bounded static inspection discovering Python
// .py modules/packages and recognized Python native extension modules. It does
// NOT treat arbitrary payload directories as dynamic Python import surfaces.
func importNamesFromWheelFiles(files []WheelFile, distInfo string, entries map[string]*zip.File, limits WheelLimits) []string {
	seen := map[string]struct{}{}
	// 1. .py module / package discovery
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
	// 2. Python native extension discovery (combined with .py discovery, not only fallback)
	for _, file := range files {
		name := file.Path
		if strings.HasPrefix(name, distInfo+"/") || strings.Contains(name, ".data/") {
			continue
		}
		if candidate, ok := pythonExtensionImportName(name); ok && validPythonImportName(candidate) {
			seen[candidate] = struct{}{}
		}
	}
	// 3. Bare .so discovery: if a bare .so is proven to be a Python extension (exports PyInit_<module>),
	// derive its candidate import name.
	for _, file := range files {
		name := file.Path
		if strings.HasPrefix(name, distInfo+"/") || strings.Contains(name, ".data/") {
			continue
		}
		if isBareSO(name) {
			if entry := entries[name]; entry != nil {
				class, modName, err := classifyBareSO(entry, limits.MaxUncompressed)
				if err == nil && class == soPythonExtension && modName != "" {
					candidate := bareSOImportName(name, modName)
					if candidate != "" && validPythonImportName(candidate) {
						seen[candidate] = struct{}{}
					}
				}
			}
		}
	}
	imports := make([]string, 0, len(seen))
	for name := range seen {
		imports = append(imports, name)
	}
	sort.Strings(imports)
	return imports
}

func pythonExtensionImportName(path string) (string, bool) {
	parts := strings.Split(path, "/")
	base := parts[len(parts)-1]
	var module string
	if strings.HasSuffix(base, ".so") {
		at := strings.Index(base, ".cpython-")
		if at < 0 {
			at = strings.Index(base, ".abi3.so")
		}
		if at <= 0 {
			return "", false
		}
		module = base[:at]
	} else if strings.HasSuffix(base, ".pyd") {
		trimmed := strings.TrimSuffix(base, ".pyd")
		if at := strings.Index(trimmed, ".cpython-"); at > 0 {
			module = trimmed[:at]
		} else if at := strings.Index(trimmed, ".cp"); at > 0 {
			module = trimmed[:at]
		} else {
			module = trimmed
		}
	} else {
		return "", false
	}
	if !validPythonImportComponent(module) {
		return "", false
	}
	parts[len(parts)-1] = module
	candidate := strings.Join(parts, ".")
	if !validPythonImportName(candidate) {
		return "", false
	}
	return candidate, true
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

type soClassification int

const (
	soAmbiguous soClassification = iota
	soProvenNative
	soPythonExtension
)

func isBareSO(name string) bool {
	if !strings.HasSuffix(name, ".so") {
		return false
	}
	base := path.Base(name)
	return !strings.Contains(base, ".cpython-") && !strings.Contains(base, ".abi3.so")
}

func classifyBareSO(entry *zip.File, maxUncompressed int64) (class soClassification, pyInitModule string, err error) {
	if entry == nil || entry.UncompressedSize64 > uint64(maxUncompressed) || entry.UncompressedSize64 < 64 {
		return soAmbiguous, "", nil
	}
	r, err := entry.Open()
	if err != nil {
		return soAmbiguous, "", err
	}
	defer r.Close()

	data, err := io.ReadAll(io.LimitReader(r, int64(entry.UncompressedSize64)+1))
	if err != nil || int64(len(data)) != int64(entry.UncompressedSize64) {
		return soAmbiguous, "", err
	}

	class, modName, err := classifyELFBytes(data)
	if err != nil || class != soPythonExtension {
		return class, modName, err
	}

	// For a Python extension, the exported PyInit_<module> must match the file basename.
	base := path.Base(entry.Name)
	fileBase := strings.TrimSuffix(base, ".so")
	if modName != fileBase {
		return soAmbiguous, "", nil
	}
	return soPythonExtension, modName, nil
}

func classifyELFBytes(data []byte) (class soClassification, pyInitModule string, err error) {
	if len(data) < 64 || !bytes.HasPrefix(data, []byte("\x7fELF")) {
		return soAmbiguous, "", nil
	}

	var (
		f       *elf.File
		openErr error
	)
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				openErr = errors.New("elf parsing panic")
			}
		}()
		f, openErr = elf.NewFile(bytes.NewReader(data))
	}()
	if openErr != nil || f == nil {
		return soAmbiguous, "", nil
	}
	defer f.Close()

	if f.Type != elf.ET_DYN {
		return soAmbiguous, "", nil
	}

	var (
		symbols []elf.Symbol
		symErr  error
	)
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				symErr = errors.New("dynamic symbols panic")
			}
		}()
		symbols, symErr = f.DynamicSymbols()
	}()
	if symErr != nil {
		return soAmbiguous, "", nil
	}

	var pyInitSymbols []string
	for _, sym := range symbols {
		if strings.HasPrefix(sym.Name, "PyInit_") {
			if sym.Section != elf.SHN_UNDEF {
				mod := strings.TrimPrefix(sym.Name, "PyInit_")
				pyInitSymbols = append(pyInitSymbols, mod)
			} else {
				// Undefined/imported PyInit_ reference is abnormal for native libraries.
				return soAmbiguous, "", nil
			}
		}
	}

	if len(pyInitSymbols) == 0 {
		return soProvenNative, "", nil
	}
	if len(pyInitSymbols) == 1 {
		mod := pyInitSymbols[0]
		if validPythonImportComponent(mod) {
			return soPythonExtension, mod, nil
		}
		return soAmbiguous, "", nil
	}
	return soAmbiguous, "", nil
}

func bareSOImportName(filePath, modName string) string {
	parts := strings.Split(filePath, "/")
	fileBase := strings.TrimSuffix(parts[len(parts)-1], ".so")
	if fileBase != modName || !validPythonImportComponent(modName) {
		return ""
	}
	parts[len(parts)-1] = modName
	candidate := strings.Join(parts, ".")
	if !validPythonImportName(candidate) {
		return ""
	}
	return candidate
}

func parseEntryPointsTxt(data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		return nil, errors.New("malformed entry_points.txt: invalid UTF-8")
	}
	lines := strings.Split(string(data), "\n")
	currentSection := ""
	var results []string

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return nil, errors.New("malformed entry_points.txt: unclosed section header")
			}
			section := strings.TrimSpace(line[1 : len(line)-1])
			if !validEntryPointGroup(section) {
				return nil, errors.New("malformed entry_points.txt: invalid section name")
			}
			currentSection = section
			continue
		}
		if currentSection == "" {
			return nil, errors.New("malformed entry_points.txt: entry outside section")
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			return nil, errors.New("malformed entry_points.txt: missing '='")
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if key == "" || val == "" {
			return nil, errors.New("malformed entry_points.txt: empty key or value")
		}
		if !validEntryPointName(key) {
			return nil, errors.New("malformed entry_points.txt: invalid entry point name")
		}
		modSpec := val
		if bracketIdx := strings.Index(val, "["); bracketIdx >= 0 {
			if !strings.HasSuffix(val, "]") {
				return nil, errors.New("malformed entry_points.txt: unclosed extras bracket")
			}
			modSpec = strings.TrimSpace(val[:bracketIdx])
		}
		parts := strings.Split(modSpec, ":")
		if len(parts) > 2 {
			return nil, errors.New("malformed entry_points.txt: multiple colons in target")
		}
		moduleName := strings.TrimSpace(parts[0])
		if !validPythonImportName(moduleName) {
			return nil, errors.New("malformed entry_points.txt: invalid module name in target")
		}
		if len(parts) == 2 {
			attr := strings.TrimSpace(parts[1])
			if attr == "" {
				return nil, errors.New("malformed entry_points.txt: empty attribute in target")
			}
			for _, part := range strings.Split(attr, ".") {
				if !validPythonImportComponent(part) {
					return nil, errors.New("malformed entry_points.txt: invalid attribute identifier")
				}
			}
		}
		results = append(results, currentSection+": "+key+" = "+val)
	}
	sort.Strings(results)
	return results, nil
}

func validEntryPointGroup(group string) bool {
	if group == "" {
		return false
	}
	for _, component := range strings.Split(group, ".") {
		if component == "" {
			return false
		}
		for _, character := range component {
			if character != '_' && !unicode.IsLetter(character) && !unicode.IsDigit(character) {
				return false
			}
		}
	}
	return true
}

func validEntryPointName(name string) bool {
	if name == "" || strings.HasPrefix(name, "[") || name != strings.TrimSpace(name) {
		return false
	}
	for _, character := range name {
		if character == '=' || character == '\r' || character == '\n' || (unicode.IsControl(character) && character != '\t') {
			return false
		}
	}
	return true
}

func validateRecord(body []byte, regular []string, entries map[string]*zip.File, primaryDistInfo string) ([]WheelFile, error) {
	primaryRecord := primaryDistInfo + "/RECORD"
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
			if name != primaryRecord {
				return nil, errors.New("wheel RECORD exemption is invalid")
			}
			recorded[name] = WheelFile{Path: name}
			continue
		}
		if name == primaryRecord {
			return nil, errors.New("wheel RECORD self-entry must be empty")
		}
		if !strings.HasPrefix(row[1], "sha256=") {
			return nil, errors.New("wheel RECORD digest is invalid")
		}
		digest, err := base64.RawURLEncoding.DecodeString(row[1][len("sha256="):])
		if err != nil || len(digest) != sha256.Size {
			return nil, errors.New("wheel RECORD digest is invalid")
		}
		n, err := strconv.ParseInt(row[2], 10, 64)
		if err != nil || n < 0 {
			return nil, errors.New("wheel RECORD size is invalid")
		}
		recorded[name] = WheelFile{Path: name, Size: n, SHA256: hex.EncodeToString(digest)}
	}
	for _, name := range regular {
		if name == "" {
			continue
		}
		if name == primaryDistInfo+"/RECORD.jws" || name == primaryDistInfo+"/RECORD.p7s" {
			continue
		}
		file, ok := recorded[name]
		if !ok || file.Path == "" {
			return nil, errors.New("wheel contains unrecorded file")
		}
		if name == primaryRecord && file.Size == 0 && file.SHA256 == "" {
			continue
		}
		entry := entries[name]
		actualSize, actualDigest, err := streamRecordFile(entry)
		if err != nil || actualSize != file.Size {
			return nil, errors.New("wheel RECORD size mismatch")
		}
		if actualDigest != file.SHA256 {
			return nil, errors.New("wheel RECORD digest mismatch")
		}
	}
	expected := 0
	for _, name := range regular {
		if name != primaryDistInfo+"/RECORD.jws" && name != primaryDistInfo+"/RECORD.p7s" {
			expected++
		}
	}
	if len(recorded) != expected {
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

func streamRecordFile(file *zip.File) (int64, string, error) {
	if file == nil {
		return 0, "", errors.New("wheel RECORD file is missing")
	}
	r, err := file.Open()
	if err != nil {
		return 0, "", err
	}
	defer r.Close()
	hash := sha256.New()
	if file.UncompressedSize64 > uint64(1<<63-2) {
		return 0, "", errors.New("wheel RECORD file exceeds streaming bound")
	}
	size, err := io.Copy(hash, io.LimitReader(r, int64(file.UncompressedSize64)+1))
	if err != nil {
		return 0, "", err
	}
	if size > int64(file.UncompressedSize64) {
		return 0, "", errors.New("wheel RECORD file exceeds declared archive size")
	}
	return size, hex.EncodeToString(hash.Sum(nil)), nil
}

func (i WheelInspection) distInfo() string {
	return wheelDistInfo(i.Project, i.Version)
}

func wheelDistInfo(project, version string) string {
	return strings.ReplaceAll(project, "-", "_") + "-" + strings.ReplaceAll(version, "-", "_") + ".dist-info"
}

func isPrimaryWheelMetadata(name, distInfo string) bool {
	return name == distInfo+"/METADATA" || name == distInfo+"/WHEEL" || name == distInfo+"/RECORD"
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

func wheelEntryPath(entry *zip.File) (string, bool, bool) {
	if entry == nil {
		return "", false, false
	}
	mode := entry.FileInfo().Mode()
	if strings.HasSuffix(entry.Name, "/") {
		name, ok := wheelPath(strings.TrimSuffix(entry.Name, "/"))
		if !ok || !mode.IsDir() || mode&os.ModeSymlink != 0 || entry.UncompressedSize64 != 0 {
			return "", false, false
		}
		return name, true, true
	}
	if mode&os.ModeSymlink != 0 || mode.IsDir() || !mode.IsRegular() {
		return "", false, false
	}
	name, ok := wheelPath(entry.Name)
	return name, false, ok
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
		if key == "requires-dist" || key == "import-name" || key == "import-namespace" || key == "entry-points" || key == "tag" {
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
	// The pinned x86_64 runtime also accepts the standardized legacy aliases.
	// Keep architecture binding explicit; never treat an arbitrary linux tag
	// as a manylinux compatibility promise.
	aliases := map[string]string{
		"manylinux1_x86_64":    "manylinux_2_5_x86_64",
		"manylinux2010_x86_64": "manylinux_2_12_x86_64",
		"manylinux2014_x86_64": "manylinux_2_17_x86_64",
	}
	if canonical, ok := aliases[wheel]; ok {
		wheel = canonical
	}
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
	expanded, ok := expandWheelTags(py, abi, platform)
	if !ok || len(tags) == 0 || len(tags) != len(expanded) {
		return false
	}
	want := make(map[string]struct{}, len(expanded))
	for _, tag := range expanded {
		want[tag] = struct{}{}
	}
	got := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if !validExpandedWheelTag(tag) {
			return false
		}
		if _, duplicate := got[tag]; duplicate {
			return false
		}
		got[tag] = struct{}{}
		if _, present := want[tag]; !present {
			return false
		}
	}
	return len(got) == len(want)
}

func expandWheelTags(py, abi, platform []string) ([]string, bool) {
	if len(py) == 0 || len(abi) == 0 || len(platform) == 0 {
		return nil, false
	}
	seen := map[string]struct{}{}
	expanded := make([]string, 0, len(py)*len(abi)*len(platform))
	for _, python := range py {
		for _, applicationBinaryInterface := range abi {
			for _, operatingSystem := range platform {
				tag := python + "-" + applicationBinaryInterface + "-" + operatingSystem
				if !validExpandedWheelTag(tag) {
					return nil, false
				}
				if _, duplicate := seen[tag]; duplicate {
					return nil, false
				}
				seen[tag] = struct{}{}
				expanded = append(expanded, tag)
			}
		}
	}
	return expanded, true
}

func validExpandedWheelTag(tag string) bool {
	parts := strings.Split(tag, "-")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" || strings.Contains(part, ".") {
			return false
		}
		for _, character := range part {
			if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '_' {
				continue
			}
			return false
		}
	}
	return true
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
