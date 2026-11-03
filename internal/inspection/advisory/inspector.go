package advisory

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"runtime/debug"
	"strings"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"golang.org/x/mod/semver"
)

// Inspector adds advisory facts to the existing inspection provider. Its batch
// methods retain the provider's static-all, closure and frozen-selection entry
// points. Application still owns the runs, recording and Policy decisions.
type Inspector struct {
	inner   ports.Inspection
	client  *http.Client
	profile domain.ContentDigest
}

func NewInspector(inner ports.Inspection, client *http.Client) (*Inspector, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, errors.New("advisory calculator build identity is unavailable")
	}
	return newInspector(inner, client, info)
}

func newInspector(inner ports.Inspection, client *http.Client, info *debug.BuildInfo) (*Inspector, error) {
	if inner == nil || client == nil || client.Transport == nil || info == nil {
		return nil, errors.New("advisory inspection dependencies are unavailable")
	}
	if _, ok := inner.(ports.CheckPlanner); !ok {
		return nil, errors.New("advisory inspection requires the existing check planner")
	}
	// go.mod remains the only version selector. Read the actual linked module;
	// a missing, replaced or unchecksummed calculator cannot reuse old results.
	identity := ""
	for _, dependency := range info.Deps {
		if dependency.Path != "github.com/pandatix/go-cvss" {
			continue
		}
		sum, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dependency.Sum, "h1:"))
		if identity != "" || dependency.Replace != nil || !semver.IsValid(dependency.Version) || !strings.HasPrefix(dependency.Sum, "h1:") || err != nil || len(sum) != 32 {
			return nil, errors.New("advisory calculator build identity is invalid")
		}
		identity = dependency.Path + "@" + dependency.Version + "|" + dependency.Sum
	}
	if identity == "" {
		return nil, errors.New("advisory calculator build identity is unavailable")
	}
	profile, err := domain.CheckProfileDigest("dependency-advisory-osv-v1", osvEndpoint+"|osv-1.x|base-score|"+identity)
	if err != nil {
		return nil, err
	}
	return &Inspector{inner: inner, client: client, profile: profile}, nil
}

func (i *Inspector) ExpectedChecks(ctx context.Context, artifact domain.AcquiredArtifact, kind domain.CheckKind) ([]domain.CheckRequirement, error) {
	if ctx == nil || i == nil || i.inner == nil || kind != domain.CheckInspection {
		return nil, errors.New("advisory check declaration is unavailable")
	}
	checks, err := i.inner.(ports.CheckPlanner).ExpectedChecks(ctx, artifact, kind)
	if err != nil {
		return nil, err
	}
	id, _ := domain.NewCheckID("dependency-advisory")
	check, err := domain.NewCheckRequirement(id, kind, true, i.profile)
	return append(checks, check), err
}

