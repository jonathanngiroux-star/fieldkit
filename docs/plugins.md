# Plugins

A Fieldkit plugin is a directory (or zip) with a `manifest.json` and an
entrypoint file. The contract is versioned and CI-enforced: the sample
plugin in `testdata/plugins/sample/` must pass the contract check or the
build fails. A plugin contract nobody exercises is a contract nobody has
— the sample exists so the check has a real subject.

## Manifest (api_version 1)

```json
{
  "api_version": 1,
  "name": "sample",
  "version": "1.0.0",
  "kind": "question_type",
  "description": "optional human text",
  "entrypoint": "sample.js"
}
```

Rules:

- `api_version` must be exactly `1`. Anything else is rejected by name.
- `name`: lowercase letters, digits, dashes.
- `version`: `MAJOR.MINOR.PATCH`.
- `kind`: one of `question_type`, `webhook`, `scorer`.
- `entrypoint`: a relative path inside the plugin directory; no `..`,
  no absolute paths; the file must exist and be non-empty.

## Validate

```sh
fieldkit plugin validate testdata/plugins/sample
# ok sample 1.0.0 (question_type, api_version 1)
```

## What each kind means (v1 scope)

- **question_type** — the host renders and validates a custom question
  type (the documented answer for Typeform `matrix` fields: the dry-run
  lists them as unmapped and points here). The entrypoint exports
  `render(el, question, answer)` and `validate(answer, question)`.
- **webhook** — reserved: the manifest validates today; delivery wiring
  is deliberately not in v0.1 scope.
- **scorer** — reserved: custom scoring over a submitted response set.

Reserved kinds validate so the manifest surface is stable, but v0.1
ships no runtime for them — that honesty is the contract's point.

## Why not "the community will write integrations"

Because an unversioned, untested extension surface is how solo projects
die: every core change silently breaks downstream plugins, and there is
no one to fix them. Fieldkit's answer is a narrow, versioned, CI-tested
contract with one in-repo sample — small surface, hard guarantees.
