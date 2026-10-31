package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type pipxPinnedDownload struct {
	Path    string
	Archive archive
}
type pipxFetchPlan struct {
	Owner     pipxMaterializationOwner
	Documents []pipxFrozenDocument
	Wheels    []pipxPinnedDownload
}

func planPipxFetch(root string) (pipxFetchPlan, error) {
	owner, err := readPipxMaterializationOwner(root)
	if err != nil {
		return pipxFetchPlan{}, err
	}
	plan := pipxFetchPlan{Owner: owner}
	documents := map[string][]byte{}
	// No network request precedes verification of every canonical document.
	for _, fact := range owner.Inputs {
		if strings.HasSuffix(fact.Path, ".whl") {
			continue
		}
		body, err := readFrozenSupplyFile(root, "tools/pipx/supply/"+fact.Path, 1<<20)
		if err != nil || hash(body) != fact.SHA256 {
			return pipxFetchPlan{}, errors.New("pipx canonical input metadata differs from selected owner")
		}
		documents[fact.Path] = body
		plan.Documents = append(plan.Documents, pipxFrozenDocument{Path: fact.Path, Body: body})
	}
	for _, fact := range owner.Inputs {
		if !strings.HasSuffix(fact.Path, ".whl") {
			continue
		}
		project, filename, _ := strings.Cut(fact.Path, "/")
		var metadata struct {
			URLs []struct {
				Filename string `json:"filename"`
				URL      string `json:"url"`
				Size     int64  `json:"size"`
				Type     string `json:"packagetype"`
				Yanked   bool   `json:"yanked"`
				Digests  struct {
					SHA256 string `json:"sha256"`
				} `json:"digests"`
			} `json:"urls"`
		}
		if err := json.Unmarshal(documents[project+"/metadata.json"], &metadata); err != nil {
			return pipxFetchPlan{}, err
		}
		matches := 0
		var selected archive
		for _, row := range metadata.URLs {
			if row.Filename != filename {
				continue
			}
			matches++
			parsed, err := url.Parse(row.URL)
			if err != nil || parsed.Scheme != "https" || parsed.Host != "files.pythonhosted.org" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawPath != "" || !strings.HasPrefix(parsed.Path, "/packages/") || path.Clean(parsed.Path) != parsed.Path || path.Base(parsed.Path) != filename || row.Size <= 0 || row.Size > 8<<20 || row.Digests.SHA256 != fact.SHA256 || row.Type != "bdist_wheel" || row.Yanked {
				return pipxFetchPlan{}, errors.New("pipx frozen wheel metadata or URL differs from allowed selected input")
			}
			selected = archive{URL: row.URL, Size: row.Size, SHA256: fact.SHA256}
		}
		if matches != 1 {
			return pipxFetchPlan{}, errors.New("pipx frozen wheel selection is missing or ambiguous")
		}
		plan.Wheels = append(plan.Wheels, pipxPinnedDownload{Path: fact.Path, Archive: selected})
	}
	if len(plan.Documents) != 31 || len(plan.Wheels) != 11 {
		return pipxFetchPlan{}, errors.New("pipx frozen input census differs")
	}
	return plan, nil
}

// This prepares only selected bytes. Existing materialize-pipx still performs
// offline whole-wheel, provided-proof, closure, ZIP and NOTICE verification.
func fetchPipxSupply(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime fetch-pipx-supply", flag.ContinueOnError)
	flags.SetOutput(out)
	root := flags.String("root", "", "absolute canonical source directory")
	output := flags.String("output", "", "absolute new external frozen input directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*root) || filepath.Clean(*root) != *root || !filepath.IsAbs(*output) || filepath.Clean(*output) != *output {
		return errors.New("pipx input fetch requires canonical source and new external directory")
	}
	canonical, err := filepath.EvalSymlinks(*root)
	if err != nil || canonical != *root {
		return errors.New("pipx source directory is unavailable or aliased")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(*output))
	if err != nil || parent != filepath.Dir(*output) {
		return errors.New("pipx output parent is unavailable or aliased")
	}
	relative, err := filepath.Rel(canonical, *output)
	if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("pipx fetched inputs must be outside source")
	}
	plan, err := planPipxFetch(canonical)
	if err != nil {
		return err
	}
	if err := executePipxFetch(plan, *output, downloadArchive); err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		Scope  string `json:"scope"`
		Inputs int    `json:"inputs"`
		Wheels int    `json:"wheels"`
	}{"FROZEN_INPUTS_ONLY", len(plan.Owner.Inputs), len(plan.Wheels)})
}

func executePipxFetch(plan pipxFetchPlan, output string, download func(string, archive) error) error {
	if len(plan.Documents) != 31 || len(plan.Wheels) != 11 || !validPipxOwner(plan.Owner) || download == nil {
		return errors.New("pipx input fetch plan is incomplete")
	}
	if err := os.Mkdir(output, 0700); err != nil {
		return err
	}
	directories := map[string]bool{}
	for _, document := range plan.Documents {
		project, _, _ := strings.Cut(document.Path, "/")
		if !directories[project] {
			if err := os.Mkdir(filepath.Join(output, project), 0700); err != nil {
				return err
			}
			directories[project] = true
		}
		if err := writePipxFrozenInput(filepath.Join(output, document.Path), document.Body); err != nil {
			return err
		}
	}
	for _, wheel := range plan.Wheels {
		destination := filepath.Join(output, wheel.Path)
		if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
			return errors.New("pipx wheel destination exists or is uncertain")
		}
		if err := download(destination, wheel.Archive); err != nil {
			return fmt.Errorf("fetch pinned pipx input %s: %w", wheel.Path, err)
		}
	}
	for _, fact := range plan.Owner.Inputs {
		limit := int64(1 << 20)
		if strings.HasSuffix(fact.Path, ".whl") {
			limit = 8 << 20
		}
		body, err := readFrozenSupplyFile(output, fact.Path, limit)
		if err != nil || hash(body) != fact.SHA256 {
			return errors.New("pipx fetched input set is incomplete or changed")
		}
	}
	return nil
}
