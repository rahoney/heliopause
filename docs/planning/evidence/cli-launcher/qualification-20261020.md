# CLI-UX-003 — local qualification / 2026-10-20

Status: IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO / ACCEPTANCE_CLOSED=NO.
MISSING: exact committed-source ordinary CI including native macOS and completion
documentation. The historical CI346 timeout cause remains unknown; its bounded
diagnostic loss is corrected and locally qualified.
This is local acceptance evidence, not a main merge or permanent CLI activation.

Native CI346 failure: job118163210310 production build passes but default test
hits the unchanged outer6m deadline at16:22:31UTC. No partial package output is
visible in its error. Promotion.test orphan cleanup does not identify the cause.
The [pre-edit diagnostic amendment](./baseline-20261020.md#native-ci346-timeout-visibility-amendment--before-diagnostic-source-edits)
adds bounded timeout-output preservation before further qualification.
The source failure is retained; native qualification remains NO. Earlier status
query values lagged the completed log and are superseded by this actual failure.
Local qualification data1,758,485,414 logicalB and four one-shot operators are
already deleted; raw/proof/shared SDK tools remain. A new diagnostic source/cache
was subsequently qualified and deleted (1,269,719,952 logicalB plus three operators).
The current Darwin test-only source/cache is separately tracked and removed after
its final use; shared pinned tools and proof are retained.

## Existing owners and implementation boundary

The [pre-source baseline/edit map/second review](./baseline-20261020.md) was
committed as53c18256da7d1aa5fa7b8073a2447374f946f7d3 before product edits.
[Presentation](../../../user-journey-cli-ia/05-terminal-presentation.md#deferred-branded-launcher--cli-ux-003)
owns behavior. Existing help, settings, catalogs, Doctor and display-width/color
owners were reused. No implemented owner was replaced.

- [Root](../../../../internal/cli/root.go) RunE dispatches only eligible bare
  terminal use to [launcher](../../../../internal/cli/launcher.go); direct commands,
  help, JSON mode, CI, redirected streams and unusable terminals retain existing
  behavior. Usage pages call static help, never command RunE or a shell.
- [Keys](../../../../internal/cli/launcher_keys.go) bound split/unknown/overlong
  escape input. [Unix input](../../../../internal/cli/launcher_terminal_unix.go)
  owns exact termios save/restore and context-aware polling; Linux/Darwin ioctl
  wrappers are build-selected. Existing process SIGINT cancellation remains
  effective during an explicit synchronous diagnosis.
- Language menus reuse read/writeLanguageSettings and configure. Auto persists
  as auto; flags/environment overrides remain visible and take precedence.
  Existing catalogs cover all seven core menus with complete English/Korean
  explanations and deterministic English fallback for secondary technical text.
- [Bootstrap](../../../../internal/bootstrap/bootstrap.go) binds the existing
  Doctor lazily for bare helox. Only explicit selection resolves the release
  root and performs readiness checks. The existing doctor presenter is reused
  without its direct-command exit1 footer inside the navigable menu.

Only five existing Go files are narrowly extended; ten new CLI/bootstrap files
implement and test this current item. Core/Application/Artifact/Policy/runtime/
observer/helper/installer/workflow and dependency lock inputs are unchanged.

## Local checks and exact input boundary

Checked source inventory digest:
2d0d4f86546f9a617de4bdd6f913a3fa76add102a1c4040717d01f277f283d4c.
It contains1684 source/fixture/document paths, including every new Go test.
The final workspace overlay proof compares all22 changed/new paths against that
inventory and confirms the other tracked paths unchanged according to Git.
Subsequent qualification/status changes are documentation only, separately checked.
Do not represent the earlier18-case input as the final19-case candidate.

| Check | Wall seconds | Actual scope |
| --- | --- | --- |
| canonical format | 0.33 | final candidate, no remaining format changes |
| focused CLI/bootstrap | 2.62 | actual native PTY19 cases plus key/settings/bounds/error/override/explicit-doctor/no-config-root regressions |
| canonical security | 7.38 | pinned gosec, current-tree Gitleaks and detection contract |
| canonical Quick | 17.71 | format, module/runtime drift, production/test build, Architecture, vet, pinned Staticcheck, default suite; normal cache-aware runner, no cold-cache claim |
| canonical docs | 0.20 | planning-era documents in the checked source inventory |
| related full Race | 45.22 | cli21.360s, bootstrap1.437s, cmd/helox3.819s; count1 |
| Darwin amd64 test compile | 18.78 | CLI/bootstrap/cmd test binaries, not native execution |
| Darwin arm64 test compile | 21.38 | same three packages, not native execution |
| actual candidate build | 1.07 | pinned Go1.26.9, trimpath |

The earlier236.87s Quick passed before the added live-resize test; the final
candidate above has its own Quick and focused/Race results. The first focused
failure was a test prerequisite omission: pip-upgrade example validation needed
the declared active virtual environment. Only the test supplies t.TempDir as
VIRTUAL_ENV; production validation was not weakened.

## Actual compiled CLI on WSL/Linux

Candidate identity: role=compiled test CLI,
sha256=056c4af39e43769eac2226240c4a4cef9939a7d80c3636b362edc2ec90fd500b.
Sixteen actual binary/PTY cases passed: seven language menus, help/settings/real
Doctor navigation, live80×24→20×8 resize, Ctrl-C, SIGTERM, explicit help/JSON/CI/
TERM=dumb exclusions and OS-auto selection. All cases restored exact original
termios; interactive cases restored cursor/screen and honored NO_COLOR.

The real settings menu persisted Korean. A private saved auto preference remained
auto and displayed Korean with LC_ALL=C.UTF-8; actual config output reported
WINDOWS_UI. Existing installed user settings were not modified.

Real explicit diagnosis truthfully displayed unavailable release installation
and network-policy service, with the other three checks ready. Remaining in the
menu/exiting it did not turn this into a healthy installation or claim artifact
inspection. No service repair/start, artifact execution or installation occurred.
The four installed CLI/observer/network-helper/policy records retained their
exact hashes, root ownership, modes and single-link identities.

## Proof custody and remaining qualification

Raw remains local/untracked under docs/github-logs/cli-launcher-20261020:
final-{focused,security,quick,docs,race,darwin-amd64,darwin-arm64,candidate}
logs/receipts/source inventories, final-workspace-overlay-proof.json,
actual-final.json and actual-*.txt. These are not remote attachments.
The committed report preserves normalized scope; native macOS/exact-source
remote conclusions and completed-data deletion must still be recorded.

No full CUDA campaign is required for this display-only change. Existing bounded
SEC-UPD-001 qualification remains closed. PR38 stays draft/unmerged; package names,
main merge, permanent local activation and publication remain separate.

## Bounded timeout diagnostic — local verification

Pre-edit plan commitc7bca711c29268257f84441dfb1e6287c3a7481f precedes the
runner/test edits. The original runner boundary test was RED4.66s: both fixed
stdout/stderr markers had been captured and the DeadlineExceeded cause retained,
but the error text discarded that output. The narrow timeout-detail extension
is GREEN2.93s and preserves class/step/cause,1MiB collector, command/argv and
all original profile/timeout/Required inputs.

Exact diagnostic inputfd543346aae6b9fd90b3f5a66fdb98be2d107b2ea02c17d091d9dd756d1fd890:
canonical format0.80s, Quick163.69s/security7.95s/docs0.20s, whole checker
Race30.69s, Darwin amd64/arm64 checker test compile12.39s/11.97s PASS.
The diagnostic patch affects only scripts/check/runner.go and its new Linux/
Darwin boundary test. Launcher/product/dependency/runtime/workflow bytes remain
identical to b19 and retain only their earlier qualified scope.

Raw timeout-output-{red,green} and timeout-diagnostic-* logs/receipts/source
inventories are preserved locally. Native CI346's original underlying timeout
cause remains UNKNOWN; diagnostic preservation is not a performance fix.
The new source still requires native macOS/exact-source remote qualification.

## CI347 native assertion — qualification remains open

Native macOS118189223070/ef43754 fails in15 interactive PTY cases at the
all-field terminal-state comparison; ioctl error is nil. Four noninteractive
cases do not fail. The default suite returns normally within its bound, with
promotion144.173s/checker21.190s PASS. This does not prove the cause of the earlier
CI346 timeout. Raw ci347-macos.log is preserved locally.

The [pre-test Darwin amendment](./baseline-20261020.md#darwin-terminal-state-assertion-amendment--before-test-edits)
records Apple's documented kernel state normalization, required negative tests
and the narrow test-only correction. Exact native field deltas are not present
in the old generic assertion, so no product restoration failure/cause is asserted.
Product code and restored-snapshot procedure remain unchanged pending evidence.

## Darwin comparison — test-only local verification

Plan commite8df754 precedes the test edits; canonical plan docs9.32s PASS.
The old equality is extracted unchanged into the test-only comparator. Its pure
matrix is RED5.72s on five documented Darwin representations. Only canonical
PENDIN and input-speed0 aliases are normalized in copied Darwin snapshots before
the original complete equality; Linux retains exact equality. GREEN1.35s covers
20 paired Linux/Darwin cases, including changed mode/flags/control characters/
actual speeds and noncanonical PENDIN rejection. Failures now print field values.
No production restoration, timeout/profile/Required, runtime or dependency changed.

Exact local input:
de7d2a9cb91d5365af9e4fb8aaaf057f50900a987444298a3cd04215912f12ba.
Canonical format0.28s, focused4.58s (native Linux PTY19 and the comparison matrix),
Quick230.44s/security6.68s/docs0.22s, full CLI Race41.19s (package21.654s),
Darwin amd64/arm64 CLI test compilation12.43s/13.43s PASS. All read-only checks
preserve the full input inventory. The actual CLI16 proof remains applicable
only to its identical product bytes; cross-compilation is not native execution.
Raw termios-* logs/receipts/inventories are local, not remote attachments.

CI347's exact checkout94c7439 has tree648c8bc identical to ef43754, with main/base
and ef43754 parents; its failure is preserved. Its unfinished Linux job may be
superseded under ordinary concurrency by this source push and is never reused
as a pass. Fresh final-source native macOS and full Required qualification remain.
