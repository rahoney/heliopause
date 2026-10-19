# Terminal presentation — CLI-UX-001

This leaf owns actual presentation decisions supplementing User-Journey-006.
The user requested multilingual, readable progress/results before publication.
Public registry/package names remain owner decisions. Existing inspection,
Policy, result schemas, exit codes, bounds and least-privilege execution remain.

## Modes and language

- `--output auto|human|json`: auto selects human on a real stdout TTY and preserves
  existing machine output for redirected npm/PyPI/GitHub inspect/install. Explicit
  modes allow reproducible interactive and automation use. Existing Go/Cargo/
  Terraform textual contracts must not silently become a new JSON schema.
- `--lang auto|en|ko|de|fr|es|pt-BR|ja`: explicit language wins. Otherwise
  HELOX_LANG and a saved CLI preference take precedence over system locale; auto
  now reads OS display preference, then bounded process locale hints and English
  fallback (see CLI-UX-002 below). Locale affects CLI strings only,
  never isolated child environments, package identities, identifiers or JSON enums.
  English/Korean cover complete user-facing templates. German, French, Spanish,
  Brazilian Portuguese and Japanese cover core menus/descriptions; missing
  translations fall back deterministically to English. Technical terms, error
  codes, identifiers and machine schemas remain English. No AI is involved.
- `helox config language [auto|en|ko|de|fr|es|pt-BR|ja]` shows or atomically saves
  the user preference in the OS user config directory under heliopause/cli.json.
  This is presentation-only data, not the protected runtime or policy config.
  CLI flags are one-shot overrides and do not silently rewrite saved settings.
- `--color auto|always|never` and `--progress auto|always|never`: auto uses real
  terminal capability, TERM=dumb and NO_COLOR. Non-TTY output never contains ANSI
  cursor animation. JSON stdout contains only the original schema result; no
  progress, translated enums or human footer is appended.
- Help and doctor explain user-facing commands/readiness in the selected language.
  Doctor retains exit1 for any unavailable requirement, including a missing
  verified public release installation; working controlled runtime is separate.

## Real progress

Progress is observation, never a trust/completion/Policy authority. Only actual
workflow stage transitions and actual bounded network-reader byte counts drive
it. The typed Port callback cannot return an approval, change budgets, execute an
artifact or swallow original errors. The CLI owns its bounded timer/render state
and stops it on every success/failure/cancellation before final output.

Known byte totals show actual bytes/total and a clamped bar; download100% means
bytes received, not verification or safety. Unknown totals show bytes and an
indeterminate spinner/time. Expensive static/dynamic verification shows its real
stage and elapsed time, with no fabricated percentage or ETA. Dependency stages
may repeat; completed units never imply the entire graph completed. Actual stage
coverage must be documented; missing provider instrumentation is not claimed done.

Animation uses one width-bounded line and restores the cursor/line on exit. Plain
text/NO_COLOR and narrow terminals remain legible. Raw observer attribution is
optional diagnostic output, separate from a user's final security result; suppressing
its display does not suppress required inspection, ledger or Evidence.

## Final report

Reuse existing human presenters and the exact Domain/Application outcomes used by
JSON. First show operation completed/failed/not performed independently of Policy
ALLOW/MANUAL_REVIEW/BLOCK and actual promotion. Then show exact requested/resolved
artifact, target, per-check completion/capability/limitation, important reasons and
stable error codes, Run/Operation identifiers and available Evidence handles.

A completed check is labelled completed, not automatically safe. A supported check
that did not execute, missing Policy, incomplete cleanup, unknown capability or
unsupported backend is never shown as passed. Failed promotion remains failed even
when prior Policy is ALLOW. Do not invent an available review command, explain away
missing installation or label ALL SAFE from a limited experiment. Unknown reason
codes stay visible and escaped, with a generic explanation rather than a guessed
root cause. No arbitrary provider output, credentials, URLs or terminal controls
are replayed. Strings are bounded and escaped for safe terminal rendering.

## OS display-language correction — CLI-UX-002

Status: COMPLETE for the documented bounded OS reads and fallback; local WSL/PTY,
installed CLI and ordinary CI328 including native macOS PASS.
[Current qualification](../planning/evidence/cli-os-language/qualification-20261018.md)
owns exact source and platform scope. The former locale-only algorithm was not
the user's complete OS-language intent. Command names such as install stay English independently of
translated descriptions and preferred UI language. A terminal does not have to
use English locale, but an English/C locale must not hide an available OS display
preference when the user selects auto.

Target: preserve explicit --lang, HELOX_LANG and saved explicit language precedence.
Auto reads the current user's OS preferred display languages: Windows user through
WSL interoperability, macOS user language preferences, or supported native Linux
language preferences. Select the first supported preference; when the OS lookup
is unavailable use supported process-locale hints and finally English. Saved auto
must remain auto, not persist a detected language. Remote/container/headless
execution uses the available execution-host preference, with honest fallback;
no local VS Code/client UI language can be inferred from an arbitrary server.

