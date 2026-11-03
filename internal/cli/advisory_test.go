package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
)

func TestRecordedAdvisoryDetailsKeepSubjectFactsAndOriginalDecision(t *testing.T) {
	for _, mode := range []string{"record", "unknown-severity", "allowed", "missing", "corrupt", "foreign-subject", "foreign-profile", "legacy-impact", "cancelled", "unconfigured"} {
		for _, machine := range []bool{false, true} {
			t.Run(mode+map[bool]string{false: "/human", true: "/json"}[machine], func(t *testing.T) {
				ctx := context.Background()
				source, _ := domain.NewSourceID("pypi")
				identity, _ := domain.NewResolvedArtifactIdentity(source, "fixture", "1.0.0", "wheel")
				digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
				artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:display:fixture", 1)
				if err != nil {
					t.Fatal(err)
				}
				run, _ := domain.NewRunID()
				operation, _ := domain.NewOperationID()
				id, _ := domain.NewCheckID("dependency-advisory")
				profile, _ := domain.CheckProfileDigest("advisory-display-test", "fixed original assessment")
				check, err := domain.NewCheckExecution(id, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
				if err != nil {
					t.Fatal(err)
				}
				requirement, _ := domain.NewCheckRequirement(id, domain.CheckInspection, true, profile)
				plan, err := domain.NewCheckPlan(run, artifact, []domain.CheckRequirement{requirement})
				if err != nil {
					t.Fatal(err)
				}
				record := `{"id":"OSV-2026-controlled","url":"https://osv.dev/vulnerability/OSV-2026-controlled","modified":"2026-10-20T00:00:00Z","impact":"AFFECTED","base_score":9.8,"score_known":true,"fixed_example":"1.2.0"}`
				code, decision := "M13_ADVISORY_HIGH", domain.DecisionBlock
				if mode == "unknown-severity" {
					record = strings.Replace(record, `"base_score":9.8,"score_known":true`, `"base_score":0,"score_known":false`, 1)
					code, decision = "M13_ADVISORY_REVIEW", domain.DecisionManualReview
				}
				if mode == "allowed" {
					record = strings.Replace(record, `"AFFECTED"`, `"UNAFFECTED"`, 1)
					code, decision = "M3_REQUIRED_CHECKS_COMPLETED", domain.DecisionAllow
				}
				if mode == "legacy-impact" {
					record = strings.Replace(record, `"impact":"AFFECTED"`, `"impact":2`, 1)
				}
				factID, _ := domain.NewEvidenceID("dependency-advisory-1")
				fact, err := domain.NewEvidence(factID, id, identity, digest, "advisory-record", record, profile)
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(t.TempDir(), "evidence")
				store, err := local.NewStore(path)
				if err != nil {
					t.Fatal(err)
				}
				refs, err := store.Record(ctx, run, []domain.Evidence{fact}, check)
				if err != nil {
					t.Fatal(err)
				}
				policy, _ := domain.NewPolicyDecision(decision, "advisory-display-policy", 1, []string{code})
				reference, _ := domain.NewArtifactReference(source, "fixture@1.0.0")
				result, err := domain.NewInspectOperationResult(domain.OperationResultData{OperationID: operation, Status: domain.OperationCompleted, Reference: reference, ResolvedIdentity: identity, Digest: digest, RunID: run, RunOutcome: domain.RunCompleted, Checks: []domain.CheckExecution{check}, CheckPlan: plan, Evidence: refs, PolicyDecision: policy})
				if err != nil {
					t.Fatal(err)
				}
				var out bytes.Buffer
				root, err := New(&out, io.Discard)
				if err != nil {
					t.Fatal(err)
				}
				root.SetContext(ctx)
				p := presentationFor(root.OutOrStdout())
				p.output = "human"
				if mode != "unconfigured" {
					if err := SetAdvisoryEvidenceReader(root, store); err != nil {
						t.Fatal(err)
					}
				}
				calls := 0
				original := p.readAdvisoryEvidence
				if original != nil {
					p.readAdvisoryEvidence = func(gotRun domain.RunID, ref domain.EvidenceReference, gotIdentity domain.ResolvedArtifactIdentity, gotDigest domain.ContentDigest) (domain.Evidence, domain.ContentDigest, error) {
						calls++
						if gotRun != run || gotIdentity != identity || gotDigest != digest {
							t.Fatal("display reader lost current subject")
						}
						value, hash, err := original(gotRun, ref, gotIdentity, gotDigest)
						if mode == "foreign-subject" {
							foreign, _ := domain.NewSHA256Digest(strings.Repeat("b", 64))
							value, _ = domain.NewEvidence(factID, id, identity, foreign, "advisory-record", record, profile)
						}
						if mode == "foreign-profile" {
							foreign, _ := domain.CheckProfileDigest("other-assessment", "")
							value, _ = domain.NewEvidence(factID, id, identity, digest, "advisory-record", record, foreign)
						}
						return value, hash, err
					}
				}
				if mode == "missing" {
					if err := os.Remove(filepath.Join(path, run.String(), factID.String()+".json")); err != nil {
						t.Fatal(err)
					}
				}
				if mode == "corrupt" {
					if err := os.WriteFile(filepath.Join(path, run.String(), factID.String()+".json"), []byte("changed record"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if mode == "cancelled" {
					cancelled, cancel := context.WithCancel(ctx)
					cancel()
					root.SetContext(cancelled)
				}
				if machine {
					err = WriteJSONResult(root.OutOrStdout(), result)
				} else {
					err = WriteHumanResult(root.OutOrStdout(), result)
				}
				if err != nil {
					t.Fatal("optional detail lookup replaced original result", err)
				}
				wanted := 20
				if mode == "unknown-severity" {
					wanted = 10
				}
				if mode == "allowed" {
					wanted = 0
				}
				if ExitCode(result) != wanted {
					t.Fatal("display changed original policy/exit")
				}
				if mode == "allowed" {
					if calls != 0 || strings.Contains(out.String(), "OSV-2026-controlled") {
						t.Fatal("normal success acquired extra detail I/O")
					}
					return
				}
				good := mode == "record" || mode == "unknown-severity"
				if machine {
					var report machineResult
					if json.Unmarshal(out.Bytes(), &report) != nil || report.Policy == nil || report.Policy.Decision != string(decision) || report.AdvisoryDetails == nil {
						t.Fatal("missing original result/detail status", out.String())
					}
					detail := report.AdvisoryDetails
					if detail.RunID != run.String() || detail.SHA256 != digest.String() || detail.Subject.Name != "fixture" {
						t.Fatal("detail subject changed", detail)
					}
					if good {
						if detail.Status != "AVAILABLE" || len(detail.Records) != 1 || detail.Records[0].ID != "OSV-2026-controlled" || detail.Records[0].FixedExample != "1.2.0" || detail.Records[0].ScoreKnown != (mode == "record") {
							t.Fatal("normalized record fields lost", detail)
						}
					} else if detail.Status == "AVAILABLE" || detail.Issue == "" {
						t.Fatal("missing/changed detail became complete zero", detail)
					}
				} else if good {
					for _, text := range []string{"OSV-2026-controlled", "AFFECTED", "1.2.0", "https://osv.dev/vulnerability/OSV-2026-controlled"} {
						if !strings.Contains(out.String(), text) {
							t.Fatal("human detail missing", text, out.String())
						}
					}
					if mode == "unknown-severity" && (!strings.Contains(out.String(), "unknown severity") || strings.Contains(out.String(), "CVSS base 0.0")) {
						t.Fatal("unknown severity was displayed as safe zero")
					}
				} else if !strings.Contains(out.String(), "Details incomplete") {
					t.Fatal("human output hid missing detail", out.String())
				}
			})
		}
	}
}
