package sandbox

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestFaultRawDiagnosticIsBoundedAndPreservesFailure(t *testing.T) {
	raw := FaultRawDiagnostic{Check: "DESCRIPTOR", Sysno: 46, FD: ^uint64(0), KernelImage: "CARGO", Role: "CONTROL", Provenance: "DIRECT_EXEC_ROOT", ExecutableLocator: ^uint64(0), SocketpairObserved: true, SocketpairFDMatch: true, SocketpairDomain: -2147483648, SocketpairType: ^uint32(0), SocketpairSource: "GUEST_RETURN_BUFFER"}
	record := helperRecord{ContainerID: strings.Repeat("a", 64), Kind: "stream-fault", Reason: "FD_STATE_UNKNOWN", FaultSite: "RAW", FaultRaw: &raw}
	body, err := json.Marshal(record)
	if err != nil || len(body) > maximumHelperRecordBytes {
		t.Fatal("raw diagnostic exceeds the existing record bound")
	}
	if _, err := decodeHelperRecord(body); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*helperRecord){
		func(r *helperRecord) { r.Kind = "stream-end" },
		func(r *helperRecord) { r.Reason = "STREAM_FAULT" },
		func(r *helperRecord) { r.FaultSite = "OPEN" },
		func(r *helperRecord) { r.FaultRaw.Check = "/private/descriptor" },
		func(r *helperRecord) { r.FaultRaw.Sysno = 1 },
		func(r *helperRecord) { r.FaultRaw.KernelImage = "/private/secret" },
		func(r *helperRecord) { r.FaultRaw.Role = "TRUSTED" },
		func(r *helperRecord) { r.FaultRaw.Provenance = "artifact claimed" },
		func(r *helperRecord) { r.FaultRaw.SocketpairSource = "TRUSTED" },
		func(r *helperRecord) { r.FaultRaw.SocketpairObserved = false },
	} {
		copyRecord, copyRaw := record, raw
		copyRecord.FaultRaw = &copyRaw
		mutate(&copyRecord)
		body, err := json.Marshal(copyRecord)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeHelperRecord(body); err == nil {
			t.Fatal("misplaced or unbounded raw diagnostic admitted")
		}
	}
	fault := observerFault{reason: record.Reason, site: record.FaultSite, raw: raw}
	observations, limitation, diagnostic := collectTraceDiagnostic(context.Background(), &traceReader{err: fault})
	if len(observations) != 0 || limitation != "M3_DYNAMIC_OBSERVER_FAILED" || diagnostic.SessionComplete || diagnostic.Reason != record.Reason || diagnostic.FaultRaw != raw || !strings.Contains(diagnostic.String(), "raw_check=DESCRIPTOR raw_sysno=46 raw_fd=18446744073709551615 raw_kernel_image=CARGO raw_role=CONTROL raw_provenance=DIRECT_EXEC_ROOT") {
		t.Fatal("raw diagnostic repaired or hid the original failure")
	}
}

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
		func(r *helperRecord) { r.FaultOpen.KernelImage = "/private/secret" },
		func(r *helperRecord) { r.FaultOpen.RustcArgc = 34 },
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

func TestFaultOpenKernelImageRemainsDiagnostic(t *testing.T) {
	for _, kernelImage := range []string{"UNKNOWN", "BOUNDARY", "SETPRIV", "SHELL", "ENV", "NPM_CLI", "NODE", "CARGO", "TAR", "RUSTC"} {
		open := FaultOpenDiagnostic{Image: "OTHER", KernelImage: kernelImage, ExecutableLocator: 1, MountpointLocator: 2, ExecutablePinned: true, GoDriverCreator: true, CargoDriverCreator: true, RustcVersionQuery: true, RustcMetadataQuery: true, RustcArgc: 33, RustcArgvLocator: ^uint64(0), Role: "ARTIFACT", Provenance: "DIRECT_EXEC_ROOT", Subject: "OCI_IMAGE", Mount: "oci-root", Flags: 557056}
		record := helperRecord{ContainerID: strings.Repeat("a", 64), Kind: "stream-fault", Reason: "STREAM_FAULT", FaultSite: "OPEN_RESULT_CLASSIFICATION_IMAGE", FaultOpen: &open}
		payload, err := json.Marshal(record)
		if err != nil || len(payload) > 1024 {
			t.Fatal("kernel image diagnostic exceeds the existing record bound")
		}
		if _, err := decodeHelperRecord(payload); err != nil {
			t.Fatal(err)
		}
		fault := observerFault{reason: record.Reason, site: record.FaultSite, open: open}
		observations, limitation, diagnostic := collectTraceDiagnostic(context.Background(), &traceReader{err: fault})
		if len(observations) != 0 || limitation != "M3_DYNAMIC_OBSERVER_FAILED" || diagnostic.SessionComplete || diagnostic.FaultOpen != open || !strings.Contains(diagnostic.String(), "open_kernel_image="+kernelImage) || !strings.Contains(diagnostic.String(), "open_executable_fnv1a64=0000000000000001") || !strings.Contains(diagnostic.String(), "open_mountpoint_fnv1a64=0000000000000002") {
			t.Fatal("kernel diagnostic repaired or hid the observer failure")
		}
		if !strings.Contains(diagnostic.String(), "open_executable_pinned=true open_go_driver_creator=true") {
			t.Fatal("bounded kernel provenance diagnostic lost")
		}
		if !strings.Contains(diagnostic.String(), "open_cargo_driver_creator=true open_rustc_version_query=true") {
			t.Fatal("Cargo provenance diagnostic lost")
		}
		if !strings.Contains(diagnostic.String(), "open_rustc_metadata_query=true open_rustc_argc=33 open_rustc_argv_fnv1a64=ffffffffffffffff") {
			t.Fatal("bounded Rustc argument diagnostic lost")
		}
	}
}

func TestFaultOpenFixedKernelMetadataRemainsDiagnostic(t *testing.T) {
	for _, subject := range []string{"PROC_SELF_MAPS", "PROC_SELF_STATM", "VM_OVERCOMMIT_MEMORY"} {
		t.Run(subject, func(t *testing.T) {
			open := FaultOpenDiagnostic{Image: "OTHER", KernelImage: "CARGO", Role: "CONTROL", Provenance: "DIRECT_EXEC_ROOT", Subject: "PROC_SELF_MAPS", Mount: "system", Flags: 557056}
			open.Subject = subject
			record := helperRecord{ContainerID: strings.Repeat("a", 64), Kind: "stream-fault", Reason: "STREAM_FAULT", FaultSite: "OPEN_RESULT_CLASSIFICATION_PROC", FaultOpen: &open}
			payload, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeHelperRecord(payload); err != nil {
				t.Fatal(err)
			}
			observations, limitation, diagnostic := collectTraceDiagnostic(context.Background(), &traceReader{err: observerFault{reason: record.Reason, site: record.FaultSite, open: open}})
			if len(observations) != 0 || limitation != "M3_DYNAMIC_OBSERVER_FAILED" || diagnostic.SessionComplete || diagnostic.FaultOpen != open {
				t.Fatal("self maps diagnostic changed failure")
			}
		})
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