Reads must be bounded and display-only, with no OS locale changes, elevated
privileges, arbitrary artifact-provided commands, network or new daemon/runtime
requirements. Record the effective language and selection source in language
settings output so English fallback is understandable. A failing/unsupported
lookup must never prevent inspection or contaminate JSON result stdout.
[Baseline and qualification gaps](../planning/evidence/prepublication-gaps/baseline-20261018.md)
remain historical; actual implementation/remaining qualification follows the
current result above.

### CLI-UX-002 exact change / validation plan

Pre-source baseline: 8bbe0a5; OS lookup IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/
ACCEPTANCE_CLOSED=NO; settings/catalogs/presenters already YES and reused.
Only CLI-UX-002 becomes IN_PROGRESS after the documentation review checkpoint.

- Extend cli/presentation.go configure only at the automatic-selection branch.
  Keep explicit flag > HELOX_LANG > saved explicit preference. Inject a narrow
  display-language callback through bootstrap.Run before PreparePresentation;
  no Host executable invocation/import in CLI, no new Core/Policy Port or package.
  Cache the OS probe once per command instance, including unavailable results;
  parsing/pre-run/help must not spawn the same probe repeatedly. Manual selections
  do not probe. Pass the existing process context through the composition closure.
- Add a read-only display preference provider inside existing internal/hosttool.
  It is separate from artifact tool admission and cannot register executable
  roles or accept caller-controlled command/paths. macOS reads only global
  AppleLanguages using fixed /usr/bin/defaults read -g AppleLanguages. WSL is
  detected from bounded kernel information and uses fixed standard
  /mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe with NoLogo,
  NoProfile, NonInteractive and a constant .NET CurrentUICulture.Name query.
  This reads the active Windows UI language, not keyboard/input preferences.
  Use fixed working directories, bounded/minimal local environment, no shell
  interpolation/PATH lookup/profile/module load/privilege/network. Do not loosen
  existing artifact-executable identity or runtime helper rules for this probe.
- Windows NTFS permission mapping is not Unix root0755: document that standard
  Windows system path relies on Windows system-directory access control; it is
  display telemetry only. Nonstandard automount/system-drive layouts or disabled
  interop are unavailable, with explicit setting and locale/English fallback;
  never search arbitrary PATH or user-provided scripts to make lookup succeed.
  macOS uses its existing trusted system executable verification. Native Linux
  falls back to process display locale hints (including bounded LANGUAGE lists).
- Budget the external read at 2 seconds plus at most 100ms pipe-drain allowance,
  at most4KiB stdout and4KiB stderr; stop/reap on context, overflow or time limit.
  Reject malformed/empty/oversized output and non-language tokens. Never replay
  raw stdout/stderr/errors. macOS preferred arrays and Windows primary UI tag
  are parsed as bounded language tags, then the first supported tag is chosen.
- Extend language_settings.go output with effective language source and a stable
  fallback reason, including immediately after saving auto. Saved JSON stays
  language-only and must persist auto; don't cache detected language to disk or
  change OS/shell settings. Reuse text catalogs; en/ko explanatory labels cover
  new settings output, other languages may use English fallback.
- Unit-test platform selection and strict parsing with injected command results;
  separately test real bounded process timeout/overflow/nonzero behavior. CLI
  regressions cover all seven tags, unsupported/malformed preferences, source
  reporting, overrides without probing, saved-auto persistence, one probe across
  bootstrap/pre-run/help, JSON cleanliness and no-HOME/minimal-process behavior.
  Include bootstrap tests in the actual check source inventory, not only CLI.
- Validate canonical format/Quick/security/docs/workflow and targeted Race,
  Darwin amd64/arm64 test compilation, actual compiled-binary PTY/config/help on
  WSL (English process locale plus Windows UI preference), overrides and fallback.
  Native macOS tests belong to ordinary PR CI; cross-compilation is not native
  preference lookup evidence. Do not rerun CUDA because only presentation changes.
- Review final diff for accidental Policy/runtime/helper changes, unsafe command
  selection, duplicate probes, swallowed errors, stale setting confirmations and
  unverified platform claims. Before updating installed local CLI preserve exact
  old hash/owner/mode and root rollback; confirm no active consumers/unknown
  resources, atomically replace only the qualified CLI, and verify hashes again.
  Helpers/config/runtime images stay unchanged. Preserve necessary rollback and
  raw evidence; delete completed private source/cache/binary/test settings.

References: [Microsoft UI culture](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.utility/get-uiculture),
[Apple global preferences](https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/UserDefaults/AboutPreferenceDomains/AboutPreferenceDomains.html).
Actual pre-implementation fixed Windows read returned ko-KR in0.50s on this WSL;
this is a probe-method check, not application wiring/qualification.

