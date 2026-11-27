package pypi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"maps"
	"slices"
	"sync"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type wheelInspectionScopeKey struct{}

type wheelInspectionKey struct {
	size             int64
	filename, digest string
	target           WheelTarget
	limits           WheelLimits
	source           domain.SourceID
}

type wheelInspectionScope struct {
	mu          sync.Mutex
	maxWheels   int
	inspections map[wheelInspectionKey]WheelInspection
}

// WithWheelInspectionScope lets one graph inspection reuse successful parsing
// across its static checks and observation closures.
func WithWheelInspectionScope(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}
	return context.WithValue(ctx, wheelInspectionScopeKey{}, &wheelInspectionScope{
		maxWheels:   ResourcePolicyFromContext(ctx).MaxGraphArtifacts(),
		inspections: make(map[wheelInspectionKey]WheelInspection),
	})
}

// InspectWheelForSourceInScope always verifies the supplied bytes. A matching
// result reuses archive parsing within the current graph after authentication.
func InspectWheelForSourceInScope(ctx context.Context, reader io.ReaderAt, size int64, filename, declaredSHA256 string, target WheelTarget, limits WheelLimits, source domain.SourceID) (WheelInspection, error) {
	if ctx == nil {
		return InspectWheelForSource(reader, size, filename, declaredSHA256, target, limits, source)
	}
	scope, ok := ctx.Value(wheelInspectionScopeKey{}).(*wheelInspectionScope)
	if !ok {
		return InspectWheelForSource(reader, size, filename, declaredSHA256, target, limits, source)
	}
	key := wheelInspectionKey{
		size: size, filename: filename, digest: declaredSHA256, target: target, limits: limits, source: source,
	}
	scope.mu.Lock()
	cached, found := scope.inspections[key]
	scope.mu.Unlock()
	if found {
		if reader == nil {
			return WheelInspection{}, wheelValidation(WheelValidationOther)
		}
		if _, err := verifyWheelSHA256(reader, size, declaredSHA256); err != nil {
			return WheelInspection{}, err
		}
		return cloneWheelInspection(cached), nil
	}
	info, err := InspectWheelForSource(reader, size, filename, declaredSHA256, target, limits, source)
	if err != nil {
		return WheelInspection{}, err
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if len(scope.inspections) < scope.maxWheels {
		scope.inspections[key] = cloneWheelInspection(info)
	}
	return info, nil
}

func verifyWheelSHA256(reader io.ReaderAt, size int64, declared string) (string, error) {
	hash := sha256.New()
	if err := hashReaderAt(reader, size, hash); err != nil {
		return "", wheelValidation(WheelValidationDigest)
	}
	observed := hex.EncodeToString(hash.Sum(nil))
	if observed != declared {
		return "", wheelValidation(WheelValidationDigest)
	}
	return observed, nil
}

func cloneWheelInspection(info WheelInspection) WheelInspection {
	info.PythonTags = slices.Clone(info.PythonTags)
	info.ABITags = slices.Clone(info.ABITags)
	info.PlatformTags = slices.Clone(info.PlatformTags)
	info.Tags = slices.Clone(info.Tags)
	info.Files = slices.Clone(info.Files)
	info.RequiresDist = slices.Clone(info.RequiresDist)
	info.ImportNames = slices.Clone(info.ImportNames)
	info.providesExtra = slices.Clone(info.providesExtra)
	info.EntryPoints = slices.Clone(info.EntryPoints)
	info.Scripts = slices.Clone(info.Scripts)
	info.NativeExtensions = slices.Clone(info.NativeExtensions)

	surface := &info.Surface
	surface.InstalledFiles = slices.Clone(surface.InstalledFiles)
	surface.SiteStartupHooks = slices.Clone(surface.SiteStartupHooks)
	surface.SiteHookLines = slices.Clone(surface.SiteHookLines)
	surface.PythonModules = slices.Clone(surface.PythonModules)
	surface.PythonPackages = slices.Clone(surface.PythonPackages)
	surface.NamespaceContainers = slices.Clone(surface.NamespaceContainers)
	surface.ExecutableSubpackages = slices.Clone(surface.ExecutableSubpackages)
	surface.PythonExtensions = slices.Clone(surface.PythonExtensions)
	surface.NativeLibraries = slices.Clone(surface.NativeLibraries)
	surface.Scripts = slices.Clone(surface.Scripts)
	surface.EntryPoints = slices.Clone(surface.EntryPoints)
	surface.EntryPointDetails = slices.Clone(surface.EntryPointDetails)
	for i := range surface.EntryPointDetails {
		surface.EntryPointDetails[i].Extras = slices.Clone(surface.EntryPointDetails[i].Extras)
	}
	surface.EntryPointCoverage = maps.Clone(surface.EntryPointCoverage)
	surface.ScriptCoverage = maps.Clone(surface.ScriptCoverage)
	surface.ResourceFiles = slices.Clone(surface.ResourceFiles)
	surface.UnresolvedFiles = slices.Clone(surface.UnresolvedFiles)
	return info
}
