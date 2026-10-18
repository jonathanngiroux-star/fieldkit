# Fieldkit

Fieldkit is a self-hosted form and survey backend with a survey logic
engine, a versioned plugin contract, and a migration CLI that dry-runs
imports from Typeform (Qualtrics on the roadmap). One Go binary, one
SQLite file, no external services.

## Why Fieldkit exists

Teams moving off Typeform or Qualtrics have no good way to see what a
migration would actually change. Fieldkit's core loop is:

1. `fieldkit migrate --from=typeform --dry-run` reads an export and
   produces a machine-readable report: what maps, what does not, what
   needs a plugin.
2. The survey logic engine runs the migrated survey and its branching,
   and can be exercised in CI so logic regressions fail the build, not
   the launch.
3. The versioned plugin API covers what the core does not: custom
   question types, custom scoring, webhooks to systems the core has
   never heard of — with a contract test so a plugin either passes or
   the build fails.

Fieldkit is not a hosted form SaaS, has no pricing tiers, and gates no
feature on money. If it replaces a paid line item for you, donations are
welcome (see below) — that is the only ask.

## Install

Requires Go 1.27+.

```sh
go install github.com/jonathanngiroux-star/fieldkit/cmd/fieldkit@latest
```

Or build from source:

```sh
git clone https://github.com/jonathanngiroux-star/fieldkit
cd fieldkit
go build -o fieldkit ./cmd/fieldkit
```

## Quick start

```sh
# 1. create a survey from a JSON definition
fieldkit surveys create --file my-survey.json --data-dir ./data

# 2. serve it — respondents submit against a public endpoint, no account
fieldkit serve --data-dir ./data

# 3. dry-run a Typeform import before you commit to it
fieldkit migrate --from=typeform --dry-run --format=json \
  --input typeform-export.json

# 4. run survey logic against answers without a server (CI mode)
fieldkit logic --survey my-survey.json --answers answers.json --format=json

# 5. export responses as CSV
fieldkit export csv --survey <survey-id> --data-dir ./data > responses.csv
```

Cold start is minutes: one binary, one SQLite file under `--data-dir`,
no Kubernetes.

## Desktop GUI

The GUI is a [Wails](https://github.com/wailsapp/wails) v2 desktop
app (Linux: WebKitGTK; Windows: WebView2). It is a thin renderer over
the same logic layer as the CLI — one data model, one SQLite file.

```sh
# one-time: Linux needs the webkit2gtk-4.1 dev headers
#   Debian/Ubuntu: sudo apt install libwebkit2gtk-4.1-dev
#   Arch: sudo pacman -S webkit2gtk-4.1
go build -tags "wails production webkit2_41" -o fieldkit-desktop ./cmd/fieldkit
./fieldkit-desktop desktop --data-dir ./data
```

The default (cgo-free) binary prints these instructions instead of
including the GUI, so the CLI binary keeps working in Docker/scratch/CI.

Both the GUI and the TUI open with a first-run wizard that walks setup
and the core workflow with real operations (it creates the demo survey,
answers it, and shows the export), then can be reopened any time from
the sidebar ("Help: open wizard") or with the `w` key.

The GUI covers: survey authoring from JSON, a respondent runner with
live branch logic, the response table with CSV download, the Typeform
migrate dry-run viewer, plugin manifest validation (errors shown, never
a crash), and the donate view with copy buttons.

## Commands

| Command | What it does |
|---|---|
| `fieldkit surveys` | create/list surveys from JSON definitions |
| `fieldkit serve` | HTTP server: public submit endpoint + `/donate` page |
| `fieldkit migrate` | dry-run import from Typeform exports (JSON report) |
| `fieldkit logic` | run survey logic offline against an answer file |
| `fieldkit export csv` | responses as CSV |
| `fieldkit plugin` | validate plugin manifests / run contract checks |
| `fieldkit version` | print version |
| `fieldkit donate` | print donation addresses |

## Survey definitions

A survey is a JSON document. Questions support text, single choice,
multi choice, rating, and NPS types, required flags, hidden fields,
and branch rules. See `testdata/logic/` for working examples and
`docs/plugins.md` for the plugin manifest format.

## Plugin contract

Plugins are JSON-manifested extensions with a versioned ABI. A sample
plugin ships in-repo (`testdata/plugins/sample/`) and its contract test
runs in CI — if the sample plugin does not pass, the build fails. The
community does not need to be trusted for the contract to hold; see
`docs/plugins.md`.

## Donations

Fieldkit is free to self-host, with no feature gated on money. If it
saves you a survey-platform line item, donations are welcome:

- Ethereum / USDC (ERC-20): `0x85ee7E71f762d772599cbF1EC20E651B30657521`
- Bitcoin: `bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg`

No other wallet addresses are associated with this project. Details:
`docs/donate.md`.

## Status

**Currently working:** survey model + SQLite store, logic engine with
branching/required/hidden handling, public submit endpoint with
validation errors, CSV export, Typeform migrate dry-run, plugin
manifest contract, `version`, `donate`.

**Planned:** Qualtrics migrate (gated on green Typeform fidelity),
Wails desktop GUI, Bubble Tea TUI — each with a first-run wizard.

## License

MIT. See LICENSE. Contribution policy: DCO sign-off only, no CLA — see
GOVERNANCE.md.
