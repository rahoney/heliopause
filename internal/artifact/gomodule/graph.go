package gomodule

import (
	"bufio"
	"bytes"
	"errors"
	goversion "go/version"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

type graphEdge struct{ from, to string }

// normalizeProjectGraph binds the unversioned main module and synthetic Go
// vertices to go.mod. Public edges are minimum requirements: their endpoints
// map to the selected versions from download all, never to invented artifacts.
// The complete raw requirement graph remains separately digest-bound.
func normalizeProjectGraph(body []byte, records []DownloadRecord, goMod []byte) ([]graphEdge, error) {
	file, err := parseProjectMod(goMod)
	if err != nil || len(body) > maxDownloadOutput || len(records) > 4096 {
		return nil, errors.New("go project graph is invalid or exceeds bound")
	}
	if len(records) == 0 {
		free, err := ProjectDependencyFree(goMod)
		if err != nil || !free {
			return nil, errors.New("empty Go graph differs from project requirements")
		}
	}
	selected := map[string]DownloadRecord{}
	for _, record := range records {
		if !validModuleVersion(record.Path, record.Version) || selected[record.Path].Path != "" {
			return nil, errors.New("go project selected modules are invalid or ambiguous")
		}
		if _, err := h1Digest(record.Sum); err != nil {
			return nil, err
		}
		if _, err := h1Digest(record.GoModSum); err != nil {
			return nil, err
		}
		selected[record.Path] = record
	}
	main := file.Module.Mod.Path
	mainGo, mainToolchain := "", ""
	if file.Go != nil {
		mainGo = "go" + file.Go.Version
	}
	if file.Toolchain != nil {
		mainToolchain = file.Toolchain.Name
	}
	requirements := map[string]string{}
	for _, requirement := range file.Require {
		if _, duplicate := requirements[requirement.Mod.Path]; duplicate {
			return nil, errors.New("go project requirements contain duplicate module paths")
		}
		requirements[requirement.Mod.Path] = requirement.Mod.Version
	}
	mainSeen := map[string]bool{}
	adjacent := map[string][]string{}
	edges := map[graphEdge]bool{}
	rawSeen := map[string]bool{}
	goSeen, toolchainSeen := false, false
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 2 || rawSeen[strings.Join(parts, " ")] || len(rawSeen) >= 32768 {
			return nil, errors.New("go module graph edge is invalid or duplicate")
		}
		rawSeen[strings.Join(parts, " ")] = true
		from, fromErr := projectVertex(parts[0], main, mainGo, mainToolchain, selected)
		to, toErr := projectVertex(parts[1], main, mainGo, mainToolchain, selected)
		if fromErr != nil || toErr != nil || to.kind == "main" {
			return nil, errors.New("go module graph contains an unbound vertex")
		}
		switch from.kind {
		case "main":
			switch to.kind {
			case "public":
				if requirements[to.path] != to.version {
					return nil, errors.New("go module graph main edge differs from go.mod")
				}
				mainSeen[to.path] = true
				adjacent[main] = append(adjacent[main], to.key)
			case "go":
				if "go"+to.version != mainGo {
					return nil, errors.New("go graph main Go version differs from go.mod")
				}
				goSeen = true
			case "toolchain":
				if to.version != mainToolchain {
					return nil, errors.New("go graph main toolchain differs from go.mod")
				}
				toolchainSeen = true
			}
		case "public":
			if to.kind == "public" {
				adjacent[from.key] = append(adjacent[from.key], to.key)
				if from.key != to.key {
					edges[graphEdge{from.key, to.key}] = true
				}
			} else if to.kind != "go" {
				return nil, errors.New("go graph public module has an invalid synthetic edge")
			}
		case "go":
			if to.kind != "toolchain" || "go"+from.version != to.version {
				return nil, errors.New("go graph synthetic toolchain relation is invalid")
			}
		default:
			return nil, errors.New("go graph toolchain cannot own dependency edges")
		}
	}
	if scanner.Err() != nil || len(mainSeen) != len(requirements) || (mainGo != "" && !goSeen) || (mainToolchain != "" && mainToolchain != "default" && !toolchainSeen) {
		return nil, errors.New("go project graph is incomplete")
	}
	reachable := map[string]bool{main: true}
	queue := []string{main}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adjacent[current] {
			if !reachable[next] {
				reachable[next] = true
				queue = append(queue, next)
			}
		}
	}
	if len(reachable) != len(selected)+1 {
		return nil, errors.New("go graph selected module is not reachable from the recorded main module")
	}
	result := make([]graphEdge, 0, len(edges))
	for edge := range edges {
		result = append(result, edge)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].from+" "+result[i].to < result[j].from+" "+result[j].to
	})
	return result, nil
}

type graphVertex struct{ kind, path, version, key string }

func projectVertex(value, main, mainGo, mainToolchain string, selected map[string]DownloadRecord) (graphVertex, error) {
	if value == main {
		return graphVertex{kind: "main", key: main}, nil
	}
	path, version, ok := strings.Cut(value, "@")
	if !ok {
		return graphVertex{}, errors.New("unrecorded main module")
	}
	if path == "go" {
		v := "go" + version
		if mainGo == "" || !goversion.IsValid(v) || goversion.Compare(v, mainGo) > 0 {
			return graphVertex{}, errors.New("unbound Go version")
		}
		return graphVertex{kind: "go", version: version}, nil
	}
	if path == "toolchain" {
		limit := mainGo
		if mainToolchain != "" && mainToolchain != "default" {
			limit = mainToolchain
		}
		if limit == "" || !goversion.IsValid(version) || goversion.Compare(version, limit) > 0 {
			return graphVertex{}, errors.New("unbound toolchain version")
		}
		return graphVertex{kind: "toolchain", version: version}, nil
	}
	record, known := selected[path]
	if !known || !validModuleVersion(path, version) || semver.Compare(version, record.Version) > 0 {
		return graphVertex{}, errors.New("module requirement has no satisfying selected version")
	}
	return graphVertex{kind: "public", path: path, version: version, key: recordKey(path, record.Version)}, nil
}