func (i *Inspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if ctx == nil || i == nil || i.inner == nil {
		return domain.InspectionReport{}, errors.New("advisory inspection is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return domain.InspectionReport{}, err
	}
	if artifact.Digest().String() == "" || artifact.ContentHandle() == "" {
		return domain.InspectionReport{}, errResponse
	}
	report, err := i.inner.Inspect(ctx, artifact)
	if err != nil {
		return report, err
	}
	query, mapped := packageForArtifact(artifact)
	var records [][]osvRecord
	var facts []responseFact
	if mapped {
		records, facts, err = lookupAdvisories(ctx, i.client, []osvPackage{query})
	}
	var selected []osvRecord
	if len(records) == 1 {
		selected = records[0]
	}
	result, reportErr := i.report(artifact, query, mapped, selected, facts, err)
	if reportErr != nil && len(result.Executions()) == 0 {
		return report, errors.Join(err, reportErr)
	}
	combined, combineErr := domain.NewCompositeInspectionReport([]domain.InspectionReport{report, result})
	return combined, errors.Join(err, reportErr, combineErr)
}

func packageForArtifact(artifact domain.AcquiredArtifact) (osvPackage, bool) {
	identity := artifact.Identity()
	query := osvPackage{Name: identity.Name(), Version: identity.Version()}
	var err error
	switch identity.Source().String() {
	case "npm":
		query.Ecosystem = "npm"
		var reference domain.ArtifactReference
		reference, err = artifactnpm.ParseReference(query.Name + "@" + query.Version)
		if err == nil {
			var name, spec string
			name, err = artifactnpm.RequestedPackageName(reference)
			if err == nil {
				spec, err = artifactnpm.RequestedPackageSpec(reference)
			}
			if name != query.Name || spec != query.Version {
				return osvPackage{}, false
			}
		}
	case "pypi":
		query.Ecosystem = "PyPI"
		var name, version string
		name, err = artifactpypi.NormalizeProjectName(query.Name)
		if err == nil {
			version, err = artifactpypi.NormalizeVersion(query.Version)
		}
		if name != query.Name || version != query.Version {
			return osvPackage{}, false
		}
	case artifactgo.Source().String():
		query.Ecosystem = "Go"
		_, err = artifactgo.ParseReference(query.Name + "@" + query.Version)
	case artifactcargo.Source().String():
		query.Ecosystem = "crates.io"
		_, err = artifactcargo.ParseReference(query.Name + "@" + query.Version)
	default:
		return osvPackage{}, false
	}
	if err != nil || !validPackage(query) {
		return osvPackage{}, false
	}
	if _, err := compareVersions(query.Ecosystem, query.Version, query.Version); err != nil {
		return osvPackage{}, false
	}
	return query, true
}

func (i *Inspector) InspectGraph(ctx context.Context, graph domain.LockedDependencyGraph, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if ctx == nil || i == nil || i.inner == nil {
		return nil, errors.New("advisory graph inspection is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if graph.Primary().String() == "" {
		return nil, errResponse
	}
	if err := validateAcquired(graph.Nodes(), acquired); err != nil {
		return nil, err
	}
	var reports map[domain.DependencyNodeID]domain.InspectionReport
	var err error
	if inner, ok := i.inner.(interface {
		InspectGraph(context.Context, domain.LockedDependencyGraph, map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error)
	}); ok {
		reports, err = inner.InspectGraph(ctx, graph, acquired)
	} else {
		reports, err = i.inspectEntries(ctx, graph.Nodes(), acquired)
	}
	if err != nil {
		return reports, err
	}
	return i.completeBatch(ctx, graph.Nodes(), acquired, reports)
}

func (i *Inspector) InspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	return i.inspectProject(ctx, snapshot, nil, acquired)
}

func (i *Inspector) InspectProjectSelection(ctx context.Context, update domain.ProjectDependencyUpdate, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if !update.Valid() {
		return nil, errors.New("advisory inspection needs the frozen project selection")
	}
	return i.inspectProject(ctx, update.Snapshot(), &update, acquired)
}

func (i *Inspector) inspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot, selection *domain.ProjectDependencyUpdate, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if ctx == nil || i == nil || i.inner == nil || !snapshot.Valid() {
		return nil, errors.New("advisory project inspection is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	nodes := make([]domain.LockedDependency, 0, len(snapshot.Dependencies()))
	for _, artifact := range snapshot.Dependencies() {
		node, err := domain.ProjectDependencyNodeID(artifact.Identity())
		if err != nil {
			return nil, err
		}
		dependency, err := domain.NewLockedDependency(node, domain.DependencyTransitive, artifact)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, dependency)
	}
	if err := validateAcquired(nodes, acquired); err != nil {
		return nil, err
	}
	var reports map[domain.DependencyNodeID]domain.InspectionReport
	var err error
	if inner, ok := i.inner.(interface {
		InspectProjectSelection(context.Context, domain.ProjectDependencyUpdate, map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error)
	}); ok && selection != nil {
		reports, err = inner.InspectProjectSelection(ctx, *selection, acquired)
	} else if inner, ok := i.inner.(interface {
		InspectProject(context.Context, domain.ProjectDependencySnapshot, map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error)
	}); ok {
		reports, err = inner.InspectProject(ctx, snapshot, acquired)
	} else {
		reports, err = i.inspectEntries(ctx, nodes, acquired)
	}
	if err != nil {
		return reports, err
	}
	return i.completeBatch(ctx, nodes, acquired, reports)
}

func (i *Inspector) InspectExecutionSelection(ctx context.Context, selection domain.ExecutionDependencySelection, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if ctx == nil || i == nil || i.inner == nil || !selection.Valid() {
		return nil, errors.New("advisory execution selection is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateAcquired(selection.Nodes(), acquired); err != nil {
		return nil, err
	}
	var reports map[domain.DependencyNodeID]domain.InspectionReport
	var err error
	if inner, ok := i.inner.(interface {
		InspectExecutionSelection(context.Context, domain.ExecutionDependencySelection, map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error)
	}); ok {
		reports, err = inner.InspectExecutionSelection(ctx, selection, acquired)
	} else {
		return nil, errors.New("original inspector does not support execution selection")
	}
	if err != nil {
		return reports, err
	}
	return i.completeBatch(ctx, selection.Nodes(), acquired, reports)
}

func (i *Inspector) inspectEntries(ctx context.Context, nodes []domain.LockedDependency, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	reports := make(map[domain.DependencyNodeID]domain.InspectionReport, len(nodes))
	for _, node := range nodes {
		report, err := i.inner.Inspect(ctx, acquired[node.Node()])
		if len(report.Executions()) != 0 {
			reports[node.Node()] = report
		}
		if err != nil {
			return reports, err
		}
	}
	return reports, nil
}

func (i *Inspector) completeBatch(ctx context.Context, nodes []domain.LockedDependency, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact, reports map[domain.DependencyNodeID]domain.InspectionReport) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if len(reports) != len(nodes) {
		return reports, errResponse
	}
	queries := make([]osvPackage, 0, len(nodes))
	indices := make(map[domain.DependencyNodeID]int, len(nodes))
	for _, node := range nodes {
		artifact, ok := acquired[node.Node()]
		if !ok || len(reports[node.Node()].Executions()) == 0 {
			return reports, errResponse
		}
		if query, ok := packageForArtifact(artifact); ok {
			indices[node.Node()] = len(queries)
			queries = append(queries, query)
		}
	}
	var records [][]osvRecord
	var facts []responseFact
	var lookupErr error
	if len(queries) != 0 {
		records, facts, lookupErr = lookupAdvisories(ctx, i.client, queries)
	}
	var resultErr error
	for _, node := range nodes {
		index, mapped := indices[node.Node()]
		var query osvPackage
		var selected []osvRecord
		if mapped {
			query = queries[index]
			if lookupErr == nil {
				selected = records[index]
			}
		}
		additional, err := i.report(acquired[node.Node()], query, mapped, selected, facts, lookupErr)
		if err != nil {
			resultErr = errors.Join(resultErr, err)
			if len(additional.Executions()) == 0 {
				continue
			}
		}
		combined, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{reports[node.Node()], additional})
		if err != nil {
			resultErr = errors.Join(resultErr, err)
			continue
		}
		reports[node.Node()] = combined
	}
	return reports, errors.Join(lookupErr, resultErr)
}

// Validate the complete frozen membership before invoking the wrapped provider.
func validateAcquired(nodes []domain.LockedDependency, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) error {
	if len(nodes) > maxLookupPackages || len(acquired) != len(nodes) {
		return errResponse
	}
	for _, node := range nodes {
		artifact, ok := acquired[node.Node()]
		declared, present := artifact.DeclaredIntegrity()
		if !ok || artifact.Identity() != node.Artifact().Identity() || artifact.Digest().String() == "" || artifact.ContentHandle() == "" || !present || declared != node.Artifact().DeclaredIntegrity() {
			return errResponse
		}
	}
	return nil
}
