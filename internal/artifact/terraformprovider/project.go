package terraformprovider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/mod/semver"
)

const ConfigurationControl = "terraform-configuration.json"
const LockControl = ".terraform.lock.hcl"

type configurationFile struct {
	Path string `json:"path"`
	Body []byte `json:"body"`
}
type configurationDocument struct {
	Schema int                 `json:"schema"`
	Files  []configurationFile `json:"files"`
}

// ProjectCapture binds complete reachable local-module configuration, directory
// identities and the original lock. The configuration control is an opaque,
// controller-generated snapshot, never a file installed in the user's project.
type ProjectCapture struct {
	Controls []domain.ProjectControlFile
	Members  map[string]os.FileInfo
}

type Requirement struct {
	Address     string
	Constraints []string
}
type LockedProvider struct {
	Address     string
	Version     string
	Constraints string
	Hashes      []string
}

func readProjectMember(root *os.Root, name string, limit int64) ([]byte, os.FileInfo, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, nil, err
	}
	links, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 || !ok || links.Nlink != 1 || info.Size() < 0 || info.Size() > limit {
		return nil, nil, errors.New("terraform control must be bounded single-link regular content")
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil, errors.New("open terraform control")
	}
	opened, e := f.Stat()
	if e != nil || !os.SameFile(info, opened) {
		_ = f.Close()
		return nil, nil, errors.New("terraform control identity changed")
	}
	b, readErr := io.ReadAll(io.LimitReader(f, limit+1))
	after, afterErr := f.Stat()
	closeErr := f.Close()
	current, currentErr := root.Lstat(name)
	if readErr != nil || afterErr != nil || closeErr != nil || currentErr != nil || !os.SameFile(info, current) || !os.SameFile(info, after) || after.ModTime() != info.ModTime() || after.Size() != info.Size() || int64(len(b)) != info.Size() {
		return nil, nil, errors.New("terraform control changed during read")
	}
	return b, info, nil
}

