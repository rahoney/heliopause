package sandbox

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestFaultOpenDiagnosticIsBoundedAndDoesNotRepairFailure(t *testing.T) {
	open := FaultOpenDiagnostic{Image: "GO", Role: "CONTROL", Provenance: "DIRECT_EXEC_ROOT", Subject: "PROC_SELF_AUXV", Mount: "system", Flags: 0}
	record := helperRecord{ContainerID: strings.Repeat("a", 64), Kind: "stream-fault", Reason: "STREAM_FAULT", FaultSite: "OPEN_RESULT_CLASSIFICATION_PROC", FaultOpen: &open}
	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeHelperRecord(payload); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*helperRecord){
		func(r *helperRecord) { r.Kind = "stream-end" },
		func(r *helperRecord) { r.FaultSite = "OPEN" },
		func(r *helperRecord) { r.FaultOpen.Image = "artifact-controlled-image" },
		func(r *helperRecord) { r.FaultOpen.Subject = "/private/secret" },
		func(r *helperRecord) { r.FaultOpen.Role = "TRUSTED" },
		func(r *helperRecord) { r.FaultOpen.Provenance = "artifact claimed" },
		func(r *helperRecord) { r.FaultOpen.Mount = "/private/mount" },
	} {
		copyRecord := record
		copyOpen := open
		copyRecord.FaultOpen = &copyOpen
		mutate(&copyRecord)
		payload, err := json.Marshal(copyRecord)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeHelperRecord(payload); err == nil {
			t.Fatal("unbounded or misplaced diagnostic admitted")
		}
	}
	fault := observerFault{reason: "STREAM_FAULT", site: record.FaultSite, open: open}
	observations, limitation, diagnostic := collectTraceDiagnostic(context.Background(), &traceReader{err: fault})
	if len(observations) != 0 || limitation != "M3_DYNAMIC_OBSERVER_FAILED" || diagnostic.SessionComplete || diagnostic.FaultOpen != open {
		t.Fatalf("diagnostic changed failure: %#v %q %#v", observations, limitation, diagnostic)
	}
	if !strings.Contains(diagnostic.String(), "open_image=GO open_role=CONTROL open_provenance=DIRECT_EXEC_ROOT open_subject=PROC_SELF_AUXV open_mount=system open_flags=0") {
		t.Fatal("first causal diagnostic lost")
	}
}

func TestFaultBudgetDiagnosticPreservesLimitFailure(t *testing.T) {
	budget := FaultBudgetDiagnostic{Charged: 10000, Limit: 10000, Close: 3000, Fcntl: 6900, Other: 100, Workspace: 50}
	record := helperRecord{ContainerID: strings.Repeat("a", 64), Kind: "stream-fault", Reason: "EVENT_LIMIT", FaultSite: "EVENT_LIMIT", FaultBudget: &budget}
	payload, err := json.Marshal(record)
	if err != nil || len(payload) > 1024 {
		t.Fatal("diagnostic exceeded the existing record boundary")
	}
	if _, err := decodeHelperRecord(payload); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*helperRecord){
		func(r *helperRecord) { r.Kind = "stream-end" },
		func(r *helperRecord) { r.Reason = "STREAM_FAULT" },
		func(r *helperRecord) { r.FaultSite = "OPEN" },
		func(r *helperRecord) { r.FaultBudget.Limit = 0 },
		func(r *helperRecord) { r.FaultBudget.Limit = 500001 },
		func(r *helperRecord) { r.FaultBudget.Charged = 10001 },
		func(r *helperRecord) { r.FaultBudget.Close = 10001 },
		func(r *helperRecord) { r.FaultBudget.Fcntl = 7000 },
		func(r *helperRecord) { r.FaultBudget.Workspace = 10001 },
	} {
		copyRecord := record
		copyBudget := budget
		copyRecord.FaultBudget = &copyBudget
		mutate(&copyRecord)
		payload, err := json.Marshal(copyRecord)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeHelperRecord(payload); err == nil {
			t.Fatal("misplaced or unbounded counter accepted")
		}
	}
	fault := observerFault{reason: "EVENT_LIMIT", site: "EVENT_LIMIT", budget: budget}
	observations, limitation, diagnostic := collectTraceDiagnostic(context.Background(), &traceReader{err: fault})
	if len(observations) != 0 || limitation != "M3_DYNAMIC_OBSERVER_FAILED" || diagnostic.SessionComplete || diagnostic.Reason != "EVENT_LIMIT" || diagnostic.FaultBudget != budget {
		t.Fatal("diagnostic repaired or obscured the original limit failure")
	}
	if !strings.Contains(diagnostic.String(), "budget_charged=10000 budget_limit=10000 budget_close=3000 budget_fcntl=6900") {
		t.Fatal("budget cause lost")
	}
}
