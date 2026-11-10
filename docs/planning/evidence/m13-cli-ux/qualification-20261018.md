# CLI-UX-001 qualification — 2026-10-18

IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES / ACCEPTANCE_CLOSED=YES for
[terminal presentation](../../../user-journey-cli-ia/05-terminal-presentation.md).
MISSING: none in this local usability item. The [baseline](./baseline-20261018.md)
is the pre-source audit; its initial NO states remain historical.

## Result and responsibility

- Existing human result presenters are extended and wired to the six npm/PyPI/
  GitHub inspect/install bindings. Real TTY defaults to human; redirected existing
  machine output and explicit JSON keep original schemas/enums/exit codes.
  Go/Cargo/Terraform retain their existing non-TTY text and gain interactive
  summaries; unsupported explicit JSON is rejected, not fabricated.
- `internal/cli/presentation.go`, `language_settings.go`, `presentation_catalog.go`
  and `translations.go` own locale, output mode, help/doctor and saved preference.
  Seven locales: en/ko complete owned presentation templates; de/fr/es/pt-BR/ja
  core menus/descriptions with English fallback. System locale is the WSL/Linux
  process locale (LC_ALL/LC_MESSAGES/LANG), not Windows display-language discovery.
  Technical identifiers/codes stay English. No runtime AI or new dependency.
- `human_report.go` separates operation completion, Policy and actual promotion;
  shows required checks/capability/limitations, fixed explanations, real IDs and
  next actions. Unknown codes remain visible with a generic explanation.
  `presentation_error.go` preserves original causes/exit behavior and separately
  displays failed finalization; actual context cancellation is not called a broken
  observer. No completion is promoted to a general safety guarantee.
- `internal/core/ports/progress.go` carries optional observational events only.
  Existing Application workflows emit actual phase transitions; bounded npm,
  PyPI and GitHub acquisition readers count actual bytes inside original limits.
  Known totals use a bar; unknown totals/static/dynamic stages use spinner/time.
  There is no invented inspection percentage/ETA. Provider-specific Go/Cargo/
  Terraform download-byte instrumentation is not claimed. Progress grants no
  Policy authority, adds no retries and changes no inspection budget/order.
- Bootstrap composes the correct backend even with leading global flags. Help and
  language settings work without runtime composition. Only display diagnostics
  are suppressed in concise human mode; inspection, observer ledger and Evidence
  remain enabled. Terminal controls/untrusted text are escaped and bounded.
  The architecture checker permits only the reviewed existing Port imports and
  already-pinned x/sys/unix TTY query; no blanket allowlist relaxation.

## Validation

[Portable result](./result.json) binds exact changed source hashes, local build,
check receipts and actual operations. Original raw owner is
`docs/github-logs/m13-cli-ux-20261018/` (local, not bundled into this commit).
Pinned Go1.26.9, GOTOOLCHAIN=local, offline shared modules/tools, private native
source and build/Staticcheck caches were used. Read-only profiles preserve source.

- Canonical Quick (format check, build, vet, pinned Staticcheck, tests), security,
  docs and workflow PASS. A final source-inventory audit found the new bootstrap
  regression file missing from the first private-source projection. The projection
  was corrected and the complete candidate's canonical Quick was rerun; the
  earlier Quick is not claimed to have exercised that test. Production source
  hashes still match the already built/installed binary; only tests/docs differ.
- Targeted Race PASS: locale/settings, seven-language help, human/JSON decision
  axes, unknown codes, bounded progress, cancellation/finalization and affected
  Application installation cases. This is not resolution of historical full-Race.
- Darwin amd64 and arm64 CLI test compilation PASS. These are cross-compiles,
  not native macOS execution or fresh remote CI results.
- Actual compiled binary in a real PTY: 14 cases PASS (seven-language help, saved
  Korean, one-shot English without overwriting preference, reset auto, truthful
  unavailable doctor, 28-column/no-color output).
- Actual installed CLI as ordinary UID1000: Korean npm inspect is-number@7.0.0
  completed/ALLOW with three required checks; real download bytes/100% bar and
  timed inspection stages appeared. English npm install completed/ALLOW and
  promotion COMPLETED; target's package identity/files were checked. Forced JSON
  inspect retained schema v1, all three checks/ALLOW and clean JSON stdout.
  Missing exact version failed with ARTIFACT_RESOLVE_FAILED, no completed checks
  and no Policy decision. Doctor with helper live retained exit1 for missing
  verified release while the four controlled-runtime checks were OK.

## Local activation and cleanup

Qualified local-development CLI SHA256:
`027970d56fa8dcea7266a9044614e5965128980c8526b433de47f92af19fe268`
(26,466,716 bytes; Go1.26.9, CGO_ENABLED=0).
Before atomic root0755 replacement of `/usr/libexec/heliopause/helox`, all four
protected file identities, no active CLI/helper/runtime consumers, no unknown
queries and no existing runtime endpoints were checked. The exact old CLI was
preserved in a root0600 rollback copy under the existing durable operator-state
area. CLI symlink, helper/observer/configuration and runtime images were unchanged.

Actual tests used the unchanged canonical `scripts/ci-integration-cleanup.sh`:
foreground helper launch, owning-parent wait0, CONFIRMED_STOPPED, primary0 and
final cleanup0. Fresh privileged postcheck confirmed idle resources, all protected
identities and the rollback hash. The user language preference was not changed;
all settings/cache/install tests used task-private paths. Nine original Evidence
files and installed file hashes were preserved before discarding payloads.
[Cleanup receipt](./cleanup.json) records completed private source/cache/binary/
settings/payload deletion; shared SDK/modules/tools, active runtime, necessary
rollback and original logs/Evidence remain. Logical bytes are not VHDX compaction.

This source follows protected v0.1.0 and is not present in its previously verified
assets. The tag was not moved. No registry package name was chosen, tap created,
public package/Release published, main merged or fresh remote CI claimed here.
M13-004 waits for owner-approved names, then a new reviewed source/version/build
candidate before publication; M13-005 clean Host/public installation is still open.
