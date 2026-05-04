# Agent context: guesschema

Go module **`github.com/Olian04/guesschema`** ([repo](https://github.com/Olian04/guesschema)). CLI: **JSONL stdin → guessed JSON Schema 2020-12 stdout**; optional **`--debug`** on stderr. No HTTP server, no Prometheus, no YAML config in this repo shape.

## Layout

| Path | Role |
| --- | --- |
| `cmd/guesschema` | CLI entry: urfave/cli v3, signal context, `app.RunGuesschema`. |
| `cmd/guesschema/version` | `Version`, `Revision`, `BuildTime` for `-ldflags -X` and `--version`. |
| `internal/app` | Stdin read loop, read-window timer, `--every` ticker scheduling, stdout emit + flush. |
| `internal/domain/schemaguess` | Cross-line accumulator (RFC 6901 paths, missing-key rules), materialization (winner vs `oneOf` by `--variant-threshold`). |
| `test/unit/...` | Unit tests beside mirrored paths under `test/unit/domain/...` and `test/unit/app/...`. |

## Dependency direction

`cmd/guesschema` → `internal/app` → `internal/domain/schemaguess`. Domain must not import `internal/app`.

## Behavior summary

- **`--no-extra`:** omit vendor extensions by stripping JSON object keys starting with **`x-`** before stdout (recursive).
- **`--once` (default):** read at most **`--read-window`** (default **1s**) from process start, emit once, exit; EOF before budget → emit on EOF.
- **`--every`:** `time.Ticker` period **`every`**; **`lastWindowStarted`** updated when a read window **starts**; on each tick, if not busy and **`now - lastWindowStarted >= every`**, start the next window. While read→emit→reset is in progress, ticks are no-ops. First cycle starts immediately.
- **`effective read-window <= every`** when periodic (defaults imply **`every` ≥ 1s**).
- **`--once` + `--every`** → error.

## Errors and logging

Use `%w` when wrapping errors in `cmd` / `internal/app`. Default logger discarded unless **`--debug`** (stderr text handler).

## Commands (`make`)

`build` → `./dist/guesschema`, `run`, `test`, `test-race`, `lint`, `format`.

## Releases

GoReleaser publishes **pre-built binaries** and SBOMs to GitHub Releases (no Docker image). Install from source with **`go install github.com/Olian04/guesschema/cmd/guesschema@<tag>`** or pin with **`go get -tool`** and run **`go tool github.com/Olian04/guesschema/cmd/guesschema`**. This module’s **`go.mod`** `tool` block includes `cmd/guesschema` for **`go tool`** from a checkout.

## Go proverbs ([source](https://go-proverbs.github.io/), compressed)

Concurrency channels coordinate mutex serializes · not parallelism · small interface sharp · zero value useful · `any` untyped tame it · gofmt settles bikeshed · tiny copy beats dep hairball syscall/cgo build tags isolate · cgo not Go · `unsafe` no contract · clarity beats wit · reflection stay cold path · errors values inspect wrap once · architecture name docs users · panic stays in `main` / hard startup.

## Uber style distill ([guide](https://github.com/uber-go/guide/blob/master/style.md), compressed)

Rare `*Iface` · `var _ I = (*T)(nil)` at export boundary · defer unlock pairs · chan buffer zero or one usually · slice/map copy exported API boundaries · typed errors `%w` chain handle once · assert comma-ok · goroutine bounded ctx/waitgroup · no zombie `init()` · globals inject not mutate · exits from `main` only · strconv hot paths · structs field-named literals · table tests sub `t.Run`.

Canon links above beat bullet memory when tradeoff unclear.
