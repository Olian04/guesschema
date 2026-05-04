# guesschema

Repository: [github.com/Olian04/guesschema](https://github.com/Olian04/guesschema) · Go module **`github.com/Olian04/guesschema`**

`guesschema` reads **JSON Lines** from **stdin** and writes a guessed **JSON Schema (draft 2020-12)** document to **stdout**. Optional **`--debug`** logs to **stderr** (long flag only).

## Install / build

**From this repo (development):**

```bash
make build              # ./dist/guesschema
go install ./cmd/guesschema
go tool github.com/Olian04/guesschema/cmd/guesschema --help   # uses go.mod tool block
```

**Released versions** (after a GitHub tag exists):

```bash
# Global install (binary on $GOBIN or $GOPATH/bin)
go install github.com/Olian04/guesschema/cmd/guesschema@v0.1.0

# Pin in another module and run via the Go toolchain
go get -tool github.com/Olian04/guesschema/cmd/guesschema@v0.1.0
go tool github.com/Olian04/guesschema/cmd/guesschema --help
```

Use `@latest` or a concrete tag. Each GitHub **Release** also attaches **pre-built** `guesschema-<os>-<arch>` binaries plus `checksums.txt` and SBOMs (see GoReleaser release notes).

## Usage

**Default (single emit):** read for at most **`--read-window`** (default **1s** if omitted), emit one schema, exit. If **EOF** arrives first, emit on EOF.

```bash
printf '%s\n' '{"id":1,"name":"a"}' '{"id":2,"name":"b"}' | ./dist/guesschema
```

**Periodic:** **`--every`** runs repeated read → emit → reset cycles. Spacing targets **time between window starts** (ticker + `lastWindowStarted`, not “sleep `every` after each cycle”). Requires **`every` ≥ default read window** (so at least **1s** with default **`--read-window`**).

```bash
printf '%s\n' '{"x":1}' | ./dist/guesschema --every 2s --read-window 1s
```

Each stdout line is one JSON schema (NDJSON) in periodic mode.

## Flags

| Flag | Meaning |
| --- | --- |
| `--once` | Explicit single-shot (default when `--every` is not set). |
| `--every <duration>` | Periodic mode; mutually exclusive with `--once`. |
| `--read-window <duration>` | Max wall time to read per window (default **1s** if omitted). |
| `--variant-threshold <float>` | **T** for same-path `oneOf` vs single winner (default **0.1**). |
| `--no-extra` | Strip vendor extensions: remove object keys starting with **`x-`** from stdout JSON. |
| `--debug` | stderr **slog** (no short alias). |
| `-v` / `--version` | Version (urfave/cli). |

## Schema output

- Root **`$schema`**: `https://json-schema.org/draft/2020-12/schema`
- **`x-guesschema-generated-at`**: RFC3339 on every emit
- **`x-guesschema-invalid-json-lines`**: invalid JSONL count at root
- Per-variant stats as siblings: **`x-guesschema-lines-with`**, **`x-guesschema-lines-total`**, **`x-guesschema-likelihood`**

Paths follow [RFC 6901](https://www.rfc-editor.org/rfc/rfc6901) from each line’s JSON root.

## Agent / contributor context

See [docs/AGENTS.md](docs/AGENTS.md) and [docs/design/guesschema.md](docs/design/guesschema.md).
