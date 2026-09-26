package promotion

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const pypiVenvMetadata = ".heliopause/pypi-transaction.json"
const pypiVenvStateBound = 32 << 20

// Source is private output. Final is bound once from the resolver's Scheme and
// Relative identity; the transaction never interprets a RECORD pathname.
type pypiDestination struct {
	Scheme         string `json:"scheme"`
	Relative       string `json:"relative"`
	Distribution   string `json:"distribution"`
	Version        string `json:"version"`
	ArtifactDigest string `json:"artifact_digest"`
	Digest         string `json:"sha256"`
	Size           int64  `json:"size"`
	Source         string `json:"-"`
	Final          string `json:"-"`
}

func (d pypiDestination) key() string { return d.Scheme + "/" + d.Relative }

type pypiVenvPlan struct{ root, site string }

func discoverPythonVenv(root string) (pypiVenvPlan, error) {
	p := pypiVenvPlan{root, filepath.Join(root, "lib", "python3.14", "site-packages")}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" || trustedExistingDirectory(root) != nil {
		return p, errors.New("untrusted virtual environment root")
	}
	if _, err := snapshotPyPIFile(filepath.Join(root, "pyvenv.cfg")); err != nil {
		return p, err
	}
	cfg, err := readBoundedPromotionFile(filepath.Join(root, "pyvenv.cfg"), 64<<10)
	if err != nil || !utf8.Valid(cfg) || bytes.IndexByte(cfg, 0) >= 0 {
		return p, errors.New("invalid virtual environment configuration")
	}
	version := ""
	for _, line := range strings.Split(string(cfg), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return p, errors.New("malformed virtual environment configuration")
		}
		if strings.TrimSpace(key) == "version" {
			if version != "" {
				return p, errors.New("duplicate virtual environment version")
			}
			version = strings.TrimSpace(value)
		}
	}
	parts := strings.Split(version, ".")
	if len(parts) != 3 || parts[0] != "3" || parts[1] != "14" || parts[2] == "" || strings.Trim(parts[2], "0123456789") != "" {
		return p, errors.New("unsupported virtual environment version")
	}
	for _, dir := range []string{p.site, filepath.Join(root, "bin")} {
		if trustedExistingDirectory(dir) != nil {
			return p, errors.New("virtual environment scheme unavailable")
		}
	}
	data := filepath.Join(root, "share")
	if _, err := os.Lstat(data); !errors.Is(err, os.ErrNotExist) && trustedExistingDirectory(data) != nil {
		return p, errors.New("unsafe virtual environment data root")
	}
	return p, nil
}
func (p pypiVenvPlan) bind(d pypiDestination) (pypiDestination, error) {
	root := ""
	switch d.Scheme {
	case "site":
		root = p.site
	case "scripts":
		root = filepath.Join(p.root, "bin")
	case "data":
		root = filepath.Join(p.root, "share")
	default:
		return d, errors.New("unknown installation scheme")
	}
	if d.Relative == "" || strings.ContainsAny(d.Relative, "\\\x00:") || filepath.IsAbs(d.Relative) || filepath.ToSlash(filepath.Clean(d.Relative)) != d.Relative {
		return d, errors.New("noncanonical destination")
	}
	final := filepath.Join(root, filepath.FromSlash(d.Relative))
	if !pathWithin(root, final) || (d.Final != "" && d.Final != final) {
		return d, errors.New("destination escapes scheme")
	}
	d.Final = final
	return d, nil
}

type pypiFileSnapshot struct {
	Digest string
	Size   int64
	Mode   os.FileMode
	Device uint64
	Inode  uint64
	info   os.FileInfo
}

