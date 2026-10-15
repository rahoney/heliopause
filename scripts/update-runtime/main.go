// Command update-runtime prepares isolated, explicitly selected runtime upgrades.
// It never activates a host installation or qualifies a product runtime itself.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

const runtimeFile = "scripts/runtimes.lock.json"
const supportFile = "scripts/version-support.lock.json"
const generatedFile = "internal/runtimeidentity/runtime_lock_gen.go"

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type inputFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Mode   uint32 `json:"mode"`
}
type change struct {
	Path   string `json:"path"`
	Before []byte `json:"before"`
	After  []byte `json:"after"`
}
type plan struct {
	Schema                int              `json:"schema"`
	ID                    string           `json:"id"`
	Root                  string           `json:"root"`
	Head                  string           `json:"head"`
	Component             string           `json:"component"`
	Current               string           `json:"current"`
	Version               string           `json:"version"`
	Inputs                []inputFile      `json:"inputs"`
	Metadata              metadata         `json:"metadata"`
	Review                []map[string]any `json:"review"`
	GeneratorSHA256       string           `json:"generator_sha256"`
	Changes               []change         `json:"changes"`
	Checks                []string         `json:"checks"`
	RequiredQualification []string         `json:"required_qualification"`
}
type journal struct {
	PlanID       string   `json:"plan_id"`
	Phase        string   `json:"phase"`
	FirstFailure string   `json:"first_failure,omitempty"`
	Attempts     []string `json:"attempts"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: update-runtime <plan|apply|verify> --root REPO --state EXTERNAL_DIR [--component go|node|python|rust|terraform|github --version X.Y.Z --review JSON] [--go ABSOLUTE_TARGET_GO]")
	}
	fs := flag.NewFlagSet("update-runtime "+args[0], flag.ContinueOnError)
	fs.SetOutput(out)
	root := fs.String("root", ".", "clean versioned repository")
	state := fs.String("state", "", "external persistent plan/journal directory (required)")
	component := fs.String("component", "", "explicit runtime or inspection subject")
	version := fs.String("version", "", "exact stable version; inspection subjects use no version")
	review := fs.String("review", "", "JSON array of reviewed support records for changed versions")
	goPath := fs.String("go", "", "absolute target Go executable; verified against owner and official archive")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if args[0] != "plan" && args[0] != "apply" && args[0] != "verify" {
		return errors.New("unknown operation")
	}
	absRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	absRoot, err = filepath.EvalSymlinks(absRoot)
	if err != nil {
		return err
	}
	if *state == "" || !filepath.IsAbs(*state) {
		return errors.New("state must be an absolute external directory")
	}
	absState := filepath.Clean(*state)
	if err := os.MkdirAll(absState, 0o700); err != nil {
		return err
	}
	absState, err = filepath.EvalSymlinks(absState)
	if err != nil {
		return err
	}
	if within(absRoot, absState) || within(absState, absRoot) {
		return errors.New("state and repository must be disjoint")
	}
	unlock, err := lockState(absState)
	if err != nil {
		return err
	}
	defer unlock()
	if args[0] == "plan" {
		p, err := makePlan(absRoot, absState, *component, *version, *review)
		if err != nil {
			return err
		}
		if err := saveJSON(filepath.Join(absState, "plan.json"), p); err != nil {
			return err
		}
		fmt.Fprintf(out, "plan %s: %s %s -> %s; %d changed files; qualification=%v\n", p.ID, p.Component, p.Current, p.Version, len(p.Changes), p.RequiredQualification)
		return nil
	}
	var p plan
	if err := loadJSON(filepath.Join(absState, "plan.json"), &p); err != nil {
		return err
	}
	if p.Root != absRoot || p.Schema != 1 || p.ID != planID(p) {
		return errors.New("plan identity/root mismatch")
	}
	if *component != "" || *version != "" || *review != "" {
		return errors.New("apply/verify use frozen plan inputs only")
	}
	if args[0] == "apply" {
		return applyPlan(p, absState, out, nil)
	}
	return verifyPlan(p, absState, *goPath, out)
}

func makePlan(root, state, component, version, reviewPath string) (plan, error) {
	return makePlanResolving(root, state, component, version, reviewPath, resolveMetadata)
}

func makePlanResolving(root, state, component, version, reviewPath string, resolve func(string, string) (metadata, error)) (plan, error) {
	p := plan{Schema: 1, Root: root, Component: component, Version: version, Changes: []change{}, Review: []map[string]any{}, Checks: []string{"bootstrap", "quick", "security", "workflow", "release-gate", "docs", "qualification-freshness"}, RequiredQualification: []string{}}
	if _, err := os.Lstat(filepath.Join(state, "plan.json")); err == nil {
		return p, errors.New("state already contains a plan; use a new state directory")
	} else if !errors.Is(err, os.ErrNotExist) {
		return p, err
	}
	var err error
	p.Head, p.Inputs, err = snapshot(root, true)
	if err != nil {
		return p, err
	}
	p.Current, err = selectedVersion(root, component)
	if err != nil {
		return p, err
	}
	if component == "terraform" || component == "github" {
		if version != "" || reviewPath != "" {
			return p, errors.New("inspection subjects have no managed language version; user artifacts are unchanged")
		}
		p.RequiredQualification = []string{component + " adapter/common-runtime impact review; no artifact rewrite"}
	} else {
		if !versionPattern.MatchString(version) {
			return p, errors.New("an exact stable X.Y.Z version is required")
		}
		if version != p.Current {
			p.Metadata, err = resolve(component, version)
			if err != nil {
				return p, fmt.Errorf("official acquisition failed (source unchanged): %w", err)
			}
			if reviewPath == "" {
				return p, errors.New("changed version requires --review with actual support review records; metadata alone is not a support decision")
			}
			if err := loadJSON(reviewPath, &p.Review); err != nil {
				return p, err
			}
			p.RequiredQualification = qualification(component)
		} else if reviewPath != "" {
			return p, errors.New("same-version reconciliation preserves existing review records")
		}
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		return p, err
	}
	goBin, err = filepath.EvalSymlinks(goBin)
	if err != nil {
		return p, err
	}
	if err := checkGo(goBin, root); err != nil {
		return p, err
	}
	if _, err := command(root, goEnvironment(goBin), goBin, "build", "-trimpath", "-o", filepath.Join(state, "generator"), "./scripts/generate-runtime-lock.go"); err != nil {
		return p, err
	}
	body, err := os.ReadFile(filepath.Join(state, "generator"))
	if err != nil {
		return p, err
	}
	p.GeneratorSHA256 = hash(body)
	p.Changes, err = renderChanges(p, state)
	if err != nil {
		return p, err
	}
	if err := compareSnapshot(p); err != nil {
		return p, err
	}
	p.ID = planID(p)
	return p, nil
}

func qualification(component string) []string {
	switch component {
	case "go":
		return []string{"quality-tool compiler/rebuild", "gVisor/observer rebuild", "installed npm/PyPI/GitHub/Go/Cargo/Terraform", "PyTorch CPU", "PyTorch cu126", "PyTorch cu130", "PyTorch cu132", "fresh Required/platform evidence", "verified activation/retirement"}
	case "python":
		return []string{"PyPI wheel/sdist/metadata/entry-point/promotion", "PyTorch CPU", "PyTorch cu126", "PyTorch cu130", "PyTorch cu132", "fresh Required/platform evidence", "verified activation"}
	case "node":
		return []string{"npm resolver/inspect/install/promotion", "common-runtime impact review", "fresh Required/platform evidence", "verified activation"}
	case "rust":
		return []string{"Cargo add/transitive/offline build/security/rollback", "common-runtime impact review", "fresh Required/platform evidence", "verified activation"}
	}
	return []string{"unclassified: full impact review"}
}

func snapshot(root string, clean bool) (string, []inputFile, error) {
	head, err := command(root, nil, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", nil, err
	}
	if clean {
		status, err := command(root, nil, "git", "status", "--porcelain=v1", "--untracked-files=no")
		if err != nil {
			return "", nil, err
		}
		if len(status) != 0 {
			return "", nil, errors.New("tracked/index input is dirty")
		}
	}
	paths, err := command(root, nil, "git", "ls-files", "-z")
	if err != nil {
		return "", nil, err
	}
	var inputs []inputFile
	for _, path := range strings.Split(string(paths), "\x00") {
		if path == "" {
			continue
		}
		body, mode, err := readRegular(root, path)
		if err != nil {
			return "", nil, err
		}
		inputs = append(inputs, inputFile{path, hash(body), uint32(mode.Perm())})
	}
	// Untracked developer source can introduce consumers that the baseline does not own.
	untracked, err := command(root, nil, "git", "ls-files", "--others", "--exclude-standard", "-z", "cmd", "internal", "scripts", ".github")
	if err != nil {
		return "", nil, err
	}
	if len(untracked) != 0 {
		return "", nil, errors.New("untracked executable/source inputs require normal Git review first")
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].Path < inputs[j].Path })
	return strings.TrimSpace(string(head)), inputs, nil
}

func compareSnapshot(p plan) error {
	head, inputs, err := snapshot(p.Root, true)
	if err != nil {
		return err
	}
	if head != p.Head || !reflect.DeepEqual(inputs, p.Inputs) {
		return errors.New("stale plan: HEAD, input bytes or modes changed")
	}
	return nil
}

func readRegular(root, path string) ([]byte, os.FileMode, error) {
	if filepath.IsAbs(path) || filepath.Clean(path) != path || path == "." || strings.HasPrefix(path, "..") {
		return nil, 0, errors.New("invalid relative input path")
	}
	full := filepath.Join(root, path)
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil {
		return nil, 0, err
	}
	if resolved != full {
		return nil, 0, fmt.Errorf("symlink input refused: %s", path)
	}
	info, err := os.Lstat(full)
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() {
		return nil, 0, fmt.Errorf("nonregular input: %s", path)
	}
	body, err := os.ReadFile(full)
	return body, info.Mode(), err
}

func planID(p plan) string    { p.ID = ""; body, _ := json.Marshal(p); return hash(body) }
func hash(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func loadJSON(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 32<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}
func saveJSON(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, append(body, '\n'), 0o600)
}
func atomicWrite(path string, body []byte, mode os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".runtime-write-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(body)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
func command(root string, env []string, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = root
	if env != nil {
		cmd.Env = env
	}
	var stdout, stderr cappedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("%s %s failed: %w: %s", filepath.Base(name), strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

type cappedBuffer struct{ bytes.Buffer }

func (b *cappedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len() < 2<<20 {
		remain := (2 << 20) - b.Len()
		if len(p) > remain {
			p = p[:remain]
		}
		_, _ = b.Buffer.Write(p)
	}
	return n, nil
}
func goEnvironment(goBin string) []string {
	env := []string{}
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GOROOT=") || strings.HasPrefix(entry, "GOTOOLCHAIN=") || strings.HasPrefix(entry, "PATH=") {
			continue
		}
		env = append(env, entry)
	}
	return append(env, "GOTOOLCHAIN=local", "GOROOT="+filepath.Dir(filepath.Dir(goBin)), "PATH="+filepath.Dir(goBin)+string(os.PathListSeparator)+os.Getenv("PATH"))
}
func checkGo(goBin, root string) error {
	selected, err := selectedVersion(root, "go")
	if err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(goBin)
	if err != nil {
		return err
	}
	body, err := command(root, goEnvironment(resolved), resolved, "version")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(body)) != "go version go"+selected+" "+runtime.GOOS+"/"+runtime.GOARCH {
		return errors.New("target Go executable/compiler does not match selected owner and host platform")
	}
	return nil
}
