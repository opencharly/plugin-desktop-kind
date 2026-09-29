# AGENTS.md — plugin-desktop-kind

Standalone plugin repo serving the four desktop-surface kinds (`kind:theme`,
`kind:session`, `kind:displaymanager`, `kind:desktopentry`). The plugin is a Go
module at `candy/plugin-desktop-kind/` (module path
`github.com/opencharly/plugin-desktop-kind/candy/plugin-desktop-kind`); the root
`charly.yml` declares `repo:` and `discover: candy` so the repo is a project and
its candy is scanned.

Canonical files:

- `candy/plugin-desktop-kind/charly.yml` — the `plugin-desktop-kind:` candy
  entity (`plugin:` block, `plan:` check).
- `candy/plugin-desktop-kind/plugin.go` — the kind provider for all four words
  (`Invoke(OpLoad)` / `Invoke(OpValidate)`) and `NewProvider()`/`NewMeta()`.
- `candy/plugin-desktop-kind/schema/desktop.cue` — the self-contained
  `#ThemeInput` / `#SessionInput` / `#DisplayManagerInput` / `#DesktopEntryInput`.
- `candy/plugin-desktop-kind/plugin_test.go` — drives the real `Invoke` for all
  four words and both validators, in each direction.
- `candy/plugin-desktop-kind/schema_parity_test.go` — asserts the served
  `#*Input` defs still match spec's authored surface (the drift guard).
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `README.md` — user overview only; never agent guidance.

## Load these skills first (R0)

- `/charly-image:layer` — the candy authoring surface the desktop kinds extend.
  Load before changing a kind's vocabulary. This candy carries no `skill:` entity
  of its own; the gap is tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- `/charly-internals:plugin` — the plugin authoring reference: the `plugin:`
  block, the `kind` provider class, the per-plugin CUE-schema contract.
- `/charly-internals:git-workflow` — before any git/PR action.

## Build / validate / test

- `go build ./...` in `candy/plugin-desktop-kind/` — compile the plugin module.
- `go test ./...` in `candy/plugin-desktop-kind/` — the plugin's Go tests (the
  four-word `Invoke` + the two validators + the schema-parity guard).
- `charly box validate` at the repo root — the structural check (the candy +
  `plugin:` block, CUE schema).
- The merge gate is the **org-wide** `charly/pr-validator` (required check
  `validate / validate`, defined in `opencharly/.github`); this repo has **no**
  per-repo candy gate.
- The candy contributes no image content; the substantive gates are the Go tests
  and the changed path is exercised by any box/deploy composing one of the four
  desktop nodes.

## Modify this repo

- Edit the `plugin-desktop-kind:` candy entity, the Go source, and
  `schema/desktop.cue` **together** — the schema is the served declaration
  surface. The schema is a SANCTIONED copy of spec's `#Theme`/`#Session`/
  `#DisplayManager`/`#DesktopEntry` (it must compile standalone, which spec's
  codegen defs cannot); `schema_parity_test.go` is what keeps the two from
  drifting.
- The `OpValidate` checks are the whole reason these are kinds rather than
  `write:` steps — a token/exec-XOR-url/session cross-check is not optional
  decoration.

## Landing

- PR-only. Every change lands through a pull request; the org-required
  `charly/pr-validator` validates the diff and body and arms native auto-merge on
  PASS. Direct pushes to `main` are blocked.
- History lives in `CHANGELOG/` (written by `tag-on-merge` at merge time); the PR
  body IS the changelog.
- The authoritative rulebook is the umbrella `AGENTS.md` in
  `opencharly/opencharly` and `charly/AGENTS.md` in the charly repo. Do not
  restate its rules here.
