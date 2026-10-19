# AGENTS.md — Fieldkit working rules

Fieldkit is a solo-maintainable, self-hosted form/survey backend with a
versioned plugin contract and a migration CLI (Typeform first, Qualtrics
gated on a green Typeform fixture). The wedge is **migration + contracts**,
not another form builder.

## Monetization (binding, series-wide)

Donation-only. No subscriptions, no invoices, no seat math, no usage
bills, no "contact sales", no payment processor, no paid tier — ever.
The only ask:

- Ethereum / USDC (ERC-20): `0x85ee7E71f762d772599cbF1EC20E651B30657521`
- Bitcoin: `bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg`

Both addresses appear in README, `docs/donate.md`, `/donate` HTTP page,
every UI footer (GUI and TUI), and `fieldkit donate`. No other wallets
without the maintainer. No feature is ever gated on a donation.

The original Reality Seed map sold hosted tiers; that path is closed for
this project. Success = self-host deploys, migrate dry-run usage, fixture
quality, cold-start time, on-chain donations.

## Refusals (name the trap, then stop)

- Cloud SKU / pricing tier / Stripe → point at `docs/donate.md`
- "Community will write the integrations" → the sample plugin ships in-repo
- Generic form-builder features with no migrate/contract angle
- Kubernetes / multi-service self-host story
- Feature gates on the logic engine or export
- Qualtrics migrate work before the Typeform fixture is green

## Stack rules

- Go + SQLite (`modernc.org/sqlite`, pure-Go — no cgo), single binary.
- CLI: `fieldkit serve|migrate|logic|plugin|version|donate|tui|desktop|surveys|export` (all landed).
- Cold start under 15 minutes; one binary or one compose. Kubernetes is a
  bug, not a feature.
- License: MIT whole-repo. No BSL/ELv2 layer in this series.
- GUI (step 3+): Wails v2 desktop app, Go bindings — never a browser-only
  admin site. TUI: Bubble Tea.
- Tests call `run(args, stdout, stderr)` in-process; never exec the binary.

## Working rules

- TDD: failing test first, watch it fail, minimal code to pass.
- Every feature answers: does this help the wedge, deploy time, or donate
  visibility?
- Gates that run on every `go test ./...`: donate-exact, no-billing walk,
  fixture hygiene. Gates, not intent.
- Golden fixtures checked in; CI fails on drift.
- Module path `github.com/jonathanngiroux-star/fieldkit`, go 1.27.
- Internal briefs (`IDEA.md`, `NN-*.md`) are gitignored; they never ship.

## v0.1 definition of done

1. Survey model + logic engine (branching, required, hidden) with CI fixtures
2. Public submit endpoint, validation errors, SQLite responses, CSV export
3. Versioned plugin contract; sample plugin must pass or the build fails
4. `fieldkit migrate --from=typeform --dry-run --format=json`
5. Donate exact on every surface; no billing anywhere
6. Wails GUI + Bubble Tea TUI, each with a first-run wizard
7. `docs/audit/*.md` for backend, gui, tui, system
