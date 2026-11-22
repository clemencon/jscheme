# JScheme

An **interpreter for JavaScript** as defined by [ECMA-262, 1st edition](/spec/ECMA-262_1st_edition_june_1997.pdf), implemented in [Scheme](https://en.wikipedia.org/wiki/History_of_the_Scheme_programming_language).

How it is built:

> Humans own the spec and the tests.
> An agent writes the entire implementation, driven only by the test feedback loop.
> No human ever reads the implementation.

## Principles

1. **The spec is the oracle.**
   ECMA-262 1st edition is the source of truth.
2. **The spec beats the tests.**
   A test that contradicts the spec is a bug in the test.
3. **The implementation is a black box.**
   Humans do not read `src/`, correctness is judged by the tests.

## Usage

```sh
jscheme              # Start the REPL on stdin.
jscheme <file.js>    # Evaluate a file.
jscheme -e <source>  # Evaluate `<source>`.
```

## Getting started

Requires [mise](https://mise.jdx.dev/), which provides Scheme and Go.
No JavaScript runtime is part of the toolchain.

```sh
mise install         # Install pinned tool versions.
mise run build       # Build bin/jscheme from src/.
mise run test        # Build, then run go test against bin/jscheme.
```

`mise run test:stub` checks that every test fails against an empty binary,
`mise run coverage` checks that every spec section is named by a test.
To run the suite against another binary: `JSCHEME_BIN=/path/to/jscheme go test ./...` (from `tests/`).

## Repository layout and ownership

| Path                  | Contents                                          | Owner                          |
| --------------------- | ------------------------------------------------- | ------------------------------ |
| `spec/`               | Spec PDF (canonical) + markdown, host contract | Humans |
| `tests/`              | `go test` suite | Humans |
| `src/`                | Interpreter implementation                    | Agent |


## Host contract

ECMA-262 leaves input/output and program invocation to the host environment (§4).

See [spec/host-contract.md](spec/host-contract.md) for invocation, exit codes, the `print` function, the REPL and implementation-defined behaviour.