// CaptureProject reads only data through anchored roots; remote modules and
// configuration aliases fail before any provider acquisition or execution.
func CaptureProject(ctx context.Context, directory string) (ProjectCapture, error) {
	if ctx == nil || ctx.Err() != nil || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || directory == "/" {
		return ProjectCapture{}, errors.New("terraform project capture request is invalid")
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ProjectCapture{}, errors.New("terraform project root is untrusted")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return ProjectCapture{}, err
	}
	defer root.Close()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return ProjectCapture{}, errors.New("terraform project root changed")
	}
	result := ProjectCapture{Members: map[string]os.FileInfo{".": info}}
	doc := configurationDocument{Schema: 1}
	pending := []string{"."}
	seen := map[string]bool{}
	total := 0
	for len(pending) > 0 {
		if ctx.Err() != nil {
			return ProjectCapture{}, ctx.Err()
		}
		dir := pending[0]
		pending = pending[1:]
		if seen[dir] {
			continue
		}
		seen[dir] = true
		if len(seen) > 32 {
			return ProjectCapture{}, errors.New("terraform local module graph exceeds directory bound")
		}
		// Every ancestor is independently checked; os.Root confinement alone
		// does not reject symlinks pointing back inside the project.
		for current := dir; ; current = path.Dir(current) {
			di, e := root.Lstat(current)
			if e != nil || !di.IsDir() || di.Mode()&os.ModeSymlink != 0 {
				return ProjectCapture{}, errors.New("terraform module directory is aliased")
			}
			result.Members[current] = di
			if current == "." {
				break
			}
		}
		dirFile, e := root.Open(dir)
		if e != nil {
			return ProjectCapture{}, e
		}
		entries, e := dirFile.ReadDir(10001)
		ce := dirFile.Close()
		if e != nil && !errors.Is(e, io.EOF) || ce != nil || len(entries) > 10000 {
			return ProjectCapture{}, errors.New("terraform module directory inventory exceeds bounds")
		}
		files := map[string][]byte{}
		folded := map[string]bool{}
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".tf") && !strings.HasSuffix(strings.ToLower(name), ".tf.json") {
				continue
			}
			if folded[strings.ToLower(name)] || strings.HasSuffix(name, "override.tf") || strings.HasSuffix(name, "override.tf.json") || strings.ToLower(name) != name {
				return ProjectCapture{}, errors.New("terraform override or case-ambiguous configuration is unsupported")
			}
			folded[strings.ToLower(name)] = true
			member := path.Join(dir, name)
			b, fi, e := readProjectMember(root, member, 1<<20)
			if e != nil {
				return ProjectCapture{}, e
			}
			total += len(b)
			if total > 3<<20 || len(doc.Files) >= 256 {
				return ProjectCapture{}, errors.New("terraform configuration snapshot exceeds bounds")
			}
			result.Members[member] = fi
			files[member] = b
			doc.Files = append(doc.Files, configurationFile{member, b})
		}
		_, modules, e := parseConfiguration(files)
		if e != nil {
			return ProjectCapture{}, e
		}
		for _, module := range modules {
			next := path.Clean(path.Join(dir, module))
			if next == ".." || strings.HasPrefix(next, "../") || strings.HasPrefix(next, "/") {
				return ProjectCapture{}, errors.New("terraform local module escapes project")
			}
			pending = append(pending, next)
		}
	}
	if len(doc.Files) == 0 {
		return ProjectCapture{}, errors.New("terraform project configuration is unavailable")
	}
	sort.Slice(doc.Files, func(i, j int) bool { return doc.Files[i].Path < doc.Files[j].Path })
	body, e := json.Marshal(doc)
	if e != nil {
		return ProjectCapture{}, e
	}
	config, e := domain.NewProjectControlFile(ConfigurationControl, body, true)
	if e != nil {
		return ProjectCapture{}, e
	}
	result.Controls = append(result.Controls, config)
	lock, li, e := readProjectMember(root, LockControl, 4<<20)
	present := true
	if errors.Is(e, os.ErrNotExist) {
		lock, li, e, present = nil, nil, nil, false
	}
	if e != nil {
		return ProjectCapture{}, e
	}
	result.Members[LockControl] = li
	control, e := domain.NewProjectControlFile(LockControl, lock, present)
	if e != nil {
		return ProjectCapture{}, e
	}
	result.Controls = append(result.Controls, control)
	current, e := os.Lstat(directory)
	if e != nil || !os.SameFile(info, current) {
		return ProjectCapture{}, errors.New("terraform root changed during capture")
	}
	return result, nil
}

func parseBody(body []byte, name string) (hcl.Body, error) {
	if len(body) > 1<<20 || !utf8.Valid(body) || !boundedSyntax(body) {
		return nil, errors.New("terraform HCL exceeds syntax bounds")
	}
	parser := hclparse.NewParser()
	var file *hcl.File
	var diags hcl.Diagnostics
	if strings.HasSuffix(name, ".json") {
		file, diags = parser.ParseJSON(body, name)
	} else {
		file, diags = parser.ParseHCL(body, name)
	}
	if diags.HasErrors() || file == nil {
		return nil, errors.New("terraform HCL syntax is invalid")
	}
	return file.Body, nil
}

