# JScheme — agent instructions

A JavaScript interpreter for ECMA-262 1st edition (`spec/`), written in Chez Scheme (`src/`), judged by a `go test` suite (`tests/`). See README.md.

## Ownership

- You write only `src/` (and `bin/` through `mise run build`).
- `spec/`, `tests/`, README.md, this file, mise config and `.claude/` are human-owned and read-only. `.claude/settings.json` enforces this; don't try to get around it.
- The spec is the oracle. If a test contradicts the spec, stop and report it with the spec section. Don't special-case the implementation to satisfy it.

## No JavaScript runtime

Never run, install or call node, bun, deno, qjs, npm, npx, pnpm, yarn or any other JS engine, and don't add Go dependencies. This overrides any global rule that suggests `npx` (e.g. ctx7). Behaviour comes from the spec, not from comparing against an engine.

## Loop

```sh
mise run build            # src/ -> bin/jscheme
mise run test             # build + full suite (test:verbose for per-test output)
bin/jscheme -e '<source>' # try a snippet
```

`mise run test:stub` and `mise run coverage` are suite-health gates owned by humans.
Host behaviour (exit codes, `print`, REPL) is defined in `spec/host-contract.md`.

## Chez Scheme gotcha

Top-level definitions in `src/` must not reuse names exported by `(chezscheme)`. A clash fails to compile or silently shadows a builtin.
