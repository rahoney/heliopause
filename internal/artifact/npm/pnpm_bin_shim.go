package npm

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path"
	"strings"
)

// Reviewed code-format identity, derived independently from the pinned upstream
// shell renderer. It is neither a runtime version nor artifact authorization.
const pnpmNodeBinCodeDigest = "64d69cfde5def2f7644e5ca92e1e669eaeb87bcbd30355f2c2cd9ac3b20bf2d7"

var errPNPMNodeBinShim = errors.New("pnpm Node bin shim code or path data differs from the reviewed format")

// ParsePNPMNodeBinShim checks the complete Node/no-extra-runtime-args shim code
// format. The expected target must come from the caller's archive/bin binding,
// never from the shim's own marker. Returned module paths are unverified data;
// their actual directories and full frozen graph still require validation.
func ParsePNPMNodeBinShim(body []byte, shim, expectedTarget string) ([]string, error) {
	if len(body) == 0 || len(body) > 64<<10 || !safePNPMBinPath(shim) || !safePNPMBinPath(expectedTarget) || path.Base(path.Dir(shim)) != ".bin" {
		return nil, errPNPMNodeBinShim
	}
	target := pnpmBinRelativePath(path.Dir(shim), expectedTarget)
	if target == "" || len(target) > 4096 {
		return nil, errPNPMNodeBinShim
	}
	code := string(body)
	const blockStart = "if [ -z \"$NODE_PATH\" ]; then\n"
	var modulePaths []string
	if strings.Count(code, blockStart) > 1 {
		return nil, errPNPMNodeBinShim
	}
	if start := strings.Index(code, blockStart); start >= 0 {
		end := strings.Index(code[start:], "\nfi\n")
		if end < 0 {
			return nil, errPNPMNodeBinShim
		}
		end += start + len("\nfi\n")
		lines := strings.Split(strings.TrimSuffix(code[start:end], "\n"), "\n")
		const exportPrefix = "  export NODE_PATH=\""
		if len(lines) != 5 || lines[2] != "else" || lines[4] != "fi" || !strings.HasPrefix(lines[1], exportPrefix) || !strings.HasSuffix(lines[1], "\"") {
			return nil, errPNPMNodeBinShim
		}
		value := strings.TrimSuffix(strings.TrimPrefix(lines[1], exportPrefix), "\"")
		if lines[3] != exportPrefix+value+":$NODE_PATH\"" {
			return nil, errPNPMNodeBinShim
		}
		entries := strings.Split(value, ":")
		if len(entries) > 32 {
			return nil, errPNPMNodeBinShim
		}
		seen := map[string]bool{}
		for _, entry := range entries {
			const prefix = "$basedir_abs/"
			if !strings.HasPrefix(entry, prefix) {
				return nil, errPNPMNodeBinShim
			}
			relative := strings.TrimPrefix(entry, prefix)
			if !safePNPMBinCharacters(relative) || path.Clean(relative) != relative || path.IsAbs(relative) {
				return nil, errPNPMNodeBinShim
			}
			resolved := path.Join(path.Dir(shim), relative)
			if !safePNPMBinPath(resolved) || path.Base(resolved) != "node_modules" || seen[resolved] {
				return nil, errPNPMNodeBinShim
			}
			seen[resolved] = true
			modulePaths = append(modulePaths, resolved)
		}
		code = code[:start] + code[end:]
	}
	if strings.Count(code, target) != 7 {
		return nil, errPNPMNodeBinShim
	}
	code = strings.ReplaceAll(code, target, "{{HAA_BIN_TARGET}}")
	digest := sha256.Sum256([]byte(code))
	if hex.EncodeToString(digest[:]) != pnpmNodeBinCodeDigest {
		return nil, errPNPMNodeBinShim
	}
	return modulePaths, nil
}

func safePNPMBinPath(value string) bool {
	return validInventoryPath(value) && safePNPMBinCharacters(value) &&
		(value == "node_modules" || strings.HasPrefix(value, "node_modules/"))
}

func safePNPMBinCharacters(value string) bool {
	if value == "" || len(value) > 4096 {
		return false
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("@+_./-=", char) {
			continue
		}
		return false
	}
	return true
}

// Guest slash semantics are independent of the Host OS/case rules.
func pnpmBinRelativePath(base, target string) string {
	from, to := strings.Split(base, "/"), strings.Split(target, "/")
	common := 0
	for common < min(len(from), len(to)) && from[common] == to[common] {
		common++
	}
	return strings.Repeat("../", len(from)-common) + strings.Join(to[common:], "/")
}