// This lexical bound runs before the maintained parser. It limits nesting
// without interpreting HCL strings, comments, expressions or identifiers.
func boundedSyntax(body []byte) bool {
	depth := 0
	quoted, escape, lineComment, blockComment := false, false, false, false
	for i := 0; i < len(body); i++ {
		c := body[i]
		if lineComment {
			if c == '\n' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if c == '*' && i+1 < len(body) && body[i+1] == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if quoted {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
			} else if c == '"' {
				quoted = false
			}
			continue
		}
		if c == '"' {
			quoted = true
			continue
		}
		if c == '#' {
			lineComment = true
			continue
		}
		if c == '/' && i+1 < len(body) {
			if body[i+1] == '/' {
				lineComment = true
				i++
				continue
			}
			if body[i+1] == '*' {
				blockComment = true
				i++
				continue
			}
		}
		if c == '{' || c == '[' || c == '(' {
			depth++
			if depth > 64 {
				return false
			}
		}
		if c == '}' || c == ']' || c == ')' {
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0 && !quoted && !blockComment
}

func literalString(expression hcl.Expression) (string, error) {
	v, d := expression.Value(nil)
	if d.HasErrors() || !v.IsKnown() || v.IsNull() || v.Type() != cty.String {
		return "", errors.New("terraform control requires a literal string")
	}
	s := v.AsString()
	if len(s) > 4096 || strings.ContainsAny(s, "\x00\r\n") {
		return "", errors.New("terraform control string exceeds bounds")
	}
	return s, nil
}

func parseConfiguration(files map[string][]byte) ([]Requirement, []string, error) {
	var requirements []Requirement
	var modules []string
	declared := map[string]bool{}
	used := map[string]bool{}
	for name, body := range files {
		b, e := parseBody(body, name)
		if e != nil {
			return nil, nil, e
		}
		content, _, d := b.PartialContent(&hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{Type: "terraform"}, {Type: "module", LabelNames: []string{"name"}}, {Type: "provider", LabelNames: []string{"name"}}, {Type: "resource", LabelNames: []string{"type", "name"}}, {Type: "data", LabelNames: []string{"type", "name"}}}})
		if d.HasErrors() {
			return nil, nil, errors.New("terraform configuration blocks are ambiguous")
		}
		for _, block := range content.Blocks {
			switch block.Type {
			case "terraform":
				inner, _, d := block.Body.PartialContent(&hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{Type: "required_providers"}}})
				if d.HasErrors() {
					return nil, nil, errors.New("terraform required providers block is invalid")
				}
				for _, required := range inner.Blocks {
					attrs, d := required.Body.JustAttributes()
					if d.HasErrors() {
						return nil, nil, errors.New("terraform provider declarations are invalid")
					}
					for local, a := range attrs {
						if !providerSegment.MatchString(local) || declared[local] {
							return nil, nil, errors.New("terraform provider local identity is ambiguous")
						}
						declared[local] = true
						v, d := a.Expr.Value(nil)
						if d.HasErrors() || !v.IsKnown() || v.IsNull() || (!v.Type().IsObjectType() && !v.Type().IsMapType()) {
							return nil, nil, errors.New("terraform provider declaration must be a literal object")
						}
						values := v.AsValueMap()
						for key := range values {
							if key != "source" && key != "version" && key != "configuration_aliases" {
								return nil, nil, errors.New("terraform provider declaration has unsupported fields")
							}
						}
						source, ok := values["source"]
						if !ok || source.IsNull() || source.Type() != cty.String {
							return nil, nil, errors.New("terraform provider requires literal public source")
						}
						address, e := providerAddress(source.AsString())
						if e != nil {
							return nil, nil, e
						}
						r := Requirement{Address: address}
						if constraint, ok := values["version"]; ok {
							if constraint.IsNull() || constraint.Type() != cty.String {
								return nil, nil, errors.New("terraform provider constraint must be literal")
							}
							r.Constraints = append(r.Constraints, constraint.AsString())
						}
						requirements = append(requirements, r)
					}
				}
			case "module":
				attrs, _, d := block.Body.PartialContent(&hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "source", Required: true}}})
				if d.HasErrors() {
					return nil, nil, errors.New("terraform module source is unavailable")
				}
				source, e := literalString(attrs.Attributes["source"].Expr)
				if e != nil {
					return nil, nil, e
				}
				if !strings.HasPrefix(source, "./") && !strings.HasPrefix(source, "../") || strings.ContainsAny(source, "\\:@?\x00") {
					return nil, nil, errors.New("terraform remote module acquisition is unsupported")
				}
				modules = append(modules, source)
			case "provider":
				used[block.Labels[0]] = true
			case "resource", "data":
				local := strings.SplitN(block.Labels[0], "_", 2)[0]
				used[local] = true
			}
		}
	}
	for local := range used {
		if !declared[local] {
			return nil, nil, errors.New("terraform implicit provider selection is unsupported; explicit required_providers is required")
		}
	}
	return requirements, modules, nil
}

