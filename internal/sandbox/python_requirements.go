package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const pythonRequirementsEnvironmentScript = `environment=default_environment()
marker_names={'implementation_name','implementation_version','os_name','platform_machine','platform_python_implementation','platform_system','python_full_version','python_version','sys_platform','extra'}
def check_marker(node):
 if isinstance(node,(list,tuple)):
  for part in node: check_marker(part)
 elif isinstance(node,Variable) and node.value not in marker_names:
  raise ValueError('requirements target marker is unsupported')
`

// The fixed pip parser consumes only controller-frozen data in this session.
// It never opens referenced host files, expands environment variables, imports
// target packages, or supplies completion/Policy authority.
const pythonRequirementsParseScript = `import hashlib,json,platform,posixpath,sys
import pip
from pip._internal.req.req_file import get_line_parser
from pip._vendor.packaging.requirements import Requirement
from pip._vendor.packaging.markers import default_environment,Variable
from pip._vendor.packaging.utils import canonicalize_name
` + pythonRequirementsEnvironmentScript + `
source_limit,clause_limit,depth_limit=map(int,sys.argv[2:5])
with open(sys.argv[1],'rb') as stream: raw=stream.read(source_limit+1)
if len(raw)>source_limit: raise ValueError('requirements source exceeds bounds')
wire=json.loads(raw)
parser=get_line_parser(None)
clauses=[]
require_hashes=wire['require_hashes']
active=set()
visited=set()

def parse(text,constraint,source,line,file_mode=True):
 text=text.strip()
 global require_hashes
 if '${' in text: raise ValueError('requirements environment expansion is unsupported')
 requirement,hash_options=text,{}
 if file_mode:
  requirement,opts=parser(text)
  if opts.requirements or opts.constraints:
   if requirement or opts.requirements and opts.constraints: raise ValueError('ambiguous requirements include')
   return (opts.requirements or opts.constraints)[0],bool(opts.constraints)
  if getattr(opts,'require_hashes',False): require_hashes=True
  hash_options=opts.hashes or {}
 if not requirement: return None
 req=Requirement(requirement)
 if req.marker: check_marker(req.marker._markers)
 if hash_options: require_hashes=True
 if req.url: raise ValueError('direct requirements source is unsupported')
 enabled=not req.marker or req.marker.evaluate(environment)
 req.marker=None
 req.name=canonicalize_name(req.name)
 req.extras={canonicalize_name(extra) for extra in req.extras}
 hashes={algorithm:sorted(set(value.lower() for value in values)) for algorithm,values in hash_options.items()}
 clauses.append(dict(active=bool(enabled),source=source,line=line,text_sha256=hashlib.sha256(text.encode('utf-8')).hexdigest(),requirement=str(req),project=req.name,extras=sorted(req.extras),constraint=constraint,hashes=hashes))
 if len(clauses)>clause_limit: raise ValueError('requirements clauses exceed bounds')
 return None
def parse_at(text,constraint,source,line,file_mode=True):
 try: return parse(text,constraint,source,line,file_mode)
 except (Exception,SystemExit):
  print(json.dumps(dict(version=1,input_sha256=wire['source_sha256'],parser_sha256=hashlib.sha256(raw).hexdigest(),python=platform.python_version(),pip=pip.__version__,error=dict(source=source,line=line,code='INVALID_REQUIREMENT')),separators=(',',':')))
  sys.exit(0)
def visit(name,constraint,depth):
 if depth>depth_limit or name in active or name not in wire['files']: raise ValueError('requirements closure is invalid')
 if (name,constraint) in visited: return
 active.add(name)
 for record in wire['files'][name] or []:
  include=parse_at(record['text'],constraint,name,record['number'])
  if include:
   target=posixpath.normpath(posixpath.join(posixpath.dirname(name),include[0]))
   if target.startswith('../') or target=='..' or target.startswith('/'): raise ValueError('requirements include escapes input')
   visit(target,include[1],depth+1)
 active.remove(name)
 visited.add((name,constraint))
for source in wire['sources'] or []: visit(source['name'],source['constraint'],1)
for index,text in enumerate(wire['positional'] or [],start=1):
 if parse_at(text,False,'@positional',index,False): raise ValueError('positional include is unsupported')
result=dict(version=1,input_sha256=wire['source_sha256'],parser_sha256=hashlib.sha256(raw).hexdigest(),python=platform.python_version(),pip=pip.__version__,require_hashes=require_hashes,clauses=clauses)
print(json.dumps(result,separators=(',',':')))`

