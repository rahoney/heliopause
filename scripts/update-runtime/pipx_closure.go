package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

type pipxPublishFact struct {
	Project          string `json:"project"`
	Status           string `json:"status"`
	ProvenanceSHA256 string `json:"provenance_sha256,omitempty"`
	Invocation       string `json:"invocation,omitempty"`
}

type pipxInputFact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// verifyPipxClosure composes existing providers for explicit frozen DATA. It
// does not import upstream Python, materialize a loader, install or adopt pins.
func verifyPipxClosure(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime verify-pipx-closure", flag.ContinueOnError)
	flags.SetOutput(out)
	input := flags.String("input", "", "absolute frozen directory with eleven project subdirectories: identity.json, metadata.json, wheel and supplied provenance.json")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*input) {
		return errors.New("pipx closure requires an explicit absolute directory")
	}
	supply, err := readPipxClosure(*input)
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		Schema  string                 `json:"schema"`
		Scope   string                 `json:"scope"`
		Python  string                 `json:"python"`
		Wheels  []pypi.PipxSupplyWheel `json:"wheels"`
		Files   []pypi.PipxSupplyFile  `json:"files"`
		Publish []pipxPublishFact      `json:"publish"`
		Inputs  []pipxInputFact        `json:"inputs"`
	}{"helox.pipx-closure-supply/v1", "STATIC_WHEEL_CLOSURE_AND_PUBLISH_PROOFS_ONLY", supply.closure.Python(), supply.closure.Wheels(), supply.closure.Files(), supply.proofs, supply.files})
}

type pipxClosureSupply struct {
	closure pypi.PipxSupplyClosure
	inputs  []pypi.UVWheelInput
	proofs  []pipxPublishFact
	files   []pipxInputFact
}