## Deferred branded launcher — CLI-UX-003

User-confirmed order on2026-10-18: implement and qualify SEC-UPD-001 guarded
updates first, then this separate UX item. Bare helox in a real stdin/stdout TTY
should display Heliopause Artifact Airlock and localized menus for currently
available settings, usage/examples, explicitly requested environment diagnostics
and exit. Arrow keys move, Enter selects, Esc goes back, q exits. Reuse language
selection/catalogs and existing command services. Preserve direct subcommands,
--help and machine/CI behavior without prompts. Support narrow terminals and
NO_COLOR; restore terminal modes/cursor on exit, cancellation and errors.
Status: COMPLETE at source4cfe29e/CI352 ten SUCCESS, including native macOS and
actual Linux consumers; earlier terminal/capture successes remain scoped history. The historical heading preserves deferred ordering and
existing links. The [baseline/edit map/second review](../planning/evidence/cli-launcher/baseline-20261020.md)
precedes source edits. Existing help/config/catalogs/Doctor are reused.
[Qualification](../planning/evidence/cli-launcher/qualification-20261020.md)
records local PTY19/actual CLI16, native macOS/exact-source CI350 ten SUCCESS,
completed-test data deletion and unchanged installed identities.
[CI351 storage amendment](../planning/evidence/cli-launcher/baseline-20261020.md#native-ci351-default-budget-and-private-cache-sync--before-source-edits)
keeps existing UX, original-scale tests and mandatory final durable publication.
The changed private storage path passes native/full CI352 at unchanged limits.
Final documentation delivery is checked separately on its own exact head. Main merge,
permanent local activation and publication remain separate.

### Launcher behavior and lifecycle

Interactive entry requires real stdin and stdout terminals, auto/human output,
nonempty usable TERM, no CI indication and initial geometry of at least20 columns
and8 rows. Other cases retain existing static help without reading input. Never
open /dev/tty to bypass redirected stdin. Direct subcommands and --help retain
their existing execution paths. Extremely small initial terminals use static
help; narrow supported screens and resized windows use bounded viewports.

The main menu contains usage/examples, language settings, explicitly requested
environment diagnosis and exit. Display the full product name; arrows navigate,
Enter selects, Esc returns, q/Ctrl-D exit and Ctrl-C preserves interruption.
Help/detail screens scroll. Examples are inert text derived/reviewed against
currently registered commands; there is no embedded shell or automatic install,
update, inspection, download or repair from menu navigation.

Language changes use the existing atomic preference writer and language selector.
Show saved/effective language and selection source. Preserve auto and the explicit
flag/environment override precedence; explain when a saved preference is
overridden. All seven languages cover core menu labels; English/Korean cover
complete explanations. Other missing technical descriptions fall back to English.

Only selecting diagnosis calls the existing Doctor service. Release-root lookup
and Host readiness checks remain lazy; missing configuration is a fixed
UNAVAILABLE check. A diagnosis inherits cancellation with a30-second call bound.
Reuse the exact checks/descriptions without the direct doctor's exit1 footer in
the navigable result screen. Unhealthy/cancelled/limited diagnosis never becomes
healthy because the menu stays open or exits0; direct doctor still exits1.

The CLI owns a bounded64KiB help buffer, escape decoder and context-aware terminal
poll loop. Linux/Darwin adapters use existing pinned x/sys; no new dependency,
framework, unsafe/cgo or product subprocess is required. Save exact input termios
before changes, own alternate-screen/cursor state before the first write, and
restore termios/screen/cursor on exit, cancellation, EOF and setup/read/write
failure. Preserve primary errors when joining cleanup failures. Render within
actual stdout rows/cells without last-column wrapping, and reuse existing Unicode
width and color helpers. NO_COLOR/--color=never disable SGR colors while keeping
navigation controls; explicit --color=always keeps its existing override semantics.

Native /usr/bin/script is an explicit test-only PTY driver with separately
selected Linux/Darwin arguments. Native default CI tests must fail on a missing
driver rather than skip this acceptance. Actual PTY, cancellation/error cleanup,
configuration, seven-language/narrow rendering and exact-source native macOS
qualification remain required before launcher acceptance closes.

Terminal-state qualification compares effective saved settings. Darwin's
documented automatic PENDIN when returning to canonical input and input-speed0
alias are normalized in copied test states; every remaining field is compared.
Linux retains full state equality. Echo/canonical/signal modes, all other flags,
control characters and actual speed changes remain failures. Product restoration
still applies the original full snapshot; no extra input flush discards user data.

Native PTY input readiness waits for a complete current frame at the known
fixture geometry; diagnosis readiness uses its own notice and footer. A prior
frame cannot release input. The private child-completion witness is diagnostic
only and never replaces captured output or original terminal assertions.