func (r *PyPIResolver) parseRequirementsInSession(ctx context.Context, container string, input artifactpypi.RequirementsInput) (artifactpypi.RequirementsNativeInput, error) {
	if r == nil || r.runner == nil || ctx == nil || container == "" {
		return artifactpypi.RequirementsNativeInput{}, errors.New("requirements native session is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return artifactpypi.RequirementsNativeInput{}, err
	}
	control, err := input.ParserControl()
	if err != nil {
		return artifactpypi.RequirementsNativeInput{}, err
	}
	if err := r.writeUpgradeControl(ctx, container, "requirements-input.json", control.Body()); err != nil {
		return artifactpypi.RequirementsNativeInput{}, err
	}
	body, err := r.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, "python", "-I", "-c", pythonRequirementsParseScript, pypiResolverProjectDir+"/requirements-input.json", strconv.Itoa(artifactpypi.RequirementsInputByteLimit), strconv.Itoa(artifactpypi.RequirementsClauseLimit), strconv.Itoa(artifactpypi.MaximumRequirementsFiles))...)
	if err != nil {
		return artifactpypi.RequirementsNativeInput{}, uvReaderFailure{step: "pinned requirements grammar validation failed", cause: err}
	}
	return input.ParseNativeInput(body)
}

func requirementsOriginalInput(controls []domain.ProjectControlFile, client string) (artifactpypi.RequirementsInput, error) {
	invalid := errors.New("requirements original source and selection intent are incomplete")
	if len(controls) != 2 {
		return artifactpypi.RequirementsInput{}, invalid
	}
	var source domain.ProjectControlFile
	absent := false
	for _, control := range controls {
		switch control.Name() {
		case artifactpypi.RequirementsInputControlName:
			if source.Present() || !control.Present() {
				return artifactpypi.RequirementsInput{}, invalid
			}
			source = control
		case artifactpypi.RequirementsSelectionControlName:
			if absent || control.Present() || len(control.Body()) > 0 {
				return artifactpypi.RequirementsInput{}, invalid
			}
			absent = true
		default:
			return artifactpypi.RequirementsInput{}, invalid
		}
	}
	if !source.Present() || !absent {
		return artifactpypi.RequirementsInput{}, invalid
	}
	input, err := artifactpypi.ParseRequirementsControl(source)
	if err != nil {
		return artifactpypi.RequirementsInput{}, err
	}
	actual, _, _, err := input.Installation()
	if err != nil || actual != client {
		return artifactpypi.RequirementsInput{}, invalid
	}
	return input, nil
}

// ResolveProjectLockUpdate is the existing primary-free port. It holds no
// target mount or publication authority and runs one additive native proposal.
func (r *PyPIResolver) ResolveProjectLockUpdate(ctx context.Context, install domain.InstallContext, original []domain.ProjectControlFile, selector string) (update domain.ProjectDependencyUpdate, resultErr error) {
	if r == nil || ctx == nil || selector != "" || !install.Valid() || install.Mode() != domain.InstallPythonVenv || r.profile.Source() != artifactpypi.PublicPyPIProfile().Source() {
		return update, errors.New("pip requirements need frozen public venv input")
	}
	ctx, cancel := context.WithTimeout(ctx, pypiResolverTimeout)
	defer cancel()
	defer func() {
		if err := ctx.Err(); err != nil {
			update = domain.ProjectDependencyUpdate{}
			resultErr = errors.Join(resultErr, err)
		}
	}()
	input, err := requirementsOriginalInput(original, "pip")
	if err != nil {
		return update, err
	}
	var native requirementsNativeSelection
	err = r.withPythonResolverSourceSession(ctx, r.profile.Source(), func(ctx context.Context, container string, _ PythonRuntime) error {
		var err error
		native, err = r.selectPipRequirements(ctx, container, input)
		return err
	})
	if err != nil {
		return update, err
	}
	return r.bindRequirementsUpdate(ctx, install, original, input, native)
}

