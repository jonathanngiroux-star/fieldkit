# Fieldkit v0.1.0 — draft release notes

Fieldkit is a self-hosted form and survey backend with a versioned
plugin contract and a Typeform migration CLI. One Go binary, one
SQLite file, donation-only.

## What shipped

- **Survey logic engine** — question types (text, single/multi choice,
  rating, NPS), required, hidden fields, show/jump branching with
  parse-time validation (cycles, missing targets, orphan hidden
  questions, inverted ranges rejected). Runs in CI:
  `fieldkit logic --survey s.json --answers a.json --format=json`.
- **Submit path** — public HTTP endpoint (no account), per-question
  validation errors, 413/415/404/400 discipline, hidden answers
  dropped at the door. CSV export with formula-injection defanging.
- **Typeform migrate dry-run** — `fieldkit migrate --from=typeform
  --dry-run --format=json --input export.json`: maps questions, jump
  logic, and endings; lists every unmapped field honestly (statement,
  matrix, hidden); machine-stable report gated in CI. Qualtrics is
  refused by name until Typeform fidelity is green.
- **Versioned plugin contract** — api_version 1 manifests, kinds
  (question_type, webhook, scorer), entrypoint contract checks, an
  in-repo sample plugin that must pass or the build fails:
  `fieldkit plugin validate testdata/plugins/sample`.
- **Desktop GUI** (Wails v2, `wails build`): surveys, authoring,
  respondent runner with live branch logic, response table + CSV
  download, migrate dry-run viewer, plugin validation, donate view
  with copy buttons. First-run wizard with real operations, reopenable
  any time.
- **TUI** (`fieldkit tui`, pure Go): surveys with response counts,
  logic dry-run, migrate summary, donate page with clipboard copy.
  Same wizard, same store, keyboard-first (1-4, w, q, e/b).
- **One data model** — CLI, GUI, TUI, and HTTP all read/write the same
  SQLite file; cross-surface agreement is tested.

## Donation-only

There is no paid tier, no subscription, no invoice, and no feature
gated on money — there never will be. If Fieldkit replaces a paid
survey-platform line item for you, donations are welcome:

- Ethereum / USDC (ERC-20): `0x85ee7E71f762d772599cbF1EC20E651B30657521`
- Bitcoin: `bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg`

No other wallets are associated with this project. See
GOVERNANCE.md for the wallet-change process.

## Install

```sh
go install github.com/jonathanngiroux-star/fieldkit/cmd/fieldkit@v0.1.0
```

Desktop GUI (Linux, webkit2gtk-4.1):

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0
wails build -tags "wails webkit2_41"
./build/bin/fieldkit-desktop
```

Docker: `docker build -t fieldkit .` (scratch runtime, one binary,
VOLUME /data).

## Quality gates

- `go test ./...` — backend, serve, migrate, plugin, store, plus TUI
  sim-screen keyboard suites and the donate-exact/no-billing/fixture-
  hygiene walk gates.
- `-race` on store and serve paths.
- CI fixture gates: migrate report shape (jq), plugin contract, donate
  byte-exact, logic fixture evaluation, docker build, wails build.
- Audit notes: `docs/audit/backend.md`, `gui.md`, `tui.md`, `system.md`.

## Known limits (honest list)

- Qualtrics migration is not implemented (gated on Typeform fidelity
  by design).
- Ending screens are listed in reports but not rendered.
- Matrix questions are unmapped — the documented plugin-contract use
  case; the sample plugin demonstrates the extension point.
- Apply mode for migrations (beyond dry-run) lands after v0.1.
- `docker run` is CI-verified only (not exercised on the release
  machine).

## License

MIT. DCO sign-off required for contributions; no CLA. See GOVERNANCE.md.
