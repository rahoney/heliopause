package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

func applyPlan(p plan, state string, out io.Writer, afterWrite func(int) error) (err error) {
	if err := compareSnapshot(p); err != nil {
		return err
	}
	expected, err := renderChanges(p, state)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(expected, p.Changes) {
		return errors.New("frozen patch differs from declared component/metadata/generator")
	}
	journalPath := filepath.Join(state, "journal.json")
	j := journal{PlanID: p.ID, Phase: "PREPARED", Attempts: []string{}}
	existing := false
	if err := loadJSON(journalPath, &j); err == nil {
		existing = true
		if j.PlanID != p.ID {
			return errors.New("journal plan mismatch")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	defer func() {
		if err != nil {
			j.Phase = "INTERRUPTED"
			if j.FirstFailure == "" {
				j.FirstFailure = err.Error()
			}
			j.Attempts = append(j.Attempts, err.Error())
			if e := saveJSON(journalPath, j); e != nil {
				err = fmt.Errorf("%w; journal persistence also failed: %v", err, e)
			}
		}
	}()
	if err := saveJSON(journalPath, j); err != nil {
		return err
	}
	candidate := filepath.Join(state, "candidate")
	if _, e := os.Lstat(candidate); errors.Is(e, os.ErrNotExist) {
		if _, err := command(p.Root, nil, "git", "worktree", "add", "--detach", candidate, p.Head); err != nil {
			return err
		}
		for _, f := range p.Inputs {
			if err := os.Chmod(filepath.Join(candidate, f.Path), os.FileMode(f.Mode)); err != nil {
				return err
			}
		}
	} else if e != nil {
		return e
	} else if !existing {
		return errors.New("candidate preexists without this plan's journal")
	}
	if err := checkCandidate(p, candidate, false); err != nil {
		return err
	}
	j.Phase = "APPLYING"
	if err := saveJSON(journalPath, j); err != nil {
		return err
	}
	modes := map[string]os.FileMode{}
	for _, f := range p.Inputs {
		modes[f.Path] = os.FileMode(f.Mode)
	}
	for i, c := range p.Changes {
		body, _, err := readRegular(candidate, c.Path)
		if err != nil {
			return err
		}
		if bytes.Equal(body, c.After) {
			continue
		}
		if !bytes.Equal(body, c.Before) {
			return fmt.Errorf("concurrent change preserved: %s", c.Path)
		}
		if err := atomicWrite(filepath.Join(candidate, c.Path), c.After, modes[c.Path]); err != nil {
			return err
		}
		if afterWrite != nil {
			if err := afterWrite(i); err != nil {
				return err
			}
		}
	}
	if err := checkCandidate(p, candidate, true); err != nil {
		return err
	}
	if err := compareSnapshot(p); err != nil {
		return err
	}
	j.Phase = "APPLIED"
	if err := saveJSON(journalPath, j); err != nil {
		return err
	}
	fmt.Fprintf(out, "applied %s: %s; %d declared file changes; source checkout unchanged\n", p.ID, candidate, len(p.Changes))
	return nil
}

func checkCandidate(p plan, candidate string, complete bool) error {
	head, inputs, err := snapshot(candidate, false)
	if err != nil {
		return err
	}
	if head != p.Head || len(inputs) != len(p.Inputs) {
		return errors.New("candidate HEAD/tracked inventory changed")
	}
	if _, err := command(candidate, nil, "git", "diff", "--cached", "--quiet"); err != nil {
		return errors.New("candidate index changed; preserve and review it")
	}
	patches := map[string]change{}
	for _, c := range p.Changes {
		if _, ok := patches[c.Path]; ok {
			return errors.New("duplicate patch path")
		}
		patches[c.Path] = c
	}
	for i, actual := range inputs {
		before := p.Inputs[i]
		if actual.Path != before.Path || actual.Mode != before.Mode {
			return errors.New("candidate path/mode changed")
		}
		expected := before.SHA256
		if c, ok := patches[actual.Path]; ok {
			if hash(c.Before) != before.SHA256 {
				return errors.New("patch baseline mismatch")
			}
			after := hash(c.After)
			if actual.SHA256 == after {
				continue
			}
			if complete {
				return fmt.Errorf("candidate partial update: %s", actual.Path)
			}
		}
		if actual.SHA256 != expected {
			return fmt.Errorf("candidate external bytes preserved: %s", actual.Path)
		}
	}
	return nil
}

func verifyPlan(p plan, state, goPath string, out io.Writer) error {
	if p.Component == "uv" && p.Version != p.Current {
		if err := verifyFrozenUVSupply(p.Version, p.Metadata, state); err != nil {
			return err
		}
	}
	var j journal
	if err := loadJSON(filepath.Join(state, "journal.json"), &j); err != nil {
		return err
	}
	if j.PlanID != p.ID || j.Phase != "APPLIED" {
		return errors.New("apply must complete before verify")
	}
	if err := compareSnapshot(p); err != nil {
		return err
	}
	candidate := filepath.Join(state, "candidate")
	if err := checkCandidate(p, candidate, true); err != nil {
		return err
	}
	var err error
	if p.Component == "go" && p.Version != p.Current {
		if goPath != "" {
			return errors.New("new Go checks use the frozen official SDK prepared inside state; ambient --go is refused")
		}
		goPath, err = prepareSDK(p, state)
		if err != nil {
			return err
		}
	} else if goPath == "" {
		goPath, err = exec.LookPath("go")
		if err != nil {
			return err
		}
	}
	if !filepath.IsAbs(goPath) {
		return errors.New("target Go executable must be absolute")
	}
	goPath, err = filepath.EvalSymlinks(goPath)
	if err != nil {
		return err
	}
	if err := checkGo(goPath, candidate); err != nil {
		return err
	}
	env, err := verificationEnvironment(goPath, state)
	if err != nil {
		return err
	}
	// Do not let an editable plan reduce canonical checks or claim runtime qualification.
	checks := []string{"bootstrap", "quick", "security", "workflow", "release-gate", "docs", "qualification-freshness"}
	if !reflect.DeepEqual(p.Checks, checks) {
		return errors.New("verification profile inventory changed")
	}
	type result struct {
		Profile   string `json:"profile"`
		Status    string `json:"status"`
		LogSHA256 string `json:"log_sha256"`
	}
	results := []result{}
	attempt, err := os.MkdirTemp(state, "verification-")
	if err != nil {
		return err
	}
	checkedSource, err := verificationSource(p, candidate, attempt)
	if err != nil {
		return err
	}
	writeResult := func(status string) error {
		receipt := struct {
			PlanID            string   `json:"plan_id"`
			Attempt           string   `json:"attempt"`
			Configuration     string   `json:"configuration"`
			Go                string   `json:"go"`
			Source            string   `json:"checked_source"`
			Results           []result `json:"results"`
			Required          []string `json:"required_qualification"`
			MigrationComplete bool     `json:"migration_complete"`
		}{p.ID, filepath.Base(attempt), status, goPath, checkedSource, results, p.RequiredQualification, false}
		if err := saveJSON(filepath.Join(attempt, "result.json"), receipt); err != nil {
			return err
		}
		return saveJSON(filepath.Join(state, "verification.json"), receipt)
	}
	if err := writeResult("IN_PROGRESS"); err != nil {
		return err
	}
	for _, profile := range checks {
		fmt.Fprintf(out, "verify: %s\n", profile)
		log, checkErr := command(checkedSource, env, goPath, "run", "./scripts/check", profile)
		status := "PASS"
		if checkErr != nil {
			status = "FAIL"
			log = append(log, []byte(checkErr.Error()+"\n")...)
		}
		if err := atomicWrite(filepath.Join(attempt, "verify-"+profile+".log"), log, 0o600); err != nil {
			return err
		}
		results = append(results, result{profile, status, hash(log)})
		configuration := "IN_PROGRESS"
		if checkErr != nil {
			configuration = "FAIL"
		}
		if err := writeResult(configuration); err != nil {
			return err
		}
		if checkErr != nil {
			return fmt.Errorf("configuration verification failed at %s; first failure preserved in state: %w", profile, checkErr)
		}
		if err := checkCandidate(p, candidate, true); err != nil {
			return err
		}
		if err := checkCandidate(p, checkedSource, true); err != nil {
			return err
		}
	}
	if err := compareSnapshot(p); err != nil {
		return err
	}
	if err := writeResult("PASS"); err != nil {
		return err
	}
	fmt.Fprintf(out, "configuration PASS; runtime qualification/activation remains separate: %s\n", strings.Join(p.RequiredQualification, ", "))
	return nil
}

// The selected Go SDK discovers .git directories, while linked worktrees use
// .git files. A standalone projection retains real VCS stamping and avoids
// accidentally discovering an unrelated parent repository. The apply worktree
// and its journal remain authoritative and are checked separately throughout.
func verificationSource(p plan, candidate, attempt string) (string, error) {
	if err := checkCandidate(p, candidate, true); err != nil {
		return "", err
	}
	projection := filepath.Join(attempt, "source")
	if _, err := command(p.Root, nil, "git", "clone", "--no-checkout", "--shared", "--", p.Root, projection); err != nil {
		return "", err
	}
	if _, err := command(projection, nil, "git", "checkout", "--detach", p.Head); err != nil {
		return "", err
	}
	after := make(map[string][]byte, len(p.Changes))
	for _, change := range p.Changes {
		after[change.Path] = change.After
	}
	for _, input := range p.Inputs {
		path := filepath.Join(projection, input.Path)
		if body, changed := after[input.Path]; changed {
			if err := atomicWrite(path, body, os.FileMode(input.Mode)); err != nil {
				return "", err
			}
		} else if err := os.Chmod(path, os.FileMode(input.Mode)); err != nil {
			return "", err
		}
	}
	if err := checkCandidate(p, projection, true); err != nil {
		return "", err
	}
	if err := checkCandidate(p, candidate, true); err != nil {
		return "", err
	}
	return projection, nil
}

func verificationEnvironment(goPath, state string) ([]string, error) {
	cache := filepath.Join(state, "quality-tools")
	for _, directory := range []string{cache, filepath.Join(cache, "bin")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return nil, err
		}
		resolved, err := filepath.EvalSymlinks(directory)
		if err != nil || resolved != directory {
			return nil, errors.New("candidate quality executable cache must stay inside real state directories")
		}
	}
	env := []string{}
	for _, entry := range goEnvironment(goPath) {
		if !strings.HasPrefix(entry, "HELOX_TOOL_CACHE=") {
			env = append(env, entry)
		}
	}
	return append(env, "HELOX_TOOL_CACHE="+cache), nil
}
