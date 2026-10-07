// Package terraformprovider normalizes bounded static package checks and the
// isolated provider help probe. Provider RPC/cloud behavior is not attested.
package terraformprovider

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

type Inspector struct {
	intake       string
	sandbox      ports.Sandbox
	verification ports.Verification
}

func NewInspector(intake string, sandbox ports.Sandbox, verification ports.Verification) (*Inspector, error) {
	if sandbox == nil || verification == nil || !filepath.IsAbs(intake) || filepath.Clean(intake) != intake || intake == "/" {
		return nil, errors.New("terraform inspection requires private intake and isolated sandbox")
	}
	return &Inspector{intake, sandbox, verification}, nil
}

func (i *Inspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if i == nil || ctx == nil || ctx.Err() != nil {
		return domain.InspectionReport{}, errors.New("terraform inspection request is invalid")
	}
	bundle, e := artifactterraform.ReadIntake(i.intake, artifact)
	if e != nil {
		return domain.InspectionReport{}, e
	}
	ref, e := artifactterraform.IdentityReference(artifact.Identity())
	if e != nil {
		return domain.InspectionReport{}, e
	}
	contents, archiveErr := artifactterraform.InspectPackage(ctx, bundle, ref.Locator())
	if ctx.Err() != nil {
		return domain.InspectionReport{}, ctx.Err()
	}
	checkID, _ := domain.NewCheckID("terraform-provider-static")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	id, _ := domain.NewEvidenceID("terraform-provider-static-result")
	summary := "Provider ZIP rejected by bounded path/type/size/CRC/ELF inspection."
	if archiveErr == nil {
		summary = fmt.Sprintf("Bounded complete ZIP/CRC and Linux amd64 ELF checked; h1=%s; zh=%s; executable SHA256=%s.", contents.H1, contents.ZH, contents.ExecutableDigest)
	}
	evidence, e := domain.NewEvidence(id, checkID, artifact.Identity(), artifact.Digest(), "terraform-provider-static", summary)
	if e != nil {
		return domain.InspectionReport{}, e
	}
	var findings []domain.Finding
	if archiveErr != nil {
		f, _ := domain.NewFinding("M12_TERRAFORM_ARCHIVE_INVALID", []domain.EvidenceID{id})
		findings = append(findings, f)
	}
	static, e := domain.NewInspectionReport(check, findings, []domain.Evidence{evidence})
	if e != nil || archiveErr != nil {
		return static, e
	}
	verified, e := i.verification.Verify(ctx, artifact)
	if e != nil {
		return domain.InspectionReport{}, e
	}
	if verified.Outcome() != domain.VerificationVerified || verified.Execution().Status() != domain.ExecutionCompleted {
		dynamicID, _ := domain.NewCheckID("terraform-provider-dynamic")
		stopped, _ := domain.NewCheckExecution(dynamicID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, "M12_TERRAFORM_SOURCE_NOT_VERIFIED")
		dynamic, e := domain.NewInspectionReport(stopped, nil, nil)
		if e != nil {
			return domain.InspectionReport{}, e
		}
		return domain.NewCompositeInspectionReport([]domain.InspectionReport{static, dynamic})
	}
	request, e := domain.NewSandboxRequest(artifact)
	if e != nil {
		return domain.InspectionReport{}, e
	}
	result, e := i.sandbox.Execute(ctx, request)
	if e != nil {
		return domain.InspectionReport{}, e
	}
	dynamic, e := providerProbeReports(artifact, result)
	if e != nil {
		return domain.InspectionReport{}, e
	}
	return domain.NewCompositeInspectionReport(append([]domain.InspectionReport{static}, dynamic...))
}

