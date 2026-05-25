# guesschema

Repository: [github.com/Olian04/guesschema](https://github.com/Olian04/guesschema) · Go module **`github.com/Olian04/guesschema`**

`guesschema` reads **JSON Lines** from **stdin** and writes a guessed **JSON Schema (draft 2020-12)** document to **stdout**. Optional **`--debug`** logs to **stderr** (long flag only).

## Install / build

```bash
# Global install (binary on $GOBIN or $GOPATH/bin)
go install github.com/Olian04/guesschema/cmd/guesschema@latest

# Pin in another module and run via the Go toolchain
go get -tool github.com/Olian04/guesschema/cmd/guesschema@latest
go tool guesschema --help
```

Use `@latest` or a concrete tag. Each GitHub **Release** also attaches **pre-built** `guesschema-<os>-<arch>` binaries plus `checksums.txt` and SBOMs (see GoReleaser release notes).

## Usage

Read for at most **`--read-window`** (default **1s**), emit one schema, exit. If **EOF** arrives first, emit on EOF.

```bash
printf '%s\n' '{"id":1,"name":"a"}' '{"id":2,"name":"b"}' | ./dist/guesschema
```

## Flags

| Flag                             | Meaning                                                                                                        |
| -------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `--read-window <duration>`       | Max wall time to read JSONL (default **1s**).                                                                  |
| `--variant-threshold <float>`    | **T** for same-path `oneOf` vs single winner (default **0.1**).                                                |
| `--no-extra`                     | Strip vendor extensions: remove object keys starting with **`x-`** from stdout JSON.                           |
| `--start-window-on-next-message` | Start each read-window only after first received JSONL line. Useful to avoid empty-window emits on idle stdin. |
| `--debug`                        | stderr **slog** (no short alias).                                                                              |
| `-v` / `--version`               | Version (urfave/cli).                                                                                          |

## Schema output

- Root **`$schema`**: `https://json-schema.org/draft/2020-12/schema`
- **`x-guesschema-generated-at`**: RFC3339 on every emit
- **`x-guesschema-invalid-json-lines`**: invalid JSONL count at root
- Per-variant stats as siblings: **`x-guesschema-lines-with`**, **`x-guesschema-lines-total`**, **`x-guesschema-likelihood`**

Paths follow [RFC 6901](https://www.rfc-editor.org/rfc/rfc6901) from each line’s JSON root.

## Library

Import **`github.com/Olian04/guesschema/pkg/guesschema`**. Build a **`Guesser`** with **`New`** and functional options, then **`Run`** per JSONL stream:

```go
import (
    "bytes"
    "context"
    "strings"
    "time"

    "github.com/Olian04/guesschema/pkg/guesschema"
)

g, err := guesschema.New(
    guesschema.WithReadWindow(time.Second),
    guesschema.WithVariantThreshold(0.1),
)
if err != nil {
    return err
}
var out bytes.Buffer
return g.Run(ctx, strings.NewReader("{\"a\":1}\n"), &out)
```

Lower-level schema inference without time windows uses the same package: **`NewAccumulator`**, **`BuildSchema`**, etc. See godoc on **`With*`** options for streaming examples.

## Testing

**All tests live under `test/`** — no `*_test.go` beside production code (`cmd/`, `internal/`, or `pkg/`). Tests assert **consumer contracts** only, not `internal/` implementation.

| Layer | What is tested | Where |
| --- | --- | --- |
| **Library** | Public API: `pkg/guesschema` (`New`, `Guesser.Run`, `With*`, `NewAccumulator`, `BuildSchema`) | `test/unit/pkg/guesschema/` |
| **CLI** | Compiled `cmd/guesschema` binary (stdin/stdout/flags, including invalid flags), same as a shell pipeline | `test/blackbox/` |

Do **not** import `internal/guesschema` from tests. Run **`make test`** or **`go test ./...`**.

## Agent / contributor context

See [docs/AGENTS.md](docs/AGENTS.md) and [docs/design/guesschema.md](docs/design/guesschema.md).