// Fail closed if link-count evidence is unavailable on an unsupported Host.
func pypiSingleLink(info os.FileInfo) bool {
	v := reflect.ValueOf(info.Sys())
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return false
	}
	n := v.Elem().FieldByName("Nlink")
	return n.IsValid() && n.CanUint() && n.Uint() == 1
}
func snapshotPyPIFile(path string) (pypiFileSnapshot, error) {
	if rejectSymlinkPath(path) != nil {
		return pypiFileSnapshot{}, errors.New("unsafe file path")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || !pypiSingleLink(info) {
		return pypiFileSnapshot{}, errors.New("expected single-link regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return pypiFileSnapshot{}, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return pypiFileSnapshot{}, errors.New("file identity changed")
	}
	h := sha256.New()
	n, err := io.Copy(h, f)
	after, se := os.Lstat(path)
	if err != nil || se != nil || !os.SameFile(info, after) || !pypiSingleLink(after) || n != info.Size() || after.Size() != n || after.ModTime() != info.ModTime() || after.Mode() != info.Mode() {
		return pypiFileSnapshot{}, errors.New("file changed during snapshot")
	}
	device, inode, ok := pypiObjectIdentity(info)
	if !ok {
		return pypiFileSnapshot{}, errors.New("file identity unavailable")
	}
	return pypiFileSnapshot{hex.EncodeToString(h.Sum(nil)), n, info.Mode(), device, inode, info}, nil
}

func pypiObjectIdentity(info os.FileInfo) (uint64, uint64, bool) {
	v := reflect.ValueOf(info.Sys())
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return 0, 0, false
	}
	dev, ino := v.Elem().FieldByName("Dev"), v.Elem().FieldByName("Ino")
	if !dev.IsValid() || !ino.IsValid() || !ino.CanUint() {
		return 0, 0, false
	}
	if dev.CanUint() {
		return dev.Uint(), ino.Uint(), true
	}
	if dev.CanInt() {
		return uint64(dev.Int()), ino.Uint(), true
	}
	return 0, 0, false
}

func (s pypiFileSnapshot) matches(path string) bool {
	now, err := snapshotPyPIFile(path)
	return err == nil && os.SameFile(s.info, now.info) && s.Digest == now.Digest && s.Size == now.Size && s.Mode == now.Mode
}

type pypiVenvState struct {
	Version int                        `json:"version"`
	Files   map[string]pypiDestination `json:"files"`
}

func (p pypiVenvPlan) readState() (pypiVenvState, bool, error) {
	state := pypiVenvState{2, map[string]pypiDestination{}}
	path := filepath.Join(p.root, pypiVenvMetadata)
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return state, false, nil
	}
	if _, err := snapshotPyPIFile(path); err != nil {
		return state, false, err
	}
	body, err := readBoundedPromotionFile(path, pypiVenvStateBound)
	if err != nil {
		return state, false, err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&state) != nil || state.Version != 2 || len(state.Files) == 0 || decoder.Decode(new(any)) != io.EOF {
		return state, false, errors.New("unsupported ownership metadata; explicit recovery required")
	}
	for key, d := range state.Files {
		bound, err := p.bind(d)
		digest, de := hex.DecodeString(d.Digest)
		if err != nil || key != d.key() || de != nil || len(digest) != 32 || d.Size < 0 || d.Distribution == "" || d.Version == "" {
			return state, false, errors.New("invalid ownership metadata")
		}
		state.Files[key] = bound
	}
	return state, true, nil
}
func (p pypiVenvPlan) verifyState(state pypiVenvState) error {
	for key, d := range state.Files {
		bound, err := p.bind(d)
		if err != nil || key != d.key() {
			return errors.New("invalid ownership destination")
		}
		actual, err := snapshotPyPIFile(bound.Final)
		if err != nil || actual.Digest != d.Digest || actual.Size != d.Size {
			return errors.New("HAA-owned file drift")
		}
	}
	return nil
}

type pypiVenvGuard struct {
	path     string
	identity os.FileInfo
}

