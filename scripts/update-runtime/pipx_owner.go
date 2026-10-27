package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type pipxMaterializationOwner struct {
	Version      string            `json:"pipx_version"`
	Architecture string            `json:"architecture"`
	RuntimeImage string            `json:"runtime_image"`
	Inputs       []pipxInputFact   `json:"inputs"`
	Tool         pipxAssetIdentity `json:"tool"`
	Notices      pipxAssetIdentity `json:"notices"`
}

type pipxAssetIdentity struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

func readPipxMaterializationOwner(root string) (pipxMaterializationOwner, error) {
	var pin pipxMaterializationOwner
	body, err := readFrozenSupplyFile(root, runtimeFile, 64<<10)
	if err != nil {
		return pin, err
	}
	selected, err := selectedPipxOwner(body)
	if err != nil {
		return pin, err
	}
	decoder := json.NewDecoder(bytes.NewReader(selected))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&pin) != nil || decoder.Decode(new(any)) != io.EOF || !validPipxOwner(pin) {
		return pipxMaterializationOwner{}, errors.New("pipx selected runtime owner is invalid")
	}
	return pin, nil
}

func selectedPipxOwner(body []byte) ([]byte, error) {
	fields, err := jsonSpans(body)
	if err != nil {
		return nil, err
	}
	expected := map[string]bool{}
	for _, field := range []string{"", "/pipx_version", "/architecture", "/runtime_image", "/inputs", "/tool", "/tool/path", "/tool/sha256", "/tool/size", "/notices", "/notices/path", "/notices/sha256", "/notices/size"} {
		expected["/pipx_tool"+field] = true
	}
	for i := 0; i < 27; i++ {
		for _, suffix := range []string{"", "/path", "/sha256"} {
			expected["/pipx_tool/inputs/"+strconv.Itoa(i)+suffix] = true
		}
	}
	count := 0
	for name := range fields {
		if name == "/pipx_tool" || strings.HasPrefix(name, "/pipx_tool/") {
			if !expected[name] {
				return nil, errors.New("pipx runtime owner has unknown or noncanonical fields")
			}
			count++
		}
	}
	schema, ok := fields["/schema_version"]
	var version int
	if !ok || json.Unmarshal(body[schema.start:schema.end], &version) != nil || version != 5 || count != len(expected) {
		return nil, errors.New("pipx runtime owner schema or fields differ")
	}
	owner := fields["/pipx_tool"]
	return body[owner.start:owner.end], nil
}

func validPipxOwner(pin pipxMaterializationOwner) bool {
	if !versionPattern.MatchString(pin.Version) || pin.Architecture != "amd64" || pin.RuntimeImage != "python_image" || len(pin.Inputs) != 27 || pin.Tool.Path != "pipx-runtime.zip" || !digestPattern.MatchString(pin.Tool.SHA256) || pin.Tool.Size <= 0 || pin.Tool.Size > 8<<20 || pin.Notices.Path != "tools/pipx/NOTICE" || !digestPattern.MatchString(pin.Notices.SHA256) || pin.Notices.Size <= 0 || pin.Notices.Size > 64<<10 {
		return false
	}
	seen := map[string]bool{}
	counts := map[string]int{}
	for _, f := range pin.Inputs {
		project, name, ok := strings.Cut(f.Path, "/")
		if !ok || path.Clean(f.Path) != f.Path || strings.Contains(name, "/") || len(name) > 200 || seen[f.Path] || !digestPattern.MatchString(f.SHA256) {
			return false
		}
		switch project {
		case "argcomplete", "click", "filelock", "packaging", "pipx", "platformdirs", "userpath":
		default:
			return false
		}
		if name != "identity.json" && name != "metadata.json" && (name != "provenance.json" || project == "userpath") && (!strings.HasPrefix(name, project+"-") || !strings.HasSuffix(name, "-py3-none-any.whl")) {
			return false
		}
		seen[f.Path] = true
		counts[project]++
	}
	for _, project := range []string{"argcomplete", "click", "filelock", "packaging", "pipx", "platformdirs", "userpath"} {
		want := 4
		if project == "userpath" {
			want = 3
		}
		if counts[project] != want || !seen[project+"/identity.json"] || !seen[project+"/metadata.json"] || project != "userpath" && !seen[project+"/provenance.json"] {
			return false
		}
	}
	return seen["pipx/pipx-"+pin.Version+"-py3-none-any.whl"]
}

func bindPipxOwner(pin pipxMaterializationOwner, supply pipxClosureSupply, body, notices []byte) error {
	actual := append([]pipxInputFact(nil), supply.files...)
	expected := append([]pipxInputFact(nil), pin.Inputs...)
	sort.Slice(actual, func(i, j int) bool { return actual[i].Path < actual[j].Path })
	sort.Slice(expected, func(i, j int) bool { return expected[i].Path < expected[j].Path })
	if !validPipxOwner(pin) || !reflect.DeepEqual(actual, expected) || hash(body) != pin.Tool.SHA256 || int64(len(body)) != pin.Tool.Size || hash(notices) != pin.Notices.SHA256 || int64(len(notices)) != pin.Notices.Size {
		return errors.New("pipx authenticated supply differs from selected runtime owner")
	}
	for _, wheel := range supply.closure.Wheels() {
		if wheel.Project == "pipx" && wheel.Version != pin.Version {
			return errors.New("pipx selected version differs")
		}
	}
	return nil
}
