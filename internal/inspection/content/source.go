package content

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// SourceChecks declares coverage before execution. Signals have their own
// completed check so a valid partial parse can retain findings while required
// coverage remains INCOMPLETE under the existing Domain contract.
func SourceChecks() ([]domain.CheckRequirement, error) {
	profile, err := sourceProfile()
	if err != nil {
		return nil, err
	}
	var checks []domain.CheckRequirement
	for _, spec := range []struct {
		name     string
		required bool
	}{{"source-content", true}, {"source-content-signals", false}} {
		id, err := domain.NewCheckID(spec.name)
		if err != nil {
			return nil, err
		}
		check, err := domain.NewCheckRequirement(id, domain.CheckInspection, spec.required, profile)
		if err != nil {
			return nil, err
		}
		checks = append(checks, check)
	}
	return checks, nil
}

func sourceProfile() (domain.ContentDigest, error) {
	pin := runtimeidentity.SelectedSemgrepSupply()
	return domain.CheckProfileDigest("source-content-v1", pin.Version+"|"+pin.Tool.SHA256+"|"+pin.Rules.SHA256)
}

// InspectSources consumes only archive-checked source DATA. It does not read an
// artifact again, execute its code or make a Policy decision. The existing
// ecosystem inventory owns applicability and the other content capabilities.
func InspectSources(ctx context.Context, analyzer ports.SourceAnalysis, artifact domain.AcquiredArtifact, files []ports.SourceAnalysisFile, inventories ...Inventory) (domain.InspectionReport, error) {
	if ctx == nil || artifact.Digest().String() == "" || len(inventories) > 1 {
		return domain.InspectionReport{}, errors.New("source inspection dependencies or subject are unavailable")
	}
	if err := ctx.Err(); err != nil {
		return domain.InspectionReport{}, err
	}
	noSource, inventoryComplete := false, true
	inventoryDigest, unsupportedSources := "", 0
	if len(inventories) == 1 {
		inventory := inventories[0]
		if err := inventory.validateSources(artifact, files); err != nil {
			return domain.InspectionReport{}, err
		}
		inventoryDigest, unsupportedSources = inventory.digest, inventory.unsupported
		inventoryComplete = unsupportedSources == 0
		noSource = inventory.sources == 0 && inventoryComplete
	}
	if analyzer == nil && !noSource {
		return domain.InspectionReport{}, errors.New("source analyzer is unavailable")
	}
	profile, err := sourceProfile()
	if err != nil {
		return domain.InspectionReport{}, err
	}
	input := ports.SourceAnalysisInput{Artifact: artifact, Files: files}
	var output ports.SourceAnalysisOutput
	var runtimeErr error
	if len(files) > 0 {
		output, runtimeErr = analyzer.Analyze(ctx, input)
	}
	if cancelled := ctx.Err(); cancelled != nil {
		return domain.InspectionReport{}, errors.Join(runtimeErr, cancelled)
	}
	var scan sourceScan
	var decodeErr error
	if noSource {
		scan.complete = true
	} else {
		scan, decodeErr = decodeSourceScan(input, output)
	}
	valid := runtimeErr == nil && decodeErr == nil
	if !valid {
		scan = sourceScan{}
	}
	coverageID, _ := domain.NewCheckID("source-content")
	signalsID, _ := domain.NewCheckID("source-content-signals")
	status, limitation := domain.ExecutionCompleted, ""
	if !valid || !scan.complete || !inventoryComplete {
		status, limitation = domain.ExecutionIncomplete, "M13_SOURCE_ANALYSIS_INCOMPLETE"
		if !inventoryComplete {
			limitation = "M13_SOURCE_UNSUPPORTED_CONTENT"
		}
	}
	coverage, err := domain.NewCheckExecution(coverageID, domain.CheckInspection, true, domain.CapabilitySupported, status, limitation, profile)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	signalStatus, signalLimit := domain.ExecutionCompleted, ""
	if !valid {
		signalStatus, signalLimit = domain.ExecutionNotExecuted, "M13_SOURCE_SIGNALS_UNAVAILABLE"
	}
	signals, err := domain.NewCheckExecution(signalsID, domain.CheckInspection, false, domain.CapabilitySupported, signalStatus, signalLimit, profile)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	// File names and source content never enter persisted Evidence. Hash the
	// controller's ordered inventory once; matching again does not rehash bodies.
	contentHashes := make(map[string]string, len(files))
	pathHashes := make(map[string]string, len(files))
	inventory := sha256.New()
	scannedInventory, unscannedInventory := sha256.New(), sha256.New()
	for _, file := range files {
		sum := sha256.Sum256(file.Body)
		nameSum := sha256.Sum256([]byte(file.Path))
		contentHashes[file.Path] = hex.EncodeToString(sum[:])
		pathHashes[file.Path] = hex.EncodeToString(nameSum[:])
		if err := json.NewEncoder(inventory).Encode([]any{file.Path, file.Language, contentHashes[file.Path], len(file.Body)}); err != nil {
			return domain.InspectionReport{}, err
		}
		if valid {
			destination := unscannedInventory
			if scan.scannedFiles[file.Path] {
				destination = scannedInventory
			}
			if err := json.NewEncoder(destination).Encode([]any{file.Path, file.Language, contentHashes[file.Path], len(file.Body)}); err != nil {
				return domain.InspectionReport{}, err
			}
		}
	}
	scannedDigest, unscannedDigest := "", ""
	if valid {
		scannedDigest, unscannedDigest = hex.EncodeToString(scannedInventory.Sum(nil)), hex.EncodeToString(unscannedInventory.Sum(nil))
	}
	pin := runtimeidentity.SelectedSemgrepSupply()
	runtimeCode, _ := output.Runtime.LimitationCode()
	summary, err := json.Marshal(struct {
		Schema             string `json:"schema"`
		Engine             string `json:"engine"`
		Tool               string `json:"tool_sha256"`
		Rules              string `json:"rules_sha256"`
		Inventory          string `json:"source_inventory_sha256"`
		ScannedInventory   string `json:"scanned_inventory_sha256,omitempty"`
		UnscannedInventory string `json:"unscanned_inventory_sha256,omitempty"`
		Scope              string `json:"scope"`
		Targets            int    `json:"targets"`
		Scanned            int    `json:"scanned"`
		Errors             int    `json:"parse_errors"`
		Skipped            int    `json:"skipped"`
		Timeouts           int    `json:"fixpoint_timeouts"`
		Matches            int    `json:"matches"`
		RuntimeComplete    bool   `json:"runtime_complete"`
		ReportValid        bool   `json:"report_valid"`
		CoverageComplete   bool   `json:"coverage_complete"`
		RuntimeLimitation  string `json:"runtime_limitation,omitempty"`
		ArchiveInventory   string `json:"archive_inventory_sha256,omitempty"`
		UnsupportedSources int    `json:"unsupported_sources,omitempty"`
		NoApplicableSource bool   `json:"no_applicable_source,omitempty"`
	}{"source-content-v1", pin.Version, pin.Tool.SHA256, pin.Rules.SHA256, hex.EncodeToString(inventory.Sum(nil)), scannedDigest, unscannedDigest, "same-file-utf8-ast", len(files), scan.scanned, scan.errors, scan.skipped, scan.timeouts, scan.matches, output.Runtime.Status() == domain.SandboxCompleted, valid && !noSource, valid && scan.complete && inventoryComplete, runtimeCode, inventoryDigest, unsupportedSources, noSource})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	coverageEvidenceID, _ := domain.NewEvidenceID("source-content-coverage")
	fact, err := domain.NewEvidence(coverageEvidenceID, coverageID, artifact.Identity(), artifact.Digest(), "source-content-coverage", string(summary), profile)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	coverageReport, err := domain.NewInspectionReport(coverage, nil, []domain.Evidence{fact})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	signalSummary, err := json.Marshal(struct {
		Valid              bool `json:"report_valid"`
		Matched            int  `json:"matched"`
		Retained           int  `json:"retained"`
		Limit              int  `json:"retained_limit"`
		NoApplicableSource bool `json:"no_applicable_source,omitempty"`
	}{valid && !noSource, scan.matches, len(scan.signals), sourceSignalLimit, noSource})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	signalSummaryID, _ := domain.NewEvidenceID("source-content-signals-summary")
	signalFact, err := domain.NewEvidence(signalSummaryID, signalsID, artifact.Identity(), artifact.Digest(), "source-content-signals", string(signalSummary), profile)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	evidence := []domain.Evidence{signalFact}
	var findings []domain.Finding
	for index, signal := range scan.signals {
		summary, err := json.Marshal(struct {
			Rule        string         `json:"rule_id"`
			Version     int            `json:"rule_version"`
			Severity    string         `json:"severity"`
			Disposition string         `json:"disposition"`
			Path        string         `json:"path_sha256"`
			Content     string         `json:"source_sha256"`
			Start       sourcePosition `json:"start"`
			End         sourcePosition `json:"end"`
		}{signal.Rule, signal.RuleVersion, signal.Severity, "manual-review", pathHashes[signal.Path], contentHashes[signal.Path], signal.Start, signal.End})
		if err != nil {
			return domain.InspectionReport{}, err
		}
		id, _ := domain.NewEvidenceID(fmt.Sprintf("source-content-signal-%d", index))
		fact, err := domain.NewEvidence(id, signalsID, artifact.Identity(), artifact.Digest(), "source-content-signal", string(summary), profile)
		if err != nil {
			return domain.InspectionReport{}, err
		}
		finding, err := domain.NewFinding("M13_SOURCE_REVIEW", []domain.EvidenceID{id})
		if err != nil {
			return domain.InspectionReport{}, err
		}
		evidence = append(evidence, fact)
		findings = append(findings, finding)
	}
	signalReport, err := domain.NewInspectionReport(signals, findings, evidence)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	report, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{coverageReport, signalReport})
	if cancelled := ctx.Err(); cancelled != nil {
		return domain.InspectionReport{}, errors.Join(runtimeErr, decodeErr, err, cancelled)
	}
	return report, errors.Join(runtimeErr, decodeErr, err)
}