// readPipxClosure retains the same authenticated bytes for the candidate consumer.
func readPipxClosure(input string) (pipxClosureSupply, error) {
	if !filepath.IsAbs(input) {
		return pipxClosureSupply{}, errors.New("pipx closure requires an absolute directory")
	}
	info, err := os.Lstat(input)
	if err != nil || !info.IsDir() {
		return pipxClosureSupply{}, errors.New("pipx closure directory is unavailable or a symlink")
	}
	var packages []pypi.UVLockedPackage
	var inputs []pypi.UVWheelInput
	var pages []pypi.SimpleProject
	var proofs []pipxPublishFact
	var files []pipxInputFact
	for _, project := range []string{"argcomplete", "click", "filelock", "iniconfig", "packaging", "pipx", "platformdirs", "pluggy", "pygments", "pytest", "userpath"} {
		directory := filepath.Join(input, project)
		read := func(name string, limit int64) ([]byte, error) {
			body, err := readFrozenSupplyFile(directory, name, limit)
			if err == nil {
				files = append(files, pipxInputFact{project + "/" + name, hash(body)})
			}
			return body, err
		}
		identityBody, err := read("identity.json", 4096)
		if err != nil {
			return pipxClosureSupply{}, err
		}
		var version, filename, digest string
		fact := pipxPublishFact{Project: project}
		switch project {
		case "pipx":
			var id releaseprovenance.PipxPublishIdentity
			if json.Unmarshal(identityBody, &id) != nil {
				return pipxClosureSupply{}, errors.New("pipx main identity is invalid")
			}
			version, filename, digest = id.Version, id.WheelFilename(), id.WheelSHA256
			proof, err := read("provenance.json", 1<<20)
			if err != nil {
				return pipxClosureSupply{}, err
			}
			receipt, err := releaseprovenance.VerifyPipxPublish(context.Background(), proof, id)
			if err != nil {
				return pipxClosureSupply{}, err
			}
			fact.Status = "VERIFIED"
			fact.ProvenanceSHA256 = receipt.ProvenanceSHA256()
			fact.Invocation = receipt.Invocation()
		case "userpath", "pygments":
			version, digest, err = readPipxUnsignedIdentity(identityBody)
			if err != nil {
				return pipxClosureSupply{}, err
			}
			filename = project + "-" + version + "-py3-none-any.whl"
			if _, err := os.Lstat(filepath.Join(directory, "provenance.json")); !os.IsNotExist(err) {
				return pipxClosureSupply{}, errors.New("unsigned supply with a provided proof needs a reviewed publisher policy")
			}
			fact.Status = "NOT_PROVIDED"
		default:
			var id releaseprovenance.PipxDependencyPublishIdentity
			if json.Unmarshal(identityBody, &id) != nil || id.Project != project {
				return pipxClosureSupply{}, errors.New("pipx dependency identity differs from its frozen directory")
			}
			version, filename, digest = id.Version, id.WheelFilename(), id.WheelSHA256
			proof, err := read("provenance.json", 1<<20)
			if err != nil {
				return pipxClosureSupply{}, err
			}
			receipt, err := releaseprovenance.VerifyPipxDependencyPublish(context.Background(), proof, id)
			if err != nil {
				return pipxClosureSupply{}, err
			}
			fact.Status = "VERIFIED"
			fact.ProvenanceSHA256 = receipt.ProvenanceSHA256()
			fact.Invocation = receipt.Invocation()
		}
		metadata, err := read("metadata.json", 1<<20)
		if err != nil {
			return pipxClosureSupply{}, err
		}
		page, err := pypi.ParseUVReleaseMetadata(project, version, metadata)
		if err != nil {
			return pipxClosureSupply{}, err
		}
		wheel, err := read(filename, 8<<20)
		if err != nil {
			return pipxClosureSupply{}, err
		}
		if hash(wheel) != digest {
			return pipxClosureSupply{}, errors.New("pipx closure acquired wheel digest differs")
		}
		var selected pypi.SimpleFile
		matches := 0
		for _, file := range page.Files() {
			if file.Filename() == filename && file.SHA256() == digest && file.Size() == uint64(len(wheel)) && !file.Yanked() {
				selected = file
				matches++
			}
		}
		if matches != 1 {
			return pipxClosureSupply{}, errors.New("pipx closure wheel differs from independent release metadata")
		}
		packages = append(packages, pypi.UVLockedPackage{Name: project, Version: version, Wheels: []pypi.UVLockedWheel{{URL: selected.URL(), Filename: filename, SHA256: digest, Size: int64(len(wheel))}}})
		inputs = append(inputs, pypi.UVWheelInput{URL: selected.URL(), Reader: bytes.NewReader(wheel), Size: int64(len(wheel))})
		pages = append(pages, page)
		proofs = append(proofs, fact)
	}
	closure, err := pypi.BindPipxRuntimeSupply(packages, inputs, pages)
	if err != nil {
		return pipxClosureSupply{}, err
	}
	return pipxClosureSupply{closure, inputs, proofs, files}, nil
}

func readPipxUnsignedIdentity(body []byte) (string, string, error) {
	if len(body) == 0 || len(body) > 4096 {
		return "", "", errors.New("unsigned pipx runtime dependency identity exceeds bound")
	}
	var id struct {
		Version     string `json:"version"`
		WheelSHA256 string `json:"wheel_sha256"`
	}
	spans, err := jsonSpans(body)
	if err != nil || len(spans) != 3 {
		return "", "", errors.New("unsigned pipx runtime dependency identity is invalid")
	}
	for _, key := range []string{"", "/version", "/wheel_sha256"} {
		if _, ok := spans[key]; !ok {
			return "", "", errors.New("unsigned pipx runtime dependency identity fields are invalid")
		}
	}
	if json.Unmarshal(body, &id) != nil || !versionPattern.MatchString(id.Version) || !digestPattern.MatchString(id.WheelSHA256) {
		return "", "", errors.New("unsigned pipx runtime dependency identity is unsupported")
	}
	return id.Version, id.WheelSHA256, nil
}
