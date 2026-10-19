package gomodule

import (
	"bytes"
	"errors"
	"io"
	"sort"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/mod/sumdb/dirhash"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const MaxRequiredModControls = 16

// RequiredModReference is frozen graph context, not another selected module or
// permission to acquire an archive. Its old-version SumDB proof is independent.
type RequiredModReference struct{ Version, Sum string }

// RequiredModControl carries exact same-path control bytes inside the selected
// module's acquired subject. Callers receive copies from Bundle.RequiredMods.
type RequiredModControl struct {
	Version string `json:"version"`
	Body    []byte `json:"body"`
}

func parseModuleIntegrity(value string) (archive, mod string, required []RequiredModReference, err error) {
	// Preserve the existing Domain bound rather than widening it for context.
	if len(value) > 1024 || strings.ContainsAny(value, "\r\n\t ") {
		return "", "", nil, errors.New("go module integrity exceeds its canonical bound")
	}
	pair, context, hasContext := strings.Cut(value, ";required-mods=")
	a, m, ok := strings.Cut(pair, ";go.mod=")
	if !ok || !strings.HasPrefix(a, "h1=") {
		return "", "", nil, errors.New("go module integrity pair is invalid")
	}
	a = strings.TrimPrefix(a, "h1=")
	if _, err := h1Digest(a); err != nil {
		return "", "", nil, err
	}
	if _, err := h1Digest(m); err != nil {
		return "", "", nil, err
	}
	if hasContext {
		entries := strings.Split(context, ",")
		if context == "" || len(entries) > MaxRequiredModControls {
			return "", "", nil, errors.New("required Go module controls exceed bound")
		}
		previous := ""
		for _, entry := range entries {
			version, sum, ok := strings.Cut(entry, ":")
			if !ok || !semver.IsValid(version) || module.CanonicalVersion(version) != version || version <= previous {
				return "", "", nil, errors.New("required Go module versions are not canonical")
			}
			if _, err := h1Digest(sum); err != nil {
				return "", "", nil, err
			}
			required = append(required, RequiredModReference{version, sum})
			previous = version
		}
	}
	return a, m, required, nil
}

func RequiredModReferences(value string) ([]RequiredModReference, error) {
	_, _, refs, err := parseModuleIntegrity(value)
	return refs, err
}

func ValidateRequiredModReferences(identity domain.ResolvedArtifactIdentity, refs []RequiredModReference) error {
	if identity.Source() != Source() || identity.Variant() != "module" || !validModuleVersion(identity.Name(), identity.Version()) || len(refs) > MaxRequiredModControls {
		return errors.New("required Go module identity is invalid")
	}
	previous := ""
	for _, ref := range refs {
		if !validModuleVersion(identity.Name(), ref.Version) || semver.Compare(ref.Version, identity.Version()) >= 0 || ref.Version <= previous {
			return errors.New("required Go control is not a distinct lower same-path version")
		}
		if _, err := h1Digest(ref.Sum); err != nil {
			return err
		}
		previous = ref.Version
	}
	return nil
}

// HashRequiredMods checks frozen membership and h1 agreement only. It never
// substitutes a project declaration for authenticated old-version SumDB proof.
func HashRequiredMods(bundle Bundle, identity domain.ResolvedArtifactIdentity, integrity string) ([]RequiredModReference, error) {
	var refs []RequiredModReference
	var err error
	if integrity != "" {
		refs, err = RequiredModReferences(integrity)
		if err != nil {
			return nil, err
		}
	}
	if err := ValidateRequiredModReferences(identity, refs); err != nil {
		return nil, err
	}
	if len(refs) != len(bundle.required) {
		return nil, errors.New("required Go control coverage differs from frozen declaration")
	}
	for i, control := range bundle.required {
		if control.Version != refs[i].Version || len(control.Body) == 0 || len(control.Body) > MaxModBytes {
			return nil, errors.New("required Go control identity or bound differs")
		}
		sum, err := dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(control.Body)), nil })
		if err != nil || sum != refs[i].Sum {
			return nil, errors.New("required Go control differs from declared h1")
		}
	}
	return refs, nil
}

// projectModuleIntegrities derives context only from validated raw graph
// vertices. Stale, unrelated go.sum lines never become acquisition requests.
func projectModuleIntegrities(records []DownloadRecord, graph, mod, sums []byte) (map[string]string, error) {
	if _, err := normalizeProjectGraph(graph, records, mod); err != nil {
		return nil, err
	}
	if len(records) > 0 {
		if err := ValidateProjectSums(sums, records); err != nil {
			return nil, err
		}
	}
	selected := map[string]DownloadRecord{}
	for _, record := range records {
		selected[record.Path] = record
	}
	needed := map[string]map[string]bool{}
	for _, line := range strings.Split(string(graph), "\n") {
		for _, vertex := range strings.Fields(line) {
			name, version, ok := strings.Cut(vertex, "@")
			record, public := selected[name]
			if !ok || !public || version == record.Version {
				continue
			}
			if !validModuleVersion(name, version) || semver.Compare(version, record.Version) >= 0 {
				return nil, errors.New("required Go graph version is invalid")
			}
			if needed[name] == nil {
				needed[name] = map[string]bool{}
			}
			needed[name][version] = true
			if len(needed[name]) > MaxRequiredModControls {
				return nil, errors.New("required Go graph context exceeds bound")
			}
		}
	}
	bySum := map[string]string{}
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 3 {
			return nil, errors.New("required Go checksum record is invalid")
		}
		bySum[fields[0]+" "+fields[1]] = fields[2]
	}
	result := map[string]string{}
	for _, record := range records {
		value := "h1=" + record.Sum + ";go.mod=" + record.GoModSum
		versions := make([]string, 0, len(needed[record.Path]))
		for version := range needed[record.Path] {
			versions = append(versions, version)
		}
		sort.Strings(versions)
		var entries []string
		for _, version := range versions {
			sum := bySum[record.Path+" "+version+"/go.mod"]
			if _, err := h1Digest(sum); err != nil {
				return nil, errors.New("required Go control checksum is missing or invalid")
			}
			entries = append(entries, version+":"+sum)
		}
		if len(entries) > 0 {
			value += ";required-mods=" + strings.Join(entries, ",")
		}
		if _, _, _, err := parseModuleIntegrity(value); err != nil {
			return nil, err
		}
		result[record.Path] = value
	}
	return result, nil
}