func (r *UVResolver) resolveRequirementsUpdate(ctx context.Context, install domain.InstallContext, original []domain.ProjectControlFile) (update domain.ProjectDependencyUpdate, resultErr error) {
	ctx, cancel := context.WithTimeout(ctx, pypiResolverTimeout)
	defer cancel()
	defer func() {
		if err := ctx.Err(); err != nil {
			update = domain.ProjectDependencyUpdate{}
			resultErr = errors.Join(resultErr, err)
		}
	}()
	input, err := requirementsOriginalInput(original, "uv")
	if err != nil {
		return update, err
	}
	// General pip grammar belongs to the existing Python reader profile.
	// The UV session admits the native UV selector and its bounded interpreter
	// queries, not a second arbitrary Python control program.
	var native requirementsNativeSelection
	err = r.reader.withPythonResolverSourceSession(ctx, r.reader.profile.Source(), func(ctx context.Context, container string, _ PythonRuntime) error {
		var err error
		native.input, err = r.reader.parseRequirementsInSession(ctx, container, input)
		return err
	})
	if err != nil {
		return update, err
	}
	if !native.input.HasActiveRoots() {
		return r.reader.bindRequirementsUpdate(ctx, install, original, input, native)
	}
	materialized, err := r.withUVNativeSession(ctx, func(ctx context.Context, session uvNativeSession) (uvNativeOutput, error) {
		var err error
		native, err = r.selectUVRequirements(ctx, input, native.input, session)
		return native.uv, err
	})
	if err != nil {
		return update, err
	}
	native.uv = materialized
	return r.reader.bindRequirementsUpdate(ctx, install, original, input, native)
}

func (r *PyPIResolver) bindRequirementsUpdate(ctx context.Context, install domain.InstallContext, original []domain.ProjectControlFile, input artifactpypi.RequirementsInput, native requirementsNativeSelection) (domain.ProjectDependencyUpdate, error) {
	runtime := PinnedPythonRuntime()
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	client, _, _, err := input.Installation()
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	selectionBody := native.report
	if client == "uv" {
		selectionBody = native.uv.lock
	}
	selection, err := input.NewSelection(native.input.Body(), native.delta, selectionBody, native.uv.controls, target)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	var update domain.ProjectDependencyUpdate
	err = r.withPythonArchiveInputs(ctx, selection.Wheels(), client == "uv", func(ctx context.Context, container string, inputs []artifactpypi.UVWheelInput, pages []artifactpypi.SimpleProject) error {
		bound, err := selection.BindWheels(inputs, pages, target, r.profile.ResourcePolicy().WheelLimits(), func(body []byte) ([]byte, error) {
			return r.validateRequirementsMetadataInSession(ctx, container, body)
		})
		if err != nil {
			return err
		}
		control, err := bound.Control()
		if err != nil {
			return err
		}
		selected := append([]domain.ProjectControlFile(nil), original...)
		for index := range selected {
			if selected[index].Name() == control.Name() {
				selected[index] = control
			}
		}
		snapshot, err := bound.Snapshot(install, selected, target)
		if err != nil {
			return err
		}
		update, err = domain.NewProjectDependencyUpdateFromSnapshot(original, selected, snapshot)
		return err
	})
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	return update, nil
}

