# SumDB failure diagnosis follow-up

CI38249811910 on `a32cdb0` failed after the initial gRPC1.76/41-module get
succeeded: retained download returned `go SumDB authenticated archive lookup
failed`. The earlier main CI37627224299 passed the same test. These runs differ
in host Go and hosted Ubuntu image; the previous generic error cannot establish
which underlying failure occurred. The host security patch itself passed
Scheduled Security38249522013.

The existing verifier discarded both the HTTP error category and the upstream
SumDB lookup error. The pinned x/mod client also formats its returned error with
`%v`, so wrapping that error would neither preserve context identity nor satisfy
the requirement to keep raw peer/module details out of public diagnostics.

The bounded repair records only fixed local failure categories, lookup/tile and
request/headers/body stage, HTTP status, existing request/byte counters and
elapsed time. Cancellation/deadline sentinels remain discoverable with
`errors.Is`. Signed-tree consistency rejection is recorded without peer text.
Last remote failure fields are observations: a recovered partial-tile request
is not asserted to be the final cause. Success criteria, signing key, redirect
policy, request/payload/time limits, runtime lock and Go versions are unchanged.

The same new regression failed all seven cases before the change. Afterward,
deadline, cancellation, timeout, transport, HTTP503, body-read and malformed
record failures preserve safe diagnostics and never produce verified reports.
Raw peer text and module path disclosure are rejected by the test. All existing
signature/tile/checksum/redirect/forgery negatives also pass with the race
detector, using actual Go1.26.9 and a source overlay on the Go1.26.8 baseline.

Local validation on the diagnostic candidate:

| Check | Result |
| --- | --- |
| Canonical Quick | PASS: format, locks, module integrity, build, test compilation, architecture/workflow, vet, Staticcheck, default tests |
| Canonical security, vulnerability, docs | PASS |
| Verifier package tests including safe diagnostic regression, `-race -count=1` | PASS on actual Go1.26.9 and Go1.26.8 |
| Actual public module acquisition and verified project cache integration | Both PASS on Go1.26.9 |

The first local Quick/public integration invocations were denied Unix socket /
network operations by the execution sandbox. The same commands passed with
approved host access; these environment failures do not reproduce the remote
SumDB failure. Raw command/exit/log records are preserved in the task validation
directory as `sumdb-diagnostics-before.log`, `sumdb-diagnostics-after.{log,json}`,
`sumdb-diagnostics-go1268.{log,json}`, `quick-4.log`, `quick-5.log`,
`security-4.log`, `vulnerability-2.log`, `docs-4.log` and
`public-go-integration-{2,3}.log`.

This change fixes an established diagnostic gap. It does not assert that a
network outage caused the earlier run, or qualify the full Linux integration.
The authorized candidate push and new Required CI provide the next evidence.
The separate pre-existing full-Race fixture findings remain open.