func providerProbeReports(artifact domain.AcquiredArtifact, result domain.SandboxResult) ([]domain.InspectionReport, error) {
	checkID, _ := domain.NewCheckID("terraform-provider-dynamic")
	capability, status, limitation := domain.CapabilitySupported, domain.ExecutionCompleted, ""
	if result.Status() != domain.SandboxCompleted {
		limitation, _ = result.LimitationCode()
		status = domain.ExecutionIncomplete
		if limitation == "M3_LINUX_ONLY" || limitation == "M3_RUNTIME_UNAVAILABLE" || limitation == "M3_RUNTIME_VERSION_UNSUPPORTED" || limitation == "M3_IMAGE_UNAVAILABLE" {
			capability, status = domain.CapabilityUnsupported, domain.ExecutionNotExecuted
		}
	}
	terminal, lifecycle := 0, 0
	for _, o := range result.Observations() {
		if o.Category() == domain.ObservationResource && status == domain.ExecutionCompleted {
			status, limitation = domain.ExecutionIncomplete, "M12_TERRAFORM_PROBE_RESOURCE_LIMIT"
		}
		if o.Category() == domain.ObservationProcess {
			if o.Subject() == "terraform-help-exit-0" || o.Subject() == "terraform-help-exit-1" {
				terminal += int(o.Count())
			}
			if o.Subject() == "lifecycle-completed" {
				lifecycle += int(o.Count())
			}
		}
	}
	if status == domain.ExecutionCompleted && (terminal != 1 || lifecycle != 1) {
		status, limitation = domain.ExecutionIncomplete, "M12_TERRAFORM_PROBE_TERMINAL_UNPROVEN"
	}
	summary, e := result.ObservationSummary()
	if e != nil {
		return nil, e
	}
	check, e := domain.NewCheckExecution(checkID, domain.CheckInspection, true, capability, status, limitation)
	if e != nil {
		return nil, e
	}
	id, _ := domain.NewEvidenceID("terraform-provider-dynamic-result")
	ev, e := domain.NewEvidence(id, checkID, artifact.Identity(), artifact.Digest(), "terraform-provider-dynamic", summary)
	if e != nil {
		return nil, e
	}
	seen := map[string]bool{}
	var findings []domain.Finding
	for _, o := range result.Observations() {
		code := ""
		switch o.Category() {
		case domain.ObservationHoneytoken:
			code = "M3_HONEYTOKEN_ACCESS"
		case domain.ObservationNetwork:
			code = "M3_NETWORK_ATTEMPT"
		case domain.ObservationFilesystem:
			if o.Subject() == "filesystem-violation" || o.Subject() == "filesystem-outside-workspace" {
				code = "M3_FILESYSTEM_VIOLATION"
			}
		case domain.ObservationProcess:
			if o.Subject() == "process-unexpected" || o.Subject() == "process-exec-unexpected" {
				code = "M3_UNEXPECTED_PROCESS"
			}
		}
		if code != "" && !seen[code] {
			f, _ := domain.NewFinding(code, []domain.EvidenceID{id})
			findings = append(findings, f)
			seen[code] = true
		}
	}
	if status != domain.ExecutionCompleted && len(findings) != 0 {
		// Recorded suspicious facts are a completed partial observation. They
		// do not complete the independent required probe coverage check.
		observedID, _ := domain.NewCheckID("terraform-provider-observed-facts")
		observed, _ := domain.NewCheckExecution(observedID, domain.CheckInspection, false, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		partialEvidence, err := domain.NewEvidence(id, observedID, artifact.Identity(), artifact.Digest(), "terraform-provider-dynamic", summary)
		if err != nil {
			return nil, err
		}
		partial, err := domain.NewInspectionReport(observed, findings, []domain.Evidence{partialEvidence})
		if err != nil {
			return nil, err
		}
		coverage, err := domain.NewInspectionReport(check, nil, nil)
		if err != nil {
			return nil, err
		}
		return []domain.InspectionReport{partial, coverage}, nil
	}
	report, err := domain.NewInspectionReport(check, findings, []domain.Evidence{ev})
	if err != nil {
		return nil, err
	}
	return []domain.InspectionReport{report}, nil
}
