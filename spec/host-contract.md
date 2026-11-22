# Host contract

## Invocation

| Command                  | Behaviour                                                    |
| ------------------------ | ------------------------------------------------------------ |
| `jscheme`                | Start the REPL on stdin.                                     |
| `jscheme <file.js>`      | Evaluate a file. |
| `jscheme -e <source>`    | Evaluate `<source>`.              |

## Exit codes

| Code | Meaning                                                        |
| ---- | -------------------------------------------------------------- |
| 0    | Program executed successfully.                                 |
| 1    | Runtime error (spec §16).                                      |
| 2    | Syntax error, detected before execution (spec §16).            |
| 3    | Usage or I/O error: bad arguments, missing or unreadable file. |

Code 2 also covers errors the implementation detects before execution even though the spec only requires them at run time:
§16 allows errors to be reported sooner (e.g. `1 = 2;`). For such errors the tests accept code 1 or 2.

For codes 1–3 a message is written to **stderr**.
Its text is free-form and is never asserted on by the tests; only its presence is.

## The `print` function

The global object has one host-provided function in addition to the spec's built-ins:

- `print(...args)` converts each argument with ToString,
  joins them with a single space, and writes the result plus `"\n"` to stdout.

## REPL

`jscheme` with no arguments reads from stdin:

- The prompt `>` is shown **only when stdin is a TTY** (piped input, as used by the tests, sees no prompt).
- All inputs are evaluated in **one shared global environment**.
- An input may span multiple lines; evaluation happens once a complete statement has been read.
- After each evaluated input, the completion value is printed with ToString, unless it is `undefined`.
- Errors are written to stderr and the session **continues**.
- On EOF the REPL exits with code 0.

## Implementation-defined behaviour

Where the spec leaves a choice to the implementation, this project pins it down:

| Area                         | Choice                                                         |
| ---------------------------- | -------------------------------------------------------------- |
| Local time zone (§15.9)      | The test harness runs the interpreter with `TZ=UTC`.           |
| Number → String (§9.8.1)     | Shortest digit string that round-trips (e.g. `0.1 + 0.2` → `0.30000000000000004`). |
| `Math.random()` (§15.8)      | Any value in [0, 1); tests assert only the range.              |
| `Function.prototype.toString` (§15.3) | Has the syntax of a *FunctionDeclaration*; tests assert only the form. |
| `Number.prototype.toString(radix)` (§15.7.4.2) | For an integer value: the usual positional digits in that radix, lowercase letters, a leading `-` if negative (e.g. `(255).toString(16)` → `ff`). Fractional values are not asserted. |

Any further implementation-defined choice the tests depend on must be added to this table.
