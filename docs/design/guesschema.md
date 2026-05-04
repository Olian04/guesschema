# guesschema — design notes

## Problem

Infer a practical **JSON Schema (2020-12)** from a stream of **JSONL** examples: types, nested objects, arrays, and rough “optional vs required” hints from presence/absence across lines.

Non-goals: validating that upstream data always matches the guess; Kafka/HTTP bridges; config files.

## CLI

| Flag | Default | Notes |
| --- | --- | --- |
| `--read-window` | **1s** if omitted | Max wall time reading stdin per window. |
| `--once` | implicit when no `--every` | Single emit then exit. |
| `--every` | off | Periodic NDJSON emits; **`every` ≥ read-window** with defaults. |
| `--variant-threshold` | **0.1** | Same path: `oneOf` only if **every** variant has **likelihood > T**. |
| `--no-extra` | off | Strip object keys starting with **`x-`** (vendor extensions) from stdout JSON. |
| `--debug` | off | stderr only; no `-d` (avoids clashing with `-v` / version). |

## Scheduling (`--every`)

- **Ticker** with period **`every`**.
- **`lastWindowStarted`**: set when a read window **starts** (including the first, immediate cycle).
- **Busy:** from read start through emit, stdout flush, accumulator reset.
- On tick: if busy → no-op; else if **`now - lastWindowStarted >= every`** → start next window and refresh **`lastWindowStarted`**.
- **Not** modeled as “sleep **`every`** after each cycle completes” (that would add roughly **`read-window`** to spacing between starts).

## Accumulator model

Property row = **`(path_key, value_type, value_hint)`** with **`lines_with`** counts. **`path_key`**: RFC 6901 pointer from each line’s root.

**Missing keys** at object pointer `P`: maintain **`knownKeys[P]`** and **`linesCompletedBeforeCurrent`**. On each line:

1. `knownKeys[P] \ present` → **`undefined`** tuple +1 per missing key.
2. `present \ knownKeys[P]` → **`undefined` + `linesCompletedBeforeCurrent`**, then concrete +1, add key to **`knownKeys[P]`**.
3. `present ∩ knownKeys[P]` → concrete +1 only.

## Materialization

- **Strategy B (default):** single winning variant at a path: max **`lines_with`**, tie-break higher **`likelihood`**, then lexical **`(type, hint)`**.
- **Strategy A:** **`oneOf`** when **every** concrete variant at that path has **`likelihood` > `T`**.
- **`required`:** only object keys whose values appear on **every** successful line in the window (sum of concrete **`lines_with`** at that pointer equals **`lines_total`** — combined presence **likelihood 1**). That sum includes **all** variants that become **`oneOf`** branches (types can vary per line; each branch’s own likelihood can stay below 1 while the union still covers every line). All other declared **`properties`** are optional.

Stats **`x-guesschema-*`** live as **siblings** on emitted variant subschemas. Root always includes **`x-guesschema-generated-at`** (RFC3339) and **`x-guesschema-invalid-json-lines`**.

## Decision log

| Decision | Rationale |
| --- | --- |
| Releases ship **binaries + Go install paths**; **no** container image in CI | Primary use: `go install` / `go get -tool` + `go tool`; GitHub Release still publishes cross-compiled binaries + checksums + SBOMs. |
| Default read window **1s** | Bounded work without a separate warmup flag. |
| `--debug` long-only | Avoid `-v` vs version shorthand collisions. |
| `time.Ticker` + `lastWindowStarted` | Match “frequency dial between window starts” semantics. |
| Draft **2020-12** | Stable `$schema` URL and widely supported core keywords. |

## References

- [Repository](https://github.com/Olian04/guesschema) · module `github.com/Olian04/guesschema`
- [RFC 6901](https://www.rfc-editor.org/rfc/rfc6901)
- [JSON Schema 2020-12](https://json-schema.org/draft/2020-12/json-schema-core.html)