func acquirePyPIVenvGuard(root string) (pypiVenvGuard, error) {
	path := filepath.Join(root, ".heliopause-pypi-transaction.lock")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return pypiVenvGuard{}, errors.New("virtual environment is already being mutated")
	}
	info, se := f.Stat()
	err = errors.Join(se, f.Sync(), f.Close())
	if err != nil {
		return pypiVenvGuard{}, err
	}
	return pypiVenvGuard{path, info}, nil
}
func (g pypiVenvGuard) verify() error {
	now, err := os.Lstat(g.path)
	if err != nil || !os.SameFile(g.identity, now) || !now.Mode().IsRegular() || !pypiSingleLink(now) {
		return errors.New("transaction guard identity changed")
	}
	return nil
}
func (g pypiVenvGuard) release() error {
	if err := g.verify(); err != nil {
		return err
	}
	return os.Remove(g.path)
}

// One guard, backup area and ledger cover all exact destinations.
type pypiVenvTransaction struct {
	plan                                            pypiVenvPlan
	guard                                           pypiVenvGuard
	current                                         pypiVenvState
	config                                          pypiFileSnapshot
	metadata                                        *pypiFileSnapshot
	directories                                     map[string]os.FileInfo
	absentDirectories                               map[string]bool
	created                                         []string
	before                                          map[string]*pypiFileSnapshot
	outputs                                         map[string]pypiFileSnapshot
	publishedMetadata                               *pypiFileSnapshot
	published, backed                               []pypiDestination
	backup                                          string
	metadataPublished, metadataBacked, recovery     bool
	desired                                         pypiVenvState
	journalStarted                                  bool
	journalIdentity                                 os.FileInfo
	journalBacked, journalPublished, journalCreated int
	journalMetadataBacked, journalMetadataPublished bool
	// Instance-local deterministic fault seam, never Artifact/config input.
	fault func(string) error
}

func beginPyPIVenvTransaction(p pypiVenvPlan) (_ *pypiVenvTransaction, resultErr error) {
	g, err := acquirePyPIVenvGuard(p.root)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, g.release())
		}
	}()
	if _, err := discoverPythonVenv(p.root); err != nil {
		return nil, err
	}
	t := &pypiVenvTransaction{plan: p, guard: g, directories: map[string]os.FileInfo{}, absentDirectories: map[string]bool{}, before: map[string]*pypiFileSnapshot{}}
	entries, err := os.ReadDir(p.root)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".heliopause-pypi-commit-") {
			return nil, errors.New("interrupted transaction requires explicit recovery")
		}
	}
	t.config, err = snapshotPyPIFile(filepath.Join(p.root, "pyvenv.cfg"))
	if err != nil {
		return nil, err
	}
	t.current, _, err = p.readState()
	if err != nil || p.verifyState(t.current) != nil {
		return nil, errors.New("ownership state unavailable or changed")
	}
	for _, path := range []string{p.root, p.site, filepath.Join(p.root, "bin"), filepath.Join(p.root, "share"), filepath.Join(p.root, ".heliopause")} {
		if err := t.freezeParents(path); err != nil {
			return nil, err
		}
	}
	meta := filepath.Join(p.root, pypiVenvMetadata)
	if _, err := os.Lstat(meta); !errors.Is(err, os.ErrNotExist) {
		s, err := snapshotPyPIFile(meta)
		if err != nil {
			return nil, err
		}
		t.metadata = &s
	}
	for _, d := range t.current.Files {
		s, err := snapshotPyPIFile(d.Final)
		if err != nil {
			return nil, err
		}
		t.before[d.Final] = &s
		if err := t.freezeParents(filepath.Dir(d.Final)); err != nil {
			return nil, err
		}
	}
	return t, nil
}
func (t *pypiVenvTransaction) close() error {
	// Uncertain recovery retains both the guard and the recovery directory.
	if t.recovery {
		return t.guard.verify()
	}
	return t.guard.release()
}
func (t *pypiVenvTransaction) step(name string) error {
	if t.fault != nil {
		return t.fault(name)
	}
	return nil
}
func (t *pypiVenvTransaction) freezeParents(path string) error {
	for {
		if _, ok := t.directories[path]; !ok && !t.absentDirectories[path] {
			info, err := os.Lstat(path)
			if errors.Is(err, os.ErrNotExist) {
				t.absentDirectories[path] = true
			} else if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return errors.New("unsafe destination parent")
			} else {
				t.directories[path] = info
			}
		}
		if path == "/" {
			break
		}
		path = filepath.Dir(path)
	}
	return nil
}
func (t *pypiVenvTransaction) verifyIdentity() error {
	if err := t.guard.verify(); err != nil {
		return err
	}
	if !t.config.matches(filepath.Join(t.plan.root, "pyvenv.cfg")) {
		return errors.New("virtual environment configuration drift")
	}
	for path, before := range t.directories {
		now, err := os.Lstat(path)
		if err != nil || !now.IsDir() || !os.SameFile(before, now) || now.Mode() != before.Mode() {
			return errors.New("destination directory identity drift")
		}
	}
	for path := range t.absentDirectories {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return errors.New("destination directory appeared")
		}
	}
	return nil
}
func (t *pypiVenvTransaction) checkBefore() error {
	if err := t.verifyIdentity(); err != nil {
		return err
	}
	for path, s := range t.before {
		if s == nil {
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				return errors.New("destination appeared")
			}
		} else if !s.matches(path) {
			return errors.New("destination pre-state drift")
		}
	}
	meta := filepath.Join(t.plan.root, pypiVenvMetadata)
	if t.metadata == nil {
		if _, err := os.Lstat(meta); !errors.Is(err, os.ErrNotExist) {
			return errors.New("ownership metadata appeared")
		}
	} else if !t.metadata.matches(meta) {
		return errors.New("ownership metadata drift")
	}
	return nil
}