func Requirements(controls []domain.ProjectControlFile) ([]Requirement, error) {
	var body []byte
	for _, c := range controls {
		if c.Name() == ConfigurationControl && c.Present() {
			body = c.Body()
		}
	}
	var doc configurationDocument
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&doc) != nil || doc.Schema != 1 || len(doc.Files) == 0 || len(doc.Files) > 256 {
		return nil, errors.New("terraform frozen configuration is invalid")
	}
	canonical, e := json.Marshal(doc)
	if e != nil || !bytes.Equal(canonical, body) {
		return nil, errors.New("terraform frozen configuration is noncanonical")
	}
	byDirectory := map[string]map[string][]byte{}
	for _, file := range doc.Files {
		if path.Clean(file.Path) != file.Path || strings.HasPrefix(file.Path, "../") || strings.HasPrefix(file.Path, "/") {
			return nil, errors.New("terraform frozen configuration path is invalid")
		}
		dir := path.Dir(file.Path)
		if byDirectory[dir] == nil {
			byDirectory[dir] = map[string][]byte{}
		}
		if _, ok := byDirectory[dir][file.Path]; ok {
			return nil, errors.New("terraform configuration path is duplicated")
		}
		byDirectory[dir][file.Path] = file.Body
	}
	combined := map[string]Requirement{}
	for _, files := range byDirectory {
		reqs, _, e := parseConfiguration(files)
		if e != nil {
			return nil, e
		}
		for _, r := range reqs {
			old := combined[r.Address]
			old.Address = r.Address
			old.Constraints = append(old.Constraints, r.Constraints...)
			combined[r.Address] = old
		}
	}
	if len(combined) == 0 || len(combined) > 32 {
		return nil, errors.New("terraform complete provider set is empty or exceeds bounds")
	}
	var result []Requirement
	for _, r := range combined {
		sort.Strings(r.Constraints)
		result = append(result, r)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Address < result[j].Address })
	return result, nil
}

func providerAddress(value string) (string, error) {
	value = strings.TrimPrefix(value, "registry.terraform.io/")
	parts := strings.Split(value, "/")
	if len(parts) != 2 || !providerSegment.MatchString(parts[0]) || !providerSegment.MatchString(parts[1]) {
		return "", errors.New("terraform provider source is outside the public registry")
	}
	return "registry.terraform.io/" + value, nil
}

func ParseLock(body []byte) ([]LockedProvider, error) {
	if len(body) == 0 {
		return nil, nil
	}
	b, e := parseBody(body, LockControl)
	if e != nil {
		return nil, e
	}
	content, d := b.Content(&hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{Type: "provider", LabelNames: []string{"address"}}}})
	if d.HasErrors() || len(content.Blocks) > 32 {
		return nil, errors.New("terraform lock schema is invalid")
	}
	var result []LockedProvider
	seen := map[string]bool{}
	for _, block := range content.Blocks {
		address, e := providerAddress(block.Labels[0])
		if e != nil || address != block.Labels[0] || seen[address] {
			return nil, errors.New("terraform lock provider identity is ambiguous")
		}
		seen[address] = true
		c, d := block.Body.Content(&hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "version", Required: true}, {Name: "constraints"}, {Name: "hashes", Required: true}}})
		if d.HasErrors() {
			return nil, errors.New("terraform lock fields are invalid")
		}
		version, e := literalString(c.Attributes["version"].Expr)
		if e != nil || !validProviderVersion(version) {
			return nil, errors.New("terraform lock version is invalid")
		}
		p := LockedProvider{Address: address, Version: version}
		if a := c.Attributes["constraints"]; a != nil {
			p.Constraints, e = literalString(a.Expr)
			if e != nil {
				return nil, e
			}
		}
		v, d := c.Attributes["hashes"].Expr.Value(nil)
		if d.HasErrors() || !v.IsKnown() || v.IsNull() || (!v.Type().IsTupleType() && !v.Type().IsListType()) || v.LengthInt() == 0 || v.LengthInt() > 64 {
			return nil, errors.New("terraform lock hashes must be a bounded literal list")
		}
		hashes := map[string]bool{}
		for _, v := range v.AsValueSlice() {
			if v.IsNull() || v.Type() != cty.String {
				return nil, errors.New("terraform lock hash is invalid")
			}
			hash := v.AsString()
			if !validLockHash(hash) || hashes[hash] {
				return nil, errors.New("terraform lock hash is invalid or duplicated")
			}
			hashes[hash] = true
			p.Hashes = append(p.Hashes, hash)
		}
		sort.Strings(p.Hashes)
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Address < result[j].Address })
	return result, nil
}

