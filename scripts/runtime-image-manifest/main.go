// Command runtime-image-manifest emits the lock-owned runtime image contract.
// It deliberately does not build or republish upstream Docker Official Images.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const manifestSchema = "helox.runtime-image-manifest/v1"

var (
	digestPattern  = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
)

type runtimeLock struct {
	SchemaVersion int `json:"schema_version"`
	NodeImage     struct {
		Reference  string `json:"reference"`
		NPMVersion string `json:"npm_version"`
	} `json:"node_image"`
	PythonImage struct {
		Reference     string `json:"reference"`
		PythonVersion string `json:"python_version"`
		PipVersion    string `json:"pip_version"`
	} `json:"python_image"`
}

type runtimeImageManifest struct {
	Schema              string              `json:"schema"`
	CustomGHCRImage     bool                `json:"custom_ghcr_image"`
	CustomImageDecision string              `json:"custom_image_decision"`
	Provenance          provenanceContract  `json:"provenance"`
	Images              []runtimeImageEntry `json:"images"`
}

type provenanceContract struct {
	Required     bool   `json:"required"`
	Verification string `json:"verification"`
}

type runtimeImageEntry struct {
	Name       string `json:"name"`
	Reference  string `json:"reference"`
	Repository string `json:"repository"`
	Digest     string `json:"digest"`
	Version    string `json:"runtime_version"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "runtime-image-manifest: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("runtime-image-manifest", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	lockPath := flags.String("runtime-lock", "", "canonical runtime lock")
	outputPath := flags.String("output", "", "runtime image manifest output")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("positional arguments are not supported")
	}
	if *lockPath == "" || *outputPath == "" {
		return errors.New("runtime lock and output are required")
	}
	lockBody, err := os.ReadFile(*lockPath)
	if err != nil {
		return fmt.Errorf("read runtime lock: %w", err)
	}
	var lock runtimeLock
	if err := json.Unmarshal(lockBody, &lock); err != nil {
		return fmt.Errorf("decode runtime lock: %w", err)
	}
	manifest, err := buildManifest(lock)
	if err != nil {
		return err
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if _, err := os.Lstat(*outputPath); err == nil || !errors.Is(err, os.ErrNotExist) {
		return errors.New("output already exists or is unavailable")
	}
	file, err := os.OpenFile(*outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		_ = os.Remove(*outputPath)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(*outputPath)
		return err
	}
	return file.Close()
}

func buildManifest(lock runtimeLock) (runtimeImageManifest, error) {
	if lock.SchemaVersion != 1 {
		return runtimeImageManifest{}, errors.New("runtime lock schema is unsupported")
	}
	node, err := imageEntry("node", lock.NodeImage.Reference, "docker.io/library/node", lock.NodeImage.NPMVersion)
	if err != nil {
		return runtimeImageManifest{}, err
	}
	python, err := imageEntry("python", lock.PythonImage.Reference, "docker.io/library/python", lock.PythonImage.PythonVersion)
	if err != nil {
		return runtimeImageManifest{}, err
	}
	return runtimeImageManifest{
		Schema:              manifestSchema,
		CustomGHCRImage:     false,
		CustomImageDecision: "upstream immutable image is sufficient; no HAA-owned runtime configuration is required",
		Provenance: provenanceContract{
			Required:     true,
			Verification: "release candidate attestation must bind this manifest and each exact image reference",
		},
		Images: []runtimeImageEntry{node, python},
	}, nil
}

func imageEntry(name, reference, repository, version string) (runtimeImageEntry, error) {
	separator := strings.LastIndex(reference, "@")
	if separator <= len(name)+1 || reference == "" || version == "" || !versionPattern.MatchString(version) || !digestPattern.MatchString(reference[separator+1:]) {
		return runtimeImageEntry{}, errors.New("runtime image identity is invalid")
	}
	if !strings.HasPrefix(reference, name+":") {
		return runtimeImageEntry{}, errors.New("runtime image repository is not an official image")
	}
	return runtimeImageEntry{Name: name, Reference: reference, Repository: repository, Digest: reference[strings.LastIndex(reference, "@")+1:], Version: version}, nil
}