// Check the fixed roots and the current file's ancestry at each mutation;
// full directory-set reconciliation remains at freeze and completion.
func (t *pypiVenvTransaction) verifyPathIdentity(path string) error {
	if err := t.guard.verify(); err != nil {
		return err
	}
	if !t.config.matches(filepath.Join(t.plan.root, "pyvenv.cfg")) {
		return errors.New("configuration drift")
	}
	paths := []string{t.plan.root, t.plan.site, filepath.Join(t.plan.root, "bin"), filepath.Join(t.plan.root, "share"), filepath.Join(t.plan.root, ".heliopause")}
	for {
		paths = append(paths, path)
		if path == "/" {
			break
		}
		path = filepath.Dir(path)
	}
	for _, path := range paths {
		before, exists := t.directories[path]
		now, err := os.Lstat(path)
		if exists {
			if err != nil || !now.IsDir() || !os.SameFile(before, now) || before.Mode() != now.Mode() {
				return errors.New("destination parent drift")
			}
		} else if !t.absentDirectories[path] || !errors.Is(err, os.ErrNotExist) {
			return errors.New("unplanned destination parent")
		}
	}
	return nil
}

// Host-side Linux descriptor paths anchor each mutation to the already-frozen
// parent inode. They confer no sandbox runtime/observer path classification.
func (t *pypiVenvTransaction) openParent(path string) (*os.File, string, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return nil, "", errors.New("unsupported venv transaction Host")
	}
	directory := filepath.Dir(path)
	before, ok := t.directories[directory]
	if !ok || filepath.Clean(path) != path {
		return nil, "", errors.New("unplanned mutation parent")
	}
	if err := t.verifyPathIdentity(directory); err != nil {
		return nil, "", err
	}
	f, err := os.Open(directory)
	if err != nil {
		return nil, "", err
	}
	now, err := f.Stat()
	if err != nil || !now.IsDir() || !os.SameFile(before, now) {
		_ = f.Close()
		return nil, "", errors.New("mutation parent replaced")
	}
	return f, filepath.Join("/proc/self/fd", strconv.FormatUint(uint64(f.Fd()), 10), filepath.Base(path)), nil
}
func (t *pypiVenvTransaction) rename(from, to string) error {
	source, oldPath, err := t.openParent(from)
	if err != nil {
		return err
	}
	defer source.Close()
	target, newPath, err := t.openParent(to)
	if err != nil {
		return err
	}
	defer target.Close()
	return renameNoReplace(oldPath, newPath)
}
func (t *pypiVenvTransaction) remove(path string) error {
	parent, name, err := t.openParent(path)
	if err != nil {
		return err
	}
	defer parent.Close()
	return os.Remove(name)
}
func (t *pypiVenvTransaction) prepare(destinations []pypiDestination) (pypiVenvState, []pypiDestination, error) {
	desired := pypiVenvState{2, map[string]pypiDestination{}}
	t.outputs = map[string]pypiFileSnapshot{}
	projects := map[string]string{}
	finals := map[string]bool{}
	for _, raw := range destinations {
		d, err := t.plan.bind(raw)
		if err != nil || finals[d.Final] || d.Distribution == "" || d.Version == "" {
			return desired, nil, errors.New("duplicate or invalid destination")
		}
		if v := projects[d.Distribution]; v != "" && v != d.Version {
			return desired, nil, errors.New("ambiguous distribution")
		}
		projects[d.Distribution] = d.Version
		finals[d.Final] = true
		actual, err := snapshotPyPIFile(d.Source)
		if err != nil || actual.Digest != d.Digest || actual.Size != d.Size {
			return desired, nil, errors.New("private output drift")
		}
		t.outputs[d.Final] = actual
		if err := t.freezeParents(filepath.Dir(d.Source)); err != nil {
			return desired, nil, err
		}
		if prior, exists := t.current.Files[d.key()]; exists {
			if prior.Distribution != d.Distribution {
				return desired, nil, errors.New("destination belongs to another distribution")
			}
		} else {
			if _, err := os.Lstat(d.Final); !errors.Is(err, os.ErrNotExist) {
				return desired, nil, errors.New("unmanaged destination collision")
			}
			t.before[d.Final] = nil
		}
		if err := t.freezeParents(filepath.Dir(d.Final)); err != nil {
			return desired, nil, err
		}
		desired.Files[d.key()] = d
	}
	if len(destinations) == 0 {
		return desired, nil, errors.New("empty transaction")
	}
	obsolete := []pypiDestination{}
	for key, old := range t.current.Files {
		if _, replaces := projects[old.Distribution]; replaces {
			if _, present := desired.Files[key]; !present {
				obsolete = append(obsolete, old)
			}
		} else {
			desired.Files[key] = old
		}
	}
	sort.Slice(obsolete, func(i, j int) bool { return obsolete[i].key() < obsolete[j].key() })
	return desired, obsolete, t.checkBefore()
}
func (t *pypiVenvTransaction) createParents(path string) error {
	if _, ok := t.directories[path]; ok {
		return t.verifyPathIdentity(path)
	}
	if !t.absentDirectories[path] {
		return errors.New("unplanned directory creation")
	}
	if err := t.createParents(filepath.Dir(path)); err != nil {
		return err
	}
	if err := t.journalIntent("mkdir", path); err != nil {
		return err
	}
	parent, name, err := t.openParent(path)
	if err != nil {
		return err
	}
	defer parent.Close()
	if err := os.Mkdir(name, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(name)
	if err != nil {
		return err
	}
	t.created = append(t.created, path)
	t.directories[path] = info
	delete(t.absentDirectories, path)
	return t.persistLedger()
}

func (t *pypiVenvTransaction) persistLedger() error {
	// Synced append-only records avoid rewriting the entire set for every file.
	// Intent records precede mutation. Recovery always requires explicit action.
	if !t.journalStarted {
		directories := map[string]pypiFileSnapshot{}
		for path, info := range t.directories {
			dev, ino, ok := pypiObjectIdentity(info)
			if !ok {
				return errors.New("recovery directory identity unavailable")
			}
			directories[path] = pypiFileSnapshot{Mode: info.Mode(), Device: dev, Inode: ino}
		}
		if err := t.appendJournal(struct {
			Root              string
			Before            map[string]*pypiFileSnapshot
			Current, Desired  pypiVenvState
			Config            pypiFileSnapshot
			Metadata          *pypiFileSnapshot
			Directories       map[string]pypiFileSnapshot
			AbsentDirectories map[string]bool
		}{t.plan.root, t.before, t.current, t.desired, t.config, t.metadata, directories, t.absentDirectories}); err != nil {
			return err
		}
	}
	for t.journalBacked < len(t.backed) {
		if err := t.appendJournal(struct {
			Backed pypiDestination
			Index  int
		}{t.backed[t.journalBacked], t.journalBacked}); err != nil {
			return err
		}
		t.journalBacked++
	}
	for t.journalPublished < len(t.published) {
		if err := t.appendJournal(struct{ Published pypiDestination }{t.published[t.journalPublished]}); err != nil {
			return err
		}
		t.journalPublished++
	}
	for t.journalCreated < len(t.created) {
		path := t.created[t.journalCreated]
		dev, ino, ok := pypiObjectIdentity(t.directories[path])
		if !ok {
			return errors.New("created directory identity unavailable")
		}
		pd, pi, ok := pypiObjectIdentity(t.directories[filepath.Dir(path)])
		if !ok {
			return errors.New("created directory parent unavailable")
		}
		scheme := ""
		for _, candidate := range []struct{ name, root string }{{"site", t.plan.site}, {"scripts", filepath.Join(t.plan.root, "bin")}, {"data", filepath.Join(t.plan.root, "share")}} {
			if path == candidate.root || pathWithin(candidate.root, path) {
				scheme = candidate.name
				break
			}
		}
		if path == filepath.Join(t.plan.root, ".heliopause") {
			scheme = "controller-metadata"
		}
		if scheme == "" {
			return errors.New("created directory has no exact scheme")
		}
		if err := t.appendJournal(struct {
			Created                                  string
			Scheme                                   string
			Order                                    int
			Device, Inode, ParentDevice, ParentInode uint64
		}{path, scheme, t.journalCreated, dev, ino, pd, pi}); err != nil {
			return err
		}
		t.journalCreated++
	}
	if t.metadataBacked != t.journalMetadataBacked || t.metadataPublished != t.journalMetadataPublished {
		if err := t.appendJournal(struct{ MetadataBacked, MetadataPublished bool }{t.metadataBacked, t.metadataPublished}); err != nil {
			return err
		}
		t.journalMetadataBacked, t.journalMetadataPublished = t.metadataBacked, t.metadataPublished
	}
	return nil
}
func (t *pypiVenvTransaction) appendJournal(value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	flags := os.O_WRONLY | os.O_APPEND
	if !t.journalStarted {
		flags |= os.O_CREATE | os.O_EXCL
	}
	parent, path, err := t.openParent(filepath.Join(t.backup, "ledger.json"))
	if err != nil {
		return err
	}
	defer parent.Close()
	f, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || !pypiSingleLink(info) || (t.journalStarted && !os.SameFile(t.journalIdentity, info)) {
		_ = f.Close()
		return errors.New("recovery journal identity changed")
	}
	t.journalIdentity = info
	_, we := f.Write(append(body, '\n'))
	if err := errors.Join(we, f.Sync(), f.Close()); err != nil {
		return err
	}
	if !t.journalStarted {
		t.journalStarted = true
		return syncDirectory(t.backup)
	}
	return nil
}
func (t *pypiVenvTransaction) journalIntent(action, path string) error {
	return t.appendJournal(struct{ Action, Path string }{action, path})
}
func (t *pypiVenvTransaction) commit(destinations []pypiDestination) (resultErr error) {
	desired, obsolete, err := t.prepare(destinations)
	if err != nil {
		return err
	}
	t.desired = desired
	if err := t.step("before-mutation"); err != nil {
		return err
	}
	if err := t.checkBefore(); err != nil {
		return err
	}
	t.backup, err = os.MkdirTemp(t.plan.root, ".heliopause-pypi-commit-")
	if err != nil {
		return err
	}
	if err := t.freezeParents(t.backup); err != nil {
		t.recovery = true
		return err
	}
	defer func() {
		if resultErr != nil {
			re := t.rollback()
			if re != nil {
				t.recovery = true
				resultErr = errors.Join(resultErr, re, errors.New("rollback incomplete; recovery state retained"))
				return
			}
		}
		if err := os.RemoveAll(t.backup); err != nil {
			t.recovery = true
			resultErr = errors.Join(resultErr, errors.New("recovery cleanup failed"))
		}
		if err := syncDirectory(t.plan.root); err != nil {
			t.recovery = true
			resultErr = errors.Join(resultErr, errors.New("recovery cleanup sync failed"))
		}
	}()
	if err := t.persistLedger(); err != nil {
		return err
	}
	if err := syncDirectory(t.plan.root); err != nil {
		return err
	}
	ordered := append([]pypiDestination(nil), destinations...)
	for i, d := range ordered {
		ordered[i], err = t.plan.bind(d)
		if err != nil {
			return err
		}
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].key() < ordered[j].key() })
	backups := append(append([]pypiDestination(nil), ordered...), obsolete...)
	for _, d := range backups {
		before := t.before[d.Final]
		if before == nil {
			continue
		}
		if err := t.verifyPathIdentity(filepath.Dir(d.Final)); err != nil {
			return err
		}
		if !before.matches(d.Final) {
			return errors.New("managed destination changed before backup")
		}
		backup := filepath.Join(t.backup, "file-"+strconv.Itoa(len(t.backed)))
		if err := t.journalIntent("backup", d.Final); err != nil {
			return err
		}
		if err := t.rename(d.Final, backup); err != nil {
			return err
		}
		t.backed = append(t.backed, d)
		if err := t.persistLedger(); err != nil {
			return err
		}
	}
	for _, d := range ordered {
		if err := t.createParents(filepath.Dir(d.Final)); err != nil {
			return err
		}
		if err := t.step("before-publish:" + d.key()); err != nil {
			return err
		}
		if err := t.verifyPathIdentity(filepath.Dir(d.Final)); err != nil {
			return err
		}
		if !t.outputs[d.Final].matches(d.Source) {
			return errors.New("private output changed before publication")
		}
		if err := t.journalIntent("publish", d.Final); err != nil {
			return err
		}
		if err := t.rename(d.Source, d.Final); err != nil {
			return err
		}
		t.published = append(t.published, d)
		if err := t.persistLedger(); err != nil {
			return err
		}
		if err := t.step("after-publish:" + d.Scheme); err != nil {
			return err
		}
	}
	if err := t.createParents(filepath.Join(t.plan.root, ".heliopause")); err != nil {
		return err
	}
	if err := t.step("metadata"); err != nil {
		return err
	}
	if err := t.verifyIdentity(); err != nil {
		return err
	}
	meta := filepath.Join(t.plan.root, pypiVenvMetadata)
	if t.metadata != nil {
		if !t.metadata.matches(meta) {
			return errors.New("ownership metadata drift")
		}
		if err := t.journalIntent("backup-metadata", meta); err != nil {
			return err
		}
		if err := t.rename(meta, filepath.Join(t.backup, "metadata")); err != nil {
			return err
		}
		t.metadataBacked = true
		if err := t.persistLedger(); err != nil {
			return err
		}
	}
	body, err := json.Marshal(desired)
	if err != nil || len(body) > pypiVenvStateBound {
		return errors.New("ownership metadata exceeds bound")
	}
	parent, metadataPath, err := t.openParent(filepath.Join(t.backup, "next-metadata"))
	if err != nil {
		return err
	}
	writeErr := writePromotionFile(metadataPath, body)
	closeErr := parent.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	if err := t.journalIntent("publish-metadata", meta); err != nil {
		return err
	}
	metadataSnapshot, err := snapshotPyPIFile(filepath.Join(t.backup, "next-metadata"))
	if err != nil {
		return err
	}
	t.publishedMetadata = &metadataSnapshot
	if err := t.rename(filepath.Join(t.backup, "next-metadata"), meta); err != nil {
		return err
	}
	t.metadataPublished = true
	if err := t.persistLedger(); err != nil {
		return err
	}
	if err := t.step("sync"); err != nil {
		return err
	}
	for _, d := range ordered {
		f, err := os.Open(d.Final)
		if err != nil {
			return err
		}
		if err := errors.Join(f.Sync(), f.Close()); err != nil {
			return err
		}
	}
	if err := t.syncDirectories(); err != nil {
		return err
	}
	if err := t.step("reconcile"); err != nil {
		return err
	}
	if err := t.verifyIdentity(); err != nil {
		return err
	}
	if err := t.plan.verifyState(desired); err != nil {
		return err
	}
	for _, d := range ordered {
		if !t.outputs[d.Final].matches(d.Final) {
			return errors.New("published file inode drift")
		}
	}
	if !t.publishedMetadata.matches(meta) {
		return errors.New("published metadata inode drift")
	}
	for _, d := range obsolete {
		if _, err := os.Lstat(d.Final); !errors.Is(err, os.ErrNotExist) {
			return errors.New("obsolete owned output remains")
		}
	}
	changed := map[string]bool{}
	for _, d := range t.backed {
		changed[d.Final] = true
	}
	for path, before := range t.before {
		if before != nil && !changed[path] && !before.matches(path) {
			return errors.New("retained managed file drift")
		}
	}
	actual, err := readBoundedPromotionFile(meta, pypiVenvStateBound)
	if err != nil || !bytes.Equal(body, actual) {
		return errors.New("ownership metadata reconciliation failed")
	}
	return nil
}
func (t *pypiVenvTransaction) syncDirectories() error {
	paths := []string{}
	for path := range t.directories {
		if path == t.plan.root || pathWithin(t.plan.root, path) {
			paths = append(paths, path)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	for _, path := range paths {
		if err := syncDirectory(path); err != nil {
			return err
		}
	}
	return nil
}
func (t *pypiVenvTransaction) rollback() error {
	if err := t.step("rollback"); err != nil {
		return err
	}
	if err := t.verifyIdentity(); err != nil {
		return err
	}
	meta := filepath.Join(t.plan.root, pypiVenvMetadata)
	if t.metadataPublished {
		if t.publishedMetadata == nil || !t.publishedMetadata.matches(meta) {
			return errors.New("metadata drift prevents rollback")
		}
		if err := t.remove(meta); err != nil {
			return err
		}
	}
	if t.metadataBacked {
		if !t.metadata.matches(filepath.Join(t.backup, "metadata")) {
			return errors.New("metadata backup drift")
		}
		if err := t.rename(filepath.Join(t.backup, "metadata"), meta); err != nil {
			return err
		}
	}
	for i := len(t.published) - 1; i >= 0; i-- {
		d := t.published[i]
		if !t.outputs[d.Final].matches(d.Final) {
			return errors.New("published file drift prevents rollback")
		}
		if err := t.remove(d.Final); err != nil {
			return err
		}
	}
	for i := len(t.backed) - 1; i >= 0; i-- {
		d := t.backed[i]
		backup := filepath.Join(t.backup, "file-"+strconv.Itoa(i))
		if !t.before[d.Final].matches(backup) {
			return errors.New("backup drift prevents rollback")
		}
		if err := t.rename(backup, d.Final); err != nil {
			return err
		}
	}
	for i := len(t.created) - 1; i >= 0; i-- {
		path := t.created[i]
		now, err := os.Lstat(path)
		if err != nil || !os.SameFile(t.directories[path], now) {
			return errors.New("created directory drift prevents rollback")
		}
		if err := t.remove(path); err != nil {
			return err
		}
		delete(t.directories, path)
		t.absentDirectories[path] = true
	}
	if err := t.plan.verifyState(t.current); err != nil {
		return err
	}
	if err := t.checkBefore(); err != nil {
		return err
	}
	return t.syncDirectories()
}
func (p pypiVenvPlan) String() string { return fmt.Sprintf("%s:%s", p.root, p.site) }
