package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// An acquired release is observed from its frozen bytes, independently of the
// currently installed runtime. This input grants neither completion nor Policy.
type pipxObservationInput struct {
	artifact domain.AcquiredArtifact
	tool     *pipxPreparedTool
	units    []observationUnit
}

func preparePipxObservationInput(ctx context.Context, artifact domain.AcquiredArtifact, plan artifactpypi.PipxRuntimePlan) (pipxObservationInput, error) {
	if ctx == nil {
		return pipxObservationInput{}, errors.New("pipx observation context is missing")
	}
	if err := ctx.Err(); err != nil {
		return pipxObservationInput{}, err
	}
	id := artifact.Identity()
	if !plan.Valid() || id.Source().String() != "github-release" || id.Name() != "rahoney-heliopause" || id.Version() == "" || id.Variant() != runtimeidentity.PipxRuntimeAssetName || artifact.Digest().String() != runtimeidentity.PipxToolSHA256 || artifact.SizeBytes() != runtimeidentity.PipxToolSize || artifact.ContentHandle() == "" {
		return pipxObservationInput{}, errors.New("pipx observation subject differs from selected acquired ZIP")
	}
	body := plan.ZIP().Bytes()
	digest := sha256.Sum256(body)
	if len(body) != runtimeidentity.PipxToolSize || hex.EncodeToString(digest[:]) != artifact.Digest().String() {
		return pipxObservationInput{}, errPipxToolSupply
	}
	units, err := freezePipxObservationUnits(artifact, plan.Units())
	if err != nil {
		return pipxObservationInput{}, err
	}
	if err := ctx.Err(); err != nil {
		return pipxObservationInput{}, err
	}
	return pipxObservationInput{artifact: artifact, tool: &pipxPreparedTool{body: body}, units: units}, nil
}

func freezePipxObservationUnits(artifact domain.AcquiredArtifact, declared []artifactpypi.PlannedObservationUnit) ([]observationUnit, error) {
	if len(declared) == 0 || len(declared) > artifactpypi.PublicPyPIProfile().ResourcePolicy().MaxObservationImportsPerArtifact() {
		return nil, errors.New("pipx observation unit count is invalid")
	}
	id := artifact.Identity()
	binding := id.Source().String() + ":" + id.Name() + ":" + id.Version() + ":" + id.Variant() + ":" + artifact.Digest().String()
	seen := map[string]bool{}
	units := make([]observationUnit, 0, len(declared))
	for index, d := range declared {
		if d.Kind != artifactpypi.DirectImportUnit || !pythonImportName.MatchString(d.Candidate) || seen[d.Candidate] || (d.Coverage != artifactpypi.RequiredObservation && d.Coverage != artifactpypi.PostInstallCommandObservation) || d.HookFile != "" || d.HookLine != 0 || d.Statement != "" {
			return nil, errors.New("pipx observation unit differs from direct import plan")
		}
		seen[d.Candidate] = true
		unit := observationUnit{kind: observationDirectImport, candidate: d.Candidate, program: pipxDirectObservation, ownerDigest: artifact.Digest().String(), coverage: d.Coverage}
		unit.id = observationUnitIdentity(binding, index, unit)
		units = append(units, unit)
	}
	return units, nil
}

// The same fixed program is frozen into each unit before artifact execution.
// No artifact output or interpreter state can create a completed ledger entry.
const pipxDirectObservation = `import importlib, importlib.machinery, os, sys
version = '` + runtimeidentity.PythonVersion + `'
minor = version.split('.')[:2]
stdlib = '/usr/local/lib/python' + '.'.join(minor)
expected = ['/usr/local/lib/python' + ''.join(minor) + '.zip', stdlib, stdlib + '/lib-dynload']
if sys.version.split()[0] != version or not (sys.flags.isolated and sys.flags.no_site and sys.flags.dont_write_bytecode) or 'site' in sys.modules or sys.path != expected:
    raise RuntimeError('pipx interpreter startup differs from pinned isolated runtime')
root = '/haa-pipx/pipx-runtime.zip'
name = sys.argv[1]
parts = name.split('.')
if not parts or any(not part.isidentifier() for part in parts) or parts[0] in sys.modules:
    raise RuntimeError('invalid or preloaded pipx target import identity')
search = [root]
for depth in range(len(parts)):
    spec = importlib.machinery.PathFinder.find_spec(parts[depth], search)
    if spec is None or spec.origin is None or not spec.origin.startswith(root + '/') or os.path.normpath(spec.origin) != spec.origin:
        raise RuntimeError('target origin outside original pipx ZIP')
    locations = spec.submodule_search_locations
    if locations is not None and (not locations or any(not p.startswith(root + '/') or os.path.normpath(p) != p for p in locations)):
        raise RuntimeError('target package outside original pipx ZIP')
    if depth + 1 < len(parts):
        if locations is None:
            raise RuntimeError('pipx target parent is not a package')
        search = locations
sys.path.append(root)
importlib.import_module(name)
`
