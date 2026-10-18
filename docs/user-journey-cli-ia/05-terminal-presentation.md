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
  uses LC_ALL, LC_MESSAGES then LANG, with English fallback. Locale affects CLI strings only,
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
