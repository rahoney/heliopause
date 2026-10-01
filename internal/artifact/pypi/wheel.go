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
	Surface              RuntimeSurface
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
	return info, nil
}

func parseWheelMetadata(project, version, filename string, metadata, wheel, record []byte, regular []string, entries map[string]*zip.File, limits WheelLimits) (WheelInspection, error) {
	if len(metadata) == 0 || len(wheel) == 0 || len(record) == 0 {
		return WheelInspection{}, wheelValidation(WheelValidationMetadataInvalid)
	}
	metaHeaders, err := parseRFC822Metadata(metadata, limits.MaxMetadata)
	if err != nil {
		return WheelInspection{}, wheelValidation(WheelValidationMetadataInvalid)
	}
	wheelHeaders, err := parseRFC822Metadata(wheel, limits.MaxMetadata)
	if err != nil {
		return WheelInspection{}, wheelValidation(WheelValidationMetadataInvalid)
	}
	rawName, hasName := metaHeaders.first("name")
	rawVersion, hasVersion := metaHeaders.first("version")
	if !hasName || !hasVersion {
		return WheelInspection{}, wheelValidation(WheelValidationMetadataIdentity)
	}
	metadataProject, nameErr := NormalizeProjectName(rawName)
	if nameErr != nil || metadataProject != project || rawVersion != version {
		return WheelInspection{}, wheelValidation(WheelValidationMetadataIdentity)
	}
	wheelVer, hasWheelVer := wheelHeaders.first("wheel-version")
	if !hasWheelVer || wheelVer == "" || !supportedWheelVersion(wheelVer) {
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
	declared, err := declaredWheelImports(metaHeaders.all("import-name"), metaHeaders.all("import-namespace"))
	if err != nil {
		return WheelInspection{}, wheelValidation(WheelValidationMetadataInvalid)
	}
	var entryPoints []string
	var epDetails []EntryPoint
	if metaEP := metaHeaders.all("entry-points"); len(metaEP) > 0 {
		entryPoints = append(entryPoints, metaEP...)
	}
	if epEntry, ok := entries[distInfo+"/entry_points.txt"]; ok {
		if epEntry.UncompressedSize64 > uint64(limits.MaxMetadata) {
			return WheelInspection{}, wheelValidation(WheelValidationMetadata)
		}
		epData, readErr := readZipEntry(epEntry, limits.MaxMetadata)
		if readErr != nil {
			return WheelInspection{}, wheelValidation(WheelValidationMetadata)
		}
		parsedEP, details, parseErr := parseEntryPointsTxt(epData)
		if parseErr != nil {
			return WheelInspection{}, wheelValidation(WheelValidationMetadataInvalid)
		}
		entryPoints = append(entryPoints, parsedEP...)
		epDetails = append(epDetails, details...)
	}
	sort.Strings(entryPoints)

	dataDir := strings.TrimSuffix(distInfo, ".dist-info") + ".data"
	surface, imports, noImportSurface, err := classifyWheelSurface(files, distInfo, dataDir, declared, entryPoints, epDetails, entries, limits)
	if err != nil {
		return WheelInspection{}, err
	}

	reqPy, _ := metaHeaders.first("requires-python")
	lic, _ := metaHeaders.first("license")
	licFile, _ := metaHeaders.first("license-file")

	nativeExts := append(append([]string{}, surface.PythonExtensions...), surface.NativeLibraries...)
	sort.Strings(nativeExts)

	return WheelInspection{
		Project:          project,
		Version:          version,
		Filename:         filename,
		WheelVersion:     wheelVer,
		Tags:             wheelHeaders.all("tag"),
		Files:            files,
		RequiresPython:   reqPy,
		RequiresDist:     metaHeaders.all("requires-dist"),
		ImportNames:      imports,
		NoImportSurface:  noImportSurface,
		EntryPoints:      entryPoints,
		Scripts:          surface.Scripts,
		NativeExtensions: nativeExts,
		License:          lic,
		LicenseFile:      licFile,
		Surface:          surface,
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

func mapWheelInstalledLocation(archivePath, distInfo, dataDir string) (InstallationScheme, string, error) {
	if archivePath == distInfo || strings.HasPrefix(archivePath, distInfo+"/") {
		return SchemeDistInfo, archivePath, nil
	}
	if archivePath == dataDir || strings.HasPrefix(archivePath, dataDir+"/") {
		rel := strings.TrimPrefix(archivePath, dataDir+"/")
		parts := strings.Split(rel, "/")
		if len(parts) < 2 || parts[1] == "" {
			return "", "", errors.New("empty entry in .data directory")
		}
		category := parts[0]
		subPath := strings.Join(parts[1:], "/")
		switch category {
		case "purelib", "platlib":
			destParts := strings.Split(subPath, "/")
			if strings.HasSuffix(destParts[0], ".dist-info") && destParts[0] != distInfo {
				return "", "", wheelValidation(WheelValidationDistInfoIdentity)
			}
			return SchemeSite, subPath, nil
		case "scripts":
			return SchemeScripts, "bin/" + subPath, nil
		case "headers":
			return SchemeHeaders, "include/" + subPath, nil
		case "data":
			return SchemeData, "share/" + subPath, nil
		default:
			return "", "", errors.New("unsupported .data category: " + category)
		}
	}
	parts := strings.Split(archivePath, "/")
	if strings.HasSuffix(parts[0], ".dist-info") {
		if parts[0] != distInfo {
			return "", "", wheelValidation(WheelValidationDistInfoIdentity)
		}
		return SchemeDistInfo, archivePath, nil
	}
	if strings.HasSuffix(parts[0], ".data") {
		return "", "", errors.New("unsupported foreign .data directory in wheel root")
	}
	return SchemeSite, archivePath, nil
}

func isRecognizedInertData(name string) bool {
	if strings.Contains(name, ".dist-info/") {
		return true
	}
	if strings.Contains(name, "/") && strings.HasSuffix(name, ".pth") {
		return true
	}
	base := path.Base(name)
	upper := strings.ToUpper(base)
	for _, prefix := range []string{"LICENSE", "NOTICE", "README", "COPYING", "AUTHORS", "PATENTS"} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
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
	for _, ext := range []string{
		".txt", ".md", ".rst", ".json", ".yaml", ".yml", ".xml", ".csv",
		".toml", ".ini", ".cfg", ".dat", ".bin", ".pdf", ".html", ".css",
		".js", ".svg", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".wasm", ".xsl",
	} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	for _, ext := range []string{
		".pxd", ".pyx", ".pxi", ".tmpl", ".in", ".jinja", ".j2",
		".pyi", ".typed", ".exe", ".g4", ".lark", ".al", ".lock",
	} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

func deduplicateSorted(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	sort.Strings(items)
	result := make([]string, 0, len(items))
	for i, item := range items {
		if i == 0 || item != items[i-1] {
			result = append(result, item)
		}
	}
	return result
}

// parseSiteHookLines accepts the active site-level subset of CPython .pth
// syntax. Declarative paths are checked against the complete authenticated
// installed closure before execution; unsupported syntax remains fail closed.
func parseSiteHookLines(file string, body []byte) ([]SiteHookLine, error) {
	if !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 {
		return nil, errors.New("active site hook is not UTF-8 text")
	}
	var lines []SiteHookLine
	for index, raw := range strings.Split(string(body), "\n") {
		raw = strings.TrimSuffix(raw, "\r")
		if strings.TrimSpace(raw) == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		line := strings.TrimSpace(raw)
		item := SiteHookLine{File: file, Line: index + 1}
		if strings.HasPrefix(raw, "import ") || strings.HasPrefix(raw, "import\t") {
			item.Statement = raw
		} else {
			if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "import\t") || strings.HasPrefix(line, "#") {
				return nil, errors.New("active site line has unsupported indentation")
			}
			if path.IsAbs(line) || path.Clean(line) != line || line == "." || strings.ContainsAny(line, "\\:\x00") {
				return nil, errors.New("active site path is not canonical")
			}
			for _, component := range strings.Split(line, "/") {
				if component == "" || component == "." || component == ".." {
					return nil, errors.New("active site path escapes installed closure")
				}
			}
			item.Path = line
		}
		lines = append(lines, item)
	}
	return lines, nil
}

func classifyWheelSurface(files []WheelFile, distInfo, dataDir string, declaredImports, epList []string, epDetails []EntryPoint, entries map[string]*zip.File, limits WheelLimits) (RuntimeSurface, []string, bool, error) {
	var surface RuntimeSurface
	surface.EntryPoints = append(surface.EntryPoints, epList...)
	surface.EntryPointDetails = append(surface.EntryPointDetails, epDetails...)

	type mappedFile struct {
		file   WheelFile
		scheme InstallationScheme
		dest   string
	}
	mapped := make([]mappedFile, 0, len(files))
	initDirSet := make(map[string]bool)
	seenDestinations := make(map[string]string, len(files))

	for _, file := range files {
		scheme, dest, err := mapWheelInstalledLocation(file.Path, distInfo, dataDir)
		if err != nil {
			return RuntimeSurface{}, nil, false, err
		}
		destKey := dest
		if prev, exists := seenDestinations[destKey]; exists {
			_ = prev
			return RuntimeSurface{}, nil, false, wheelValidation(WheelValidationRecord)
		}
		seenDestinations[destKey] = file.Path
		mapped = append(mapped, mappedFile{file: file, scheme: scheme, dest: dest})
		if scheme == SchemeSite {
			if dest == "__init__.py" {
				initDirSet[""] = true
			} else if strings.HasSuffix(dest, "/__init__.py") {
				dir := strings.TrimSuffix(dest, "/__init__.py")
				initDirSet[dir] = true
			}
		}
	}

	for _, m := range mapped {
		var role RuntimeRole

		switch m.scheme {
		case SchemeDistInfo:
			if strings.HasSuffix(m.dest, ".py") || strings.HasSuffix(m.dest, ".pth") {
				role = RuntimeRoleUnresolved
				surface.UnresolvedFiles = append(surface.UnresolvedFiles, m.dest)
			} else {
				role = RuntimeRoleMetadata
			}

		case SchemeScripts:
			role = RuntimeRoleScript
			surface.Scripts = append(surface.Scripts, m.dest)

		case SchemeHeaders, SchemeData:
			role = RuntimeRoleInertData
			surface.InertDataFiles = append(surface.InertDataFiles, m.dest)

		case SchemeSite:
			dest := m.dest

			if !strings.Contains(dest, "/") && strings.HasSuffix(dest, ".pth") {
				role = RuntimeRoleSiteStartupHook
				surface.SiteStartupHooks = append(surface.SiteStartupHooks, dest)
				entry := entries[m.file.Path]
				if entry == nil || entry.UncompressedSize64 > uint64(limits.MaxMetadata) {
					return RuntimeSurface{}, nil, false, wheelValidation(WheelValidationMetadata)
				}
				body, readErr := readZipEntry(entry, limits.MaxMetadata)
				if readErr != nil {
					return RuntimeSurface{}, nil, false, wheelValidation(WheelValidationMetadata)
				}
				lines, parseErr := parseSiteHookLines(dest, body)
				if parseErr != nil {
					return RuntimeSurface{}, nil, false, wheelValidation(WheelValidationMetadataInvalid)
				}
				surface.SiteHookLines = append(surface.SiteHookLines, lines...)

			} else if !strings.Contains(dest, "/") && strings.HasSuffix(dest, ".py") {
				mod := strings.TrimSuffix(dest, ".py")
				if mod != "__init__" && validPythonImportComponent(mod) {
					role = RuntimeRolePythonModule
					surface.PythonModules = append(surface.PythonModules, mod)
				} else {
					role = RuntimeRoleInertData
					surface.InertDataFiles = append(surface.InertDataFiles, dest)
				}

			} else if isWrongTargetExtension(dest) {
				role = RuntimeRoleUnresolved
				surface.UnresolvedFiles = append(surface.UnresolvedFiles, dest)

			} else if extMod, ok := pythonExtensionImportName(dest); ok && validPythonImportName(extMod) {
				role = RuntimeRolePythonExtension
				surface.PythonExtensions = append(surface.PythonExtensions, extMod)

			} else if isBareSO(dest) {
				entry := entries[m.file.Path]
				if entry != nil {
					class, modName, err := classifyBareSO(entry, limits.MaxUncompressed)
					if err == nil && class == soPythonExtension && modName != "" {
						candidate := bareSOImportName(dest, modName)
						if candidate != "" && validPythonImportName(candidate) {
							role = RuntimeRolePythonExtension
							surface.PythonExtensions = append(surface.PythonExtensions, candidate)
						} else {
							role = RuntimeRoleUnresolved
							surface.UnresolvedFiles = append(surface.UnresolvedFiles, dest)
						}
					} else if err == nil && class == soProvenNative {
						role = RuntimeRoleNativeLibrary
						surface.NativeLibraries = append(surface.NativeLibraries, dest)
					} else {
						role = RuntimeRoleUnresolved
						surface.UnresolvedFiles = append(surface.UnresolvedFiles, dest)
					}
				} else {
					role = RuntimeRoleUnresolved
					surface.UnresolvedFiles = append(surface.UnresolvedFiles, dest)
				}

			} else if isVersionedNativeSO(path.Base(dest)) || strings.HasSuffix(dest, ".a") || strings.HasSuffix(dest, ".dylib") || strings.HasSuffix(dest, ".dll") {
				role = RuntimeRoleNativeLibrary
				surface.NativeLibraries = append(surface.NativeLibraries, dest)

			} else if strings.Contains(dest, "/") {
				parts := strings.Split(dest, "/")
				topDir := parts[0]

				if initDirSet[topDir] {
					if dest == topDir+"/__init__.py" {
						role = RuntimeRolePythonPackage
						if validPythonImportComponent(topDir) {
							surface.PythonPackages = append(surface.PythonPackages, topDir)
						}
					} else if strings.HasSuffix(dest, ".py") {
						role = RuntimeRolePythonPackage
					} else if isRecognizedInertData(dest) {
						role = RuntimeRoleInertData
						surface.InertDataFiles = append(surface.InertDataFiles, dest)
					} else {
						role = RuntimeRoleUnresolved
						surface.UnresolvedFiles = append(surface.UnresolvedFiles, dest)
					}
				} else {
					surface.NamespaceContainers = append(surface.NamespaceContainers, topDir)

					if len(parts) > 1 {
						subDir := parts[0] + "/" + parts[1]
						if initDirSet[subDir] {
							subpkgName := parts[0] + "." + parts[1]
							if dest == subDir+"/__init__.py" {
								role = RuntimeRoleExecutableSubpackage
								if validPythonImportName(subpkgName) {
									surface.ExecutableSubpackages = append(surface.ExecutableSubpackages, subpkgName)
								}
							} else if strings.HasSuffix(dest, ".py") {
								role = RuntimeRoleExecutableSubpackage
							} else if isRecognizedInertData(dest) {
								role = RuntimeRoleInertData
								surface.InertDataFiles = append(surface.InertDataFiles, dest)
							} else {
								role = RuntimeRoleUnresolved
								surface.UnresolvedFiles = append(surface.UnresolvedFiles, dest)
							}
						} else if len(parts) == 2 && strings.HasSuffix(parts[1], ".py") {
							modName := parts[0] + "." + strings.TrimSuffix(parts[1], ".py")
							if validPythonImportName(modName) {
								role = RuntimeRolePythonModule
								surface.PythonModules = append(surface.PythonModules, modName)
							} else {
								role = RuntimeRoleInertData
								surface.InertDataFiles = append(surface.InertDataFiles, dest)
							}
						} else if isRecognizedInertData(dest) {
							role = RuntimeRoleInertData
							surface.InertDataFiles = append(surface.InertDataFiles, dest)
						} else {
							role = RuntimeRoleUnresolved
							surface.UnresolvedFiles = append(surface.UnresolvedFiles, dest)
						}
					} else {
						if isRecognizedInertData(dest) {
							role = RuntimeRoleInertData
							surface.InertDataFiles = append(surface.InertDataFiles, dest)
						} else {
							role = RuntimeRoleUnresolved
							surface.UnresolvedFiles = append(surface.UnresolvedFiles, dest)
						}
					}
				}

			} else if isRecognizedInertData(dest) {
				role = RuntimeRoleInertData
				surface.InertDataFiles = append(surface.InertDataFiles, dest)

			} else {
				role = RuntimeRoleUnresolved
				surface.UnresolvedFiles = append(surface.UnresolvedFiles, dest)
			}
		}

		surface.InstalledFiles = append(surface.InstalledFiles, InstalledFile{
			ArchivePath: m.file.Path,
			Scheme:      m.scheme,
			Destination: m.dest,
			Role:        role,
			Size:        m.file.Size,
			SHA256:      m.file.SHA256,
		})
	}

	surface.SiteStartupHooks = deduplicateSorted(surface.SiteStartupHooks)
	surface.PythonModules = deduplicateSorted(surface.PythonModules)
	surface.PythonPackages = deduplicateSorted(surface.PythonPackages)
	surface.NamespaceContainers = deduplicateSorted(surface.NamespaceContainers)
	surface.ExecutableSubpackages = deduplicateSorted(surface.ExecutableSubpackages)
	surface.PythonExtensions = deduplicateSorted(surface.PythonExtensions)
	surface.NativeLibraries = deduplicateSorted(surface.NativeLibraries)
	surface.Scripts = deduplicateSorted(surface.Scripts)
	surface.InertDataFiles = deduplicateSorted(surface.InertDataFiles)
	surface.UnresolvedFiles = deduplicateSorted(surface.UnresolvedFiles)

	inferred := unionImportSurfaces(surface.PythonModules, surface.PythonPackages, surface.ExecutableSubpackages, surface.PythonExtensions)
	imports := unionImportSurfaces(declaredImports, inferred)

	noImportSurface := len(imports) == 0 &&
		len(surface.SiteStartupHooks) == 0 &&
		len(surface.UnresolvedFiles) == 0 &&
		len(surface.Scripts) == 0 &&
		len(declaredImports) == 0 &&
		len(epList) == 0 &&
		provenNoImportSurfaceRoles(surface.InstalledFiles)

	return surface, imports, noImportSurface, nil
}

func provenNoImportSurfaceRoles(installed []InstalledFile) bool {
	if len(installed) == 0 {
		return false
	}
	hasNativeLib := false
	isMetadataOnly := true
	for _, f := range installed {
		switch f.Role {
		case RuntimeRoleMetadata:
		case RuntimeRoleNativeLibrary:
			hasNativeLib = true
			isMetadataOnly = false
		case RuntimeRoleInertData:
			isMetadataOnly = false
		default:
			return false
		}
	}
	return isMetadataOnly || hasNativeLib
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

func isWrongTargetExtension(path string) bool {
	base := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		base = path[idx+1:]
	}
	if strings.HasSuffix(base, ".pyd") {
		return true
	}
	if strings.HasSuffix(base, ".so") && strings.Contains(base, ".cpython-") {
		return !strings.HasSuffix(base, ".cpython-314-x86_64-linux-gnu.so")
	}
	return false
}

func pythonExtensionImportName(path string) (string, bool) {
	parts := strings.Split(path, "/")
	base := parts[len(parts)-1]
	var module string
	if strings.HasSuffix(base, ".so") {
		if strings.HasSuffix(base, ".cpython-314-x86_64-linux-gnu.so") {
			at := strings.Index(base, ".cpython-314-x86_64-linux-gnu.so")
			if at <= 0 {
				return "", false
			}
			module = base[:at]
		} else if strings.HasSuffix(base, ".abi3.so") {
			at := strings.Index(base, ".abi3.so")
			if at <= 0 {
				return "", false
			}
			module = base[:at]
		} else {
			return "", false
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

func parseEntryPointsTxt(data []byte) ([]string, []EntryPoint, error) {
	if len(data) == 0 {
		return nil, nil, nil
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		return nil, nil, errors.New("malformed entry_points.txt: invalid UTF-8")
	}
	lines := strings.Split(string(data), "\n")
	currentSection := ""
	var results []string
	var details []EntryPoint

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return nil, nil, errors.New("malformed entry_points.txt: unclosed section header")
			}
			section := strings.TrimSpace(line[1 : len(line)-1])
			if !validEntryPointGroup(section) {
				return nil, nil, errors.New("malformed entry_points.txt: invalid section name")
			}
			currentSection = section
			continue
		}
		if currentSection == "" {
			return nil, nil, errors.New("malformed entry_points.txt: entry outside section")
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			return nil, nil, errors.New("malformed entry_points.txt: missing '='")
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if key == "" || val == "" {
			return nil, nil, errors.New("malformed entry_points.txt: empty key or value")
		}
		if !validEntryPointName(key) {
			return nil, nil, errors.New("malformed entry_points.txt: invalid entry point name")
		}
		modSpec := val
		if bracketIdx := strings.Index(val, "["); bracketIdx >= 0 {
			if !strings.HasSuffix(val, "]") {
				return nil, nil, errors.New("malformed entry_points.txt: unclosed extras bracket")
			}
			modSpec = strings.TrimSpace(val[:bracketIdx])
		}
		parts := strings.Split(modSpec, ":")
		if len(parts) > 2 {
			return nil, nil, errors.New("malformed entry_points.txt: multiple colons in target")
		}
		moduleName := strings.TrimSpace(parts[0])
		if !validPythonImportName(moduleName) {
			return nil, nil, errors.New("malformed entry_points.txt: invalid module name in target")
		}
		attr := ""
		if len(parts) == 2 {
			attr = strings.TrimSpace(parts[1])
			if attr == "" {
				return nil, nil, errors.New("malformed entry_points.txt: empty attribute in target")
			}
			for _, part := range strings.Split(attr, ".") {
				if !validPythonImportComponent(part) {
					return nil, nil, errors.New("malformed entry_points.txt: invalid attribute identifier")
				}
			}
		}
		details = append(details, EntryPoint{Group: currentSection, Name: key, Module: moduleName, Attr: attr})
		results = append(results, currentSection+": "+key+" = "+val)
	}
	sort.Strings(results)
	return results, details, nil
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

type rfc822Metadata struct {
	headers map[string][]string
	keys    []string
}

func parseRFC822Metadata(body []byte, limit int64) (*rfc822Metadata, error) {
	if int64(len(body)) > limit {
		return nil, errors.New("metadata exceeds byte limit")
	}
	if !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 {
		return nil, errors.New("metadata contains invalid bytes")
	}
	m := &rfc822Metadata{
		headers: make(map[string][]string),
	}
	raw := string(body)
	lines := strings.Split(raw, "\n")
	var currentKey string

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		// Header/body separator: the first blank line terminates headers
		if line == "" {
			break
		}
		// Line folding (continuation line)
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if currentKey == "" {
				return nil, errors.New("metadata starts with continuation line")
			}
			vals := m.headers[currentKey]
			if len(vals) > 0 {
				vals[len(vals)-1] += "\n" + strings.TrimSpace(line)
			}
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon <= 0 {
			return nil, errors.New("malformed metadata line without colon")
		}
		fieldName := line[:colon]
		for _, ch := range fieldName {
			if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
				return nil, errors.New("malformed metadata field name")
			}
		}
		fieldVal := strings.TrimSpace(line[colon+1:])
		key := strings.ToLower(fieldName)
		if _, exists := m.headers[key]; !exists {
			m.keys = append(m.keys, key)
		}
		m.headers[key] = append(m.headers[key], fieldVal)
		currentKey = key
	}
	return m, nil
}

func (m *rfc822Metadata) has(key string) bool {
	if m == nil {
		return false
	}
	_, ok := m.headers[strings.ToLower(key)]
	return ok
}

func (m *rfc822Metadata) first(key string) (string, bool) {
	if m == nil {
		return "", false
	}
	vals, ok := m.headers[strings.ToLower(key)]
	if !ok || len(vals) == 0 {
		return "", false
	}
	return vals[0], true
}

func (m *rfc822Metadata) all(key string) []string {
	if m == nil {
		return nil
	}
	return m.headers[strings.ToLower(key)]
}

func headerValues(body []byte, limit int64) map[string]string {
	m, err := parseRFC822Metadata(body, limit)
	if err != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(m.headers))
	for k, vals := range m.headers {
		if k == "requires-dist" || k == "import-name" || k == "import-namespace" || k == "entry-points" || k == "tag" {
			var b strings.Builder
			for _, v := range vals {
				b.WriteString(v)
				b.WriteByte('\n')
			}
			out[k] = b.String()
		} else if len(vals) > 0 {
			out[k] = vals[0]
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