func (r *UVResolver) selectUVRequirements(ctx context.Context, source artifactpypi.RequirementsInput, native artifactpypi.RequirementsNativeInput, session uvNativeSession) (requirementsNativeSelection, error) {
	client, upgrade, metadata, err := source.Installation()
	if err != nil || client != "uv" {
		return requirementsNativeSelection{}, errors.New("requirements UV selection intent differs")
	}
	reader, container := r.reader, session.container
	control, err := source.Control()
	if err != nil || !native.Valid() || native.InputDigest() != control.Digest() {
		return requirementsNativeSelection{}, errors.New("UV requirements parser differs from its frozen source")
	}
	result := requirementsNativeSelection{input: native}
	if !native.HasActiveRoots() {
		return result, nil
	}
	roots, constraints, err := native.ResolverFiles()
	if err != nil {
		return requirementsNativeSelection{}, err
	}
	files := map[string][]byte{"requirements.txt": roots}
	for _, file := range metadata {
		files["original/"+file.Directory+"/METADATA"] = file.Body
	}
	if len(constraints) > 0 {
		files["requirements-constraints.txt"] = constraints
	}
	transport := reader.runner.(inputCommandRunner)
	if err := introduceUVSelectionFiles(ctx, transport, container, files, true); err != nil {
		return requirementsNativeSelection{}, err
	}
	common := uvNativeCommonArguments()
	rootArgs := []string{"--requirement", pythonSitePath + "/requirements.txt"}
	if len(constraints) > 0 {
		rootArgs = append(rootArgs, "--constraint", pythonSitePath+"/requirements-constraints.txt")
	}
	dryRun := append([]string{"pip", "install", "--dry-run", "--output-format", "json", "--target", pythonSitePath + "/original"}, common...)
	dryRun = append(dryRun, rootArgs...)
	if upgrade {
		dryRun = append(dryRun, "--upgrade")
	}
	result.delta, err = uvBoundedSelectionOutput(ctx, reader.runner, container, append([]string{uvToolGuestPath}, dryRun...))
	if err != nil {
		return requirementsNativeSelection{}, err
	}
	selected, err := source.SelectionConstraints(native, result.delta)
	if err != nil {
		return requirementsNativeSelection{}, err
	}
	if err := introduceUVSelectionFiles(ctx, transport, container, map[string][]byte{"constraints.txt": selected}, false); err != nil {
		return requirementsNativeSelection{}, err
	}
	files["constraints.txt"] = selected
	compile := append([]string{"pip", "compile", "--format", "pylock.toml", "--output-file", pythonSitePath + "/pylock.toml", "--python-version", session.runtime.PythonVersion, "--python-platform", session.platform}, common...)
	compile = append(compile, pythonSitePath+"/requirements.txt")
	if len(constraints) > 0 {
		compile = append(compile, "--constraint", pythonSitePath+"/requirements-constraints.txt")
	}
	compile = append(compile, "--constraint", pythonSitePath+"/constraints.txt")
	if _, err := reader.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, append([]string{uvToolGuestPath}, compile...)...)...); err != nil {
		return requirementsNativeSelection{}, uvReaderFailure{"complete constrained UV requirements compile failed", err}
	}
	lockBody, err := readUVNativeLock(ctx, reader.runner, container)
	if err != nil {
		return requirementsNativeSelection{}, err
	}
	if _, err := artifactpypi.ParseUVProjectLock(lockBody, session.runtime.PythonVersion); err != nil {
		return requirementsNativeSelection{}, err
	}
	sync := append([]string{"pip", "sync", pythonSitePath + "/pylock.toml", "--target", pythonSitePath + "/site", "--link-mode", "copy"}, common...)
	if _, err := reader.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, append([]string{uvToolGuestPath}, sync...)...)...); err != nil {
		return requirementsNativeSelection{}, uvReaderFailure{"UV requirements materialization failed", err}
	}
	if err := session.output.verifyAttachments(ctx, reader.runner, map[string]bool{container: false}); err != nil {
		return requirementsNativeSelection{}, err
	}
	result.uv, err = readUVNativeTar(ctx, reader.runner, container, pythonSitePath+"/.", func(input io.Reader) (uvNativeOutput, error) {
		return scanUVCompleteNativeOutputWithInput(input, session.runtime.PythonVersion, files)
	})
	if err != nil {
		return requirementsNativeSelection{}, err
	}
	if !bytes.Equal(result.uv.lock, lockBody) {
		return requirementsNativeSelection{}, errors.New("UV requirements lock changed during materialization")
	}
	if err := session.output.verifyAttachments(ctx, reader.runner, map[string]bool{container: false}); err != nil {
		return requirementsNativeSelection{}, err
	}
	return result, nil
}

type requirementsNativeSelection struct {
	input         artifactpypi.RequirementsNativeInput
	delta, report []byte
	uv            uvNativeOutput
}

// The resolver sees only the frozen original distributions. The runtime's own
// pip distribution must not silently satisfy a request absent from the target.
// This uses the same pinned pip internal API as the existing metadata adapter.
const pipRequirementsSelectionScript = pypiUpgradeMetadataScript + `
from pip._internal.metadata import get_environment
from pip._internal.resolution.resolvelib import factory
factory.get_default_environment=lambda:get_environment([str(installed)])
raise SystemExit(main(sys.argv[2:]))`