func validLockHash(value string) bool {
	if strings.HasPrefix(value, "zh:") {
		b, e := hex.DecodeString(value[3:])
		return e == nil && len(b) == 32 && strings.ToLower(value) == value
	}
	if strings.HasPrefix(value, "h1:") {
		b, e := base64.StdEncoding.DecodeString(value[3:])
		return e == nil && len(b) == 32 && base64.StdEncoding.EncodeToString(b) == value[3:]
	}
	return false
}

// VerifyPackageLock follows Terraform's matching-hash rule: hashes can describe
// other platforms. At least one supported existing hash must authenticate this
// exact ZIP/content, independently of its signature validation.
func VerifyPackageLock(p LockedProvider, contents PackageContents) error {
	for _, h := range p.Hashes {
		if h == contents.H1 || h == contents.ZH {
			return nil
		}
	}
	return errors.New("terraform package does not match any existing lock hash")
}

func EncodeLock(providers []LockedProvider) ([]byte, error) {
	var b strings.Builder
	b.WriteString("# HAA: independently inspected public provider packages.\n")
	sort.Slice(providers, func(i, j int) bool { return providers[i].Address < providers[j].Address })
	for _, p := range providers {
		if _, e := providerAddress(p.Address); e != nil || !validProviderVersion(p.Version) {
			return nil, errors.New("terraform selected lock identity is invalid")
		}
		fmt.Fprintf(&b, "provider %q {\n  version = %q\n", p.Address, p.Version)
		if p.Constraints != "" {
			fmt.Fprintf(&b, "  constraints = %q\n", p.Constraints)
		}
		b.WriteString("  hashes = [\n")
		sort.Strings(p.Hashes)
		for _, h := range p.Hashes {
			if !validLockHash(h) {
				return nil, errors.New("terraform selected lock hash is invalid")
			}
			fmt.Fprintf(&b, "    %q,\n", h)
		}
		b.WriteString("  ]\n}\n")
	}
	return []byte(b.String()), nil
}

func MatchesConstraint(version, constraint string) bool {
	if !validProviderVersion(version) || len(constraint) > 4096 {
		return false
	}
	if strings.TrimSpace(constraint) == "" {
		return true
	}
	for _, term := range strings.Split(constraint, ",") {
		term = strings.TrimSpace(term)
		op := "="
		for _, candidate := range []string{"~>", ">=", "<=", "!=", ">", "<", "="} {
			if strings.HasPrefix(term, candidate) {
				op = candidate
				term = strings.TrimSpace(strings.TrimPrefix(term, candidate))
				break
			}
		}
		if op == "~>" {
			parts := strings.Split(term, ".")
			if len(parts) != 2 && len(parts) != 3 {
				return false
			}
			low := term
			if len(parts) == 2 {
				low += ".0"
			}
			if !validProviderVersion(low) {
				return false
			}
			v := strings.Split(low, ".")
			index := 0
			if len(parts) == 3 {
				index = 1
			}
			var number uint64
			if _, e := fmt.Sscan(v[index], &number); e != nil || number > 1000000 {
				return false
			}
			v[index] = fmt.Sprint(number + 1)
			for i := index + 1; i < 3; i++ {
				v[i] = "0"
			}
			if semver.Compare("v"+version, "v"+low) < 0 || semver.Compare("v"+version, "v"+strings.Join(v, ".")) >= 0 {
				return false
			}
			continue
		}
		if !validProviderVersion(term) {
			return false
		}
		cmp := semver.Compare("v"+version, "v"+term)
		if op == "=" && cmp != 0 || op == "!=" && cmp == 0 || op == ">" && cmp <= 0 || op == ">=" && cmp < 0 || op == "<" && cmp >= 0 || op == "<=" && cmp > 0 {
			return false
		}
	}
	return true
}
