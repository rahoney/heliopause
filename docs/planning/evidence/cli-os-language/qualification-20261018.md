# CLI-UX-002 — OS display-language qualification

Status: COMPLETE. IMPLEMENTED=YES / WIRED=YES / QUALIFIED=YES /
ACCEPTANCE_CLOSED=YES / MISSING=none for this documented language scope.
This report is updated only from actual evidence; cross-compilation is not native
execution. User scope: detailed plans plus second review, then language work;
update commands remain NOT_STARTED and publication remains held.

## Reviewed plan and implementation

The detailed changes and second review were preserved before source in
commit4d6cc0fb174918a1cf64b117d32fafece2510947. The
[baseline/review](../prepublication-gaps/baseline-20261018.md),
[language plan](../../../user-journey-cli-ia/05-terminal-presentation.md#cli-ux-002-exact-change--validation-plan)
and [update map](../../../user-journey-cli-ia/06-update-quarantine.md#implementation-map-and-scoped-command-semantics)
identify existing owners, update selection semantics, foreign-state adoption,
rollback, per-lane tests and commit boundaries. They do not authorize a rewrite of
implemented transactions. The plan review corrected 11 assumptions/gaps; final
source review checked the exact eight changed Go/test files against frozen hashes.

CLI system_language.go owns precedence/cache/source reporting; presentation.go
extends its existing auto branch; language_settings.go preserves saved auto and
reports effective selection after a save. Bootstrap injects the read through the
existing process context before presentation preparation. hosttool/display_language.go
owns only fixed OS preference reads, strict tags,2s timeout/100ms pipe drain and
4KiB each for stdout/stderr. There is no new Core Port/package/module, generic
artifact command capability, privilege/configuration/runtime image or Policy change.
Manual flag, environment and saved selections avoid the OS query; automatic lookup
is cached once per command. JSON schemas/enums/exit semantics remain unchanged.

Windows/WSL uses the fixed standard Windows system PowerShell path, no profiles,
module autoload or artifact input, and reads current Windows UI culture. DrvFS
ownership is not substituted for trusted Unix executable identity. macOS uses
verified /usr/bin/defaults and only global AppleLanguages. Disabled/unavailable
interop, nonstandard Windows mount/system path, missing desktop preferences,
invalid output or timeout fall back to bounded locale hints then English. Settings
show a stable source/fallback reason. Remote execution uses its execution host,
not an assumed local editor/client preference. OS settings are never changed.

## Evidence and discovered failures

Portable [result](./result.json) binds exact source hashes and measured checks.
Original raw owner: docs/github-logs/cli-os-language-20261018/ (local evidence).
Pinned Go1.26.9, CGO0 candidate and offline locked tools/modules; task-private
native source/build/Staticcheck cache; complete tracked plus new source inventory.

- Source copy initially inherited an unusable ancestor /tmp/.git. Two Quick
  attempts failed before compilation; the second was mistakenly started before
  preparation completed. Exact independent Git metadata and sequential completion
  corrected the harness without changing /tmp/.git or disabling VCS checks.
- Existing actual-process smoke assumed English Usage. New native help correctly
  became Korean. Tests still execute default/minimal-environment help and validate
  stable command syntax; explicit language/auto behavior has separate coverage.
- A real bound regression was caught: embedding bytes.Buffer promoted ReaderFrom,
  allowing os/exec io.Copy to bypass the limiting Write. An earlier short helper
  timeout masked overflow. Named buffer storage removes that method promotion;
  actual stdout/stderr overflow tests now reject timeout as false proof of bounds.
  Failed frozen source/candidate988fd235 were never installed; raw is preserved.
- Final exact source: canonical Quick113.87s/security14.23s/docs7.76s/workflow8.24s,
  targeted Race29.44s PASS. Race includes actual timeout/output-bound child tests,
  parsing, selection, settings, help/JSON and native-available read paths. This is
  not closure of the historical unrelated full-Race issue.
- Darwin amd64/arm64 hosttool and bootstrap test compilation PASS (four outputs).
- Actual compiled CLI real PTY16 cases PASS: English C.UTF-8 process with Windows
  UIko-KR yields ko/WINDOWS_UI, one-shot English, environment Japanese, saved
  English, reset/persist auto, seven explicit-language helps, malformed preference
  fallback and repair. Actual automatic calls measured about0.31–0.33s; explicit
  selections about0.02s. These measurements are this host, not a timing guarantee.

## Local activation and closure boundary

Qualified local-development CLI SHA256:
eeb11993682e14a7c285dd3f79e924b587a31c9d5df9394b0d997ed2db942b13
(26,488,645 bytes). Actual protected-file hashes/modes/owners, empty runtime
endpoints and no active/unknown consumers were checked before atomic root0755
replacement. Previous CLI027970d5 was preserved root0600 in durable operator state.
Symlink/helper/observer/network-policy configuration/runtime images were unchanged.
Actual ordinary-user installed CLI returned saved auto/current ko/WINDOWS_UI;
explicit --lang en returned English/FLAG. Tool sandbox interop denial independently
returned English/OS_LANGUAGE_UNAVAILABLE without breaking help/config.

Installed ordinary-UID1000 npm inspect is-number@7.0.0 JSON smoke PASS8.01s:
COMPLETED/ALLOW with three required checks and original Evidence preserved. The
unchanged canonical helper lifecycle returned wait0/CONFIRMED_STOPPED and primary0/
cleanup0. Final protected-file/idle/rollback postcheck PASS.

Source commit3482e77275b83e5aec873bca5b85ce89c6260706 passed all ten ordinary
[CI328 jobs](https://github.com/rahoney/heliopause/actions/runs/39120811016),
including native macOS platform build/default tests, Linux integration and Required.
Actual checkout cf9925284c75fffd3797db8d74765e13a6959694 has the same complete
tree as that head. macOS suite success covers native execution, parsing and
fallback; it does not prove a positive desktop AppleLanguages preference on a
headless runner. Windows UI selection was positively observed on the actual WSL.

Closure edits documentation/evidence only; eight Go/test hashes remain exactly
qualified. Final PR-head CI is retained in the local raw owner without an extra
self-referencing evidence commit. [Draft PR38](https://github.com/rahoney/heliopause/pull/38)
remains for review; main is unmerged. Disposable cleanup amounts are in result.json.
User preference is read-only in tests; all writes use private settings/cache.
Preserve original Evidence and necessary rollback, delete completed private
source/cache/test settings/binaries after last use. No release/tag/name/publish or
main merge occurs in this item. Update command plans remain unimplemented.