// All PEP 440/508 checks consume the independently authenticated wheel
// metadata as data. No artifact module, hook or interpreter is imported.
const pythonRequirementsGraphScript = `import hashlib,json,platform,sys
from collections import deque
import pip
from pip._vendor.packaging.requirements import Requirement
from pip._vendor.packaging.specifiers import SpecifierSet
from pip._vendor.packaging.markers import default_environment,Variable
from pip._vendor.packaging.utils import canonicalize_name
` + pythonRequirementsEnvironmentScript + `
limit=int(sys.argv[2])
with open(sys.argv[1],'rb') as stream: raw=stream.read(limit+1)
if len(raw)>limit: raise ValueError('requirements metadata exceeds bounds')
wire=json.loads(raw)
packages={item['name']:item for item in wire['wheels']}
if len(packages)!=len(wire['wheels']): raise ValueError('repeated requirements wheel')
versions={}
constraints=[]
def requirement(text):
 req=Requirement(text)
 if req.url: raise ValueError('direct dependency source is unsupported')
 if req.marker: check_marker(req.marker._markers)
 return req
for text in wire['selected_constraints'].splitlines():
 if not text.strip(): continue
 req=requirement(text)
 constraints.append(req)
 specs=list(req.specifier)
 if len(specs)==1 and specs[0].operator=='==' and '*' not in specs[0].version:
  name=canonicalize_name(req.name)
  if name in versions and versions[name]!=specs[0].version: raise ValueError('conflicting frozen versions')
  versions[name]=specs[0].version
original_list=wire['originals']
originals={item['name']:item for item in original_list}
if len(originals)!=len(original_list): raise ValueError('repeated original metadata')
retained={name:item for name,item in originals.items() if versions.get(name)==item['version']}
all_packages=dict(retained)
all_packages.update(packages)
if set(all_packages)!=set(versions): raise ValueError('native changed version is absent from selected wheels')
for name,item in all_packages.items():
 if versions.get(name)!=item['version']: raise ValueError('metadata differs from native version choice')
 if item['requires_python'] and not SpecifierSet(item['requires_python']).contains(platform.python_version(),prereleases=True): raise ValueError('metadata Python constraint differs')
for req in constraints:
 name=canonicalize_name(req.name)
 if name not in versions or not req.specifier.contains(versions[name],prereleases=True): raise ValueError('retained constraint is unsatisfied')
roots=set()
queue=deque()
visited=set()
edges=set()
hashed_roots=set()
def enqueue(name,extras,selected=True):
 queue.append((name,'',selected))
 for extra in extras: queue.append((name,canonicalize_name(extra),selected))
for clause in wire['clauses']:
 if not clause['active']: continue
 req=requirement(clause['requirement'])
 name=canonicalize_name(req.name)
 if name in packages and not req.specifier.contains(packages[name]['version'],prereleases=True): raise ValueError('requested constraint is unsatisfied')
 if clause['constraint']: continue
 if name not in packages: raise ValueError('requested root is absent')
 roots.add(name)
 enqueue(name,req.extras)
 specs=list(req.specifier)
 if clause['hashes'] and len(specs)==1 and specs[0].operator=='==' and '*' not in specs[0].version: hashed_roots.add(name)
if wire['require_hashes'] and set(packages)!=hashed_roots: raise ValueError('hash mode requires every selected project to be explicitly pinned and hashed')
# Preserve unchanged installed obligations without adding them to the selected
# inspection graph. The same official marker/extra/version rules serve both.
for name in retained: enqueue(name,[],False)
while queue:
 name,extra,selected=queue.popleft()
 if (name,extra,selected) in visited: continue
 visited.add((name,extra,selected))
 environment_packages=packages if selected else all_packages
 item=environment_packages[name]
 declared={canonicalize_name(value) for value in item['provides_extra']}
 if len(declared)!=len(item['provides_extra']) or extra and extra not in declared: raise ValueError('requested extra is absent or ambiguous')
 for text in item['requires_dist']:
  req=requirement(text)
  if req.marker and not req.marker.evaluate(dict(environment,extra=extra)): continue
  dependency=canonicalize_name(req.name)
  if dependency not in environment_packages or not req.specifier.contains(environment_packages[dependency]['version'],prereleases=True): raise ValueError('authenticated dependency is unsatisfied')
  if selected and dependency!=name: edges.add((name,dependency))
  enqueue(dependency,req.extras,selected)
reached={name for name,extra,selected in visited if selected}
if reached!=set(packages): raise ValueError('selected wheel is not reachable from a real root')
result=dict(version=1,input_sha256=hashlib.sha256(raw).hexdigest(),python=platform.python_version(),pip=pip.__version__,roots=sorted(roots),edges=sorted(edges))
print(json.dumps(result,separators=(',',':')))`

