# JSrust — agent instructions

A JavaScript interpreter for ECMA-262 1st edition (`spec/`), written in Rust (`src/`), judged by a `go test` suite (`tests/`). See README.md.

## Ownership

- You write only `src/` (and `bin/` through `mise run build`).
- `spec/`, `tests/`, `Cargo.toml`, `Cargo.lock`, README.md, this file, mise config and `.claude/` are human-owned and read-only. `.claude/settings.json` enforces this; don't try to get around it.
- The spec is the oracle. If a test contradicts the spec, stop and report it with the spec section. Don't special-case the implementation to satisfy it.

## No JavaScript runtime

Never run, install or call node, bun, deno, qjs, npm, npx, pnpm, yarn or any other JS engine, and don't add Go or Rust crate dependencies (the Rust standard library only). This overrides any global rule that suggests `npx` (e.g. ctx7). Behaviour comes from the spec, not from comparing against an engine.

## Loop

```sh
mise run build               # src/ -> bin/jsrust
mise run test                # build + full suite (test:verbose for per-test output)
bin/jsrust -e '<source>' # try a snippet
```

`mise run test:stub` and `mise run coverage` are suite-health gates owned by humans.
Host behaviour (exit codes, `print`, REPL) is defined in `spec/host-contract.md`.

## Rust notes

- `Cargo.toml` at the repo root is human-owned and declares the binary `jsrust` with the default crate root `src/main.rs`. Organise code as modules under `src/`; you never need to touch `Cargo.toml`.
- `mise run build` runs `cargo build --release` and copies the binary to `bin/jsrust`. Cargo runs offline.