func (r *PyPIResolver) validateRequirementsMetadataInSession(ctx context.Context, container string, body []byte) ([]byte, error) {
	if ctx == nil || r == nil || r.runner == nil || !exactObservationContainerID(container) {
		return nil, errors.New("requirements metadata session is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.writeUpgradeControl(ctx, container, "requirements-metadata.json", body); err != nil {
		return nil, err
	}
	result, err := r.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, "python", "-I", "-c", pythonRequirementsGraphScript, pypiResolverProjectDir+"/requirements-metadata.json", strconv.Itoa(artifactpypi.RequirementsInputByteLimit))...)
	if err != nil {
		return nil, uvReaderFailure{"authenticated requirements dependency metadata validation failed", err}
	}
	return result, nil
}

func (r *PyPIResolver) selectPipRequirements(ctx context.Context, container string, source artifactpypi.RequirementsInput) (requirementsNativeSelection, error) {
	client, upgrade, metadata, err := source.Installation()
	if err != nil || client != "pip" {
		return requirementsNativeSelection{}, errors.New("requirements pip selection intent differs")
	}
	native, err := r.parseRequirementsInSession(ctx, container, source)
	if err != nil {
		return requirementsNativeSelection{}, err
	}
	result := requirementsNativeSelection{input: native}
	if !native.HasActiveRoots() {
		return result, nil
	}
	roots, constraints, err := native.ResolverFiles()
	if err != nil {
		return requirementsNativeSelection{}, err
	}
	for _, file := range []struct {
		name string
		body []byte
	}{{"requirements-roots.txt", roots}, {"requirements-constraints.txt", constraints}} {
		if len(file.body) > 0 {
			if err := r.writeUpgradeControl(ctx, container, file.name, file.body); err != nil {
				return requirementsNativeSelection{}, err
			}
		}
	}
	original, err := json.Marshal(struct {
		Installed []artifactpypi.InstalledMetadataFile `json:"installed"`
	}{metadata})
	if err != nil {
		return requirementsNativeSelection{}, err
	}
	// A null installed slice is the explicit empty original environment.
	if metadata == nil {
		original = []byte(`{"installed":[]}`)
	}
	if err := r.writeUpgradeControl(ctx, container, "requirements-original.json", original); err != nil {
		return requirementsNativeSelection{}, err
	}
	arguments := pypiResolveArguments(r.profile, "")
	arguments = append(arguments[:len(arguments)-1], "--only-binary=:all:", "--no-build-isolation", "--requirement", pypiResolverProjectDir+"/requirements-roots.txt")
	if len(constraints) > 0 {
		arguments = append(arguments, "--constraint", pypiResolverProjectDir+"/requirements-constraints.txt")
	}
	first := append([]string(nil), arguments...)
	if upgrade {
		first = append(first, "--upgrade", "--upgrade-strategy", "only-if-needed")
	}
	launch := boundaryExecArguments(container, boundaryLaunchMode, "python", "-I", "-c", pipRequirementsSelectionScript, pypiResolverProjectDir+"/requirements-original.json")
	if _, err := r.runner.Output(ctx, "docker", append(launch, first...)...); err != nil {
		return requirementsNativeSelection{}, uvReaderFailure{"installed-aware pip requirements selection failed", err}
	}
	result.delta, err = r.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, "python", "-I", "-c", boundedReadScript, pypiResolverProjectDir+"/report.json", strconv.Itoa(artifactpypi.RequirementsInputByteLimit))...)
	if err != nil {
		return requirementsNativeSelection{}, uvReaderFailure{"read pip requirements delta failed", err}
	}
	selected, err := source.SelectionConstraints(native, result.delta)
	if err != nil {
		return requirementsNativeSelection{}, err
	}
	if err := r.writeUpgradeControl(ctx, container, "requirements-selected.txt", selected); err != nil {
		return requirementsNativeSelection{}, err
	}
	arguments = append(arguments, "--ignore-installed", "--constraint", pypiResolverProjectDir+"/requirements-selected.txt")
	if _, err := r.runner.Output(ctx, "docker", append(boundaryExecArguments(container, boundaryLaunchMode, "python", "-I", "-m", "pip"), arguments...)...); err != nil {
		return requirementsNativeSelection{}, uvReaderFailure{"complete constrained pip requirements selection failed", err}
	}
	result.report, err = r.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, "python", "-I", "-c", boundedReadScript, pypiResolverProjectDir+"/report.json", strconv.Itoa(artifactpypi.RequirementsInputByteLimit))...)
	if err != nil {
		return requirementsNativeSelection{}, uvReaderFailure{"read complete pip requirements selection failed", err}
	}
	if _, err := native.ParsePipReport(result.report); err != nil {
		return requirementsNativeSelection{}, err
	}
	return result, nil
}
