package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Olian04/guesschema/internal/domain/schemaguess"
)

const defaultReadWindow = time.Second

const maxJSONLLineBytes = 16 << 20

// GuesschemaConfig drives stdin JSONL scanning and schema emission.
type GuesschemaConfig struct {
	ReadWindow       time.Duration // max read time per phase; zero => default 1s
	Every            time.Duration // zero => once mode; else periodic
	VariantThreshold float64       // T for oneOf vs winner (default 0.1 in cmd)
	NoExtra          bool          // strip JSON object keys starting with "x-" before stdout
	StartOnNextMsg   bool          // start each read-window only after first received line
	Debug            bool
}

// RunGuesschema reads JSONL from in, writes JSON Schema 2020-12 to out (NDJSON when periodic).
func RunGuesschema(ctx context.Context, in io.Reader, out io.Writer, cfg GuesschemaConfig) error {
	if cfg.Every > 0 {
		return runPeriodic(ctx, in, out, cfg)
	}
	return runOnce(ctx, in, out, cfg)
}

func effectiveReadWindow(cfg GuesschemaConfig) time.Duration {
	if cfg.ReadWindow > 0 {
		return cfg.ReadWindow
	}
	return defaultReadWindow
}

func runOnce(ctx context.Context, in io.Reader, out io.Writer, cfg GuesschemaConfig) error {
	acc := schemaguess.NewAccumulator()
	rw := effectiveReadWindow(cfg)
	var (
		eof bool
		err error
	)
	if _, ok := probeReadDeadlineSupport(in); ok {
		eof, err = readPhase(ctx, in, acc, rw)
	} else {
		pump := startLinePump(ctx, in)
		eof, err = readPhaseFromPump(ctx, acc, rw, pump.lines, pump.errs, cfg.StartOnNextMsg)
	}
	if err != nil {
		return err
	}
	if err := emitSchema(out, acc, cfg); err != nil {
		return err
	}
	if cfg.Debug && !eof {
		slog.Info("guesschema once: read window elapsed before EOF")
	}
	return nil
}

func runPeriodic(ctx context.Context, in io.Reader, out io.Writer, cfg GuesschemaConfig) error {
	every := cfg.Every
	rw := effectiveReadWindow(cfg)
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	acc := schemaguess.NewAccumulator()
	var lastStart time.Time
	_, deadlineSupported := probeReadDeadlineSupport(in)
	var pump linePump
	if !deadlineSupported {
		pump = startLinePump(ctx, in)
	}

	for cycle := 0; ; cycle++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if cycle > 0 {
		waitTick:
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
					if time.Since(lastStart) >= every {
						break waitTick
					}
				}
			}
		}
		lastStart = time.Now()
		acc.Reset()
		if cfg.Debug {
			slog.Info("guesschema periodic cycle", "cycle", cycle, "read_window", rw, "every", every)
		}
		var (
			eof bool
			err error
		)
		if deadlineSupported {
			eof, err = readPhase(ctx, in, acc, rw)
		} else {
			eof, err = readPhaseFromPump(ctx, acc, rw, pump.lines, pump.errs, cfg.StartOnNextMsg)
		}
		if err != nil {
			return err
		}
		if err := emitSchema(out, acc, cfg); err != nil {
			return err
		}
		if eof {
			return nil
		}
	}
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

func probeReadDeadlineSupport(in io.Reader) (*os.File, bool) {
	fin, isFile := in.(*os.File)
	if !isFile {
		return nil, false
	}
	if err := fin.SetReadDeadline(time.Now().Add(time.Millisecond)); err != nil {
		_ = fin.SetReadDeadline(time.Time{})
		return fin, false
	}
	_ = fin.SetReadDeadline(time.Time{})
	return fin, true
}

type linePump struct {
	lines <-chan []byte
	errs  <-chan error
}

func startLinePump(ctx context.Context, in io.Reader) linePump {
	lines := make(chan []byte, 16)
	errs := make(chan error, 1)
	br := bufio.NewReaderSize(in, 64*1024)
	go func() {
		defer close(lines)
		defer close(errs)
		for {
			line, err := br.ReadBytes('\n')
			if len(line) > maxJSONLLineBytes {
				errs <- errors.New("jsonl line exceeds max size")
				return
			}
			if len(line) > 0 {
				select {
				case lines <- trimCRLF(line):
				case <-ctx.Done():
					return
				}
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				errs <- err
				return
			}
		}
	}()
	return linePump{lines: lines, errs: errs}
}

// readPhase accumulates until read window elapses, ctx cancelled, or EOF on stdin.
func readPhase(ctx context.Context, in io.Reader, acc *schemaguess.Accumulator, window time.Duration) (eof bool, err error) {
	deadEnd := time.Now().Add(window)
	br := bufio.NewReaderSize(in, 64*1024)
	fin, isFile := in.(*os.File)

	// Some stdin/pipe/terminal types reject SetReadDeadline ("file type does not support deadline").
	readDeadlineOK := false
	if isFile {
		if err := fin.SetReadDeadline(time.Now().Add(time.Millisecond)); err == nil {
			readDeadlineOK = true
		}
		_ = fin.SetReadDeadline(time.Time{})
	}

	if readDeadlineOK {
		defer func() { _ = fin.SetReadDeadline(time.Time{}) }()
		return readPhaseWithReadDeadline(ctx, br, fin, acc, deadEnd)
	}
	return readPhaseWithoutReadDeadline(ctx, br, acc, deadEnd)
}

func readPhaseWithReadDeadline(ctx context.Context, br *bufio.Reader, fin *os.File, acc *schemaguess.Accumulator, deadEnd time.Time) (eof bool, err error) {
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		if !time.Now().Before(deadEnd) {
			return false, nil
		}

		slice := 100 * time.Millisecond
		if rem := time.Until(deadEnd); rem < slice {
			slice = rem
		}
		if slice <= 0 {
			return false, nil
		}
		if err := fin.SetReadDeadline(time.Now().Add(slice)); err != nil {
			return false, err
		}

		line, err := br.ReadBytes('\n')
		if len(line) > maxJSONLLineBytes {
			return false, errors.New("jsonl line exceeds max size")
		}
		line = trimCRLF(line)
		if err != nil {
			if isTimeout(err) {
				if len(line) > 0 {
					if err2 := acc.ObserveLine(line); err2 != nil {
						return false, err2
					}
				}
				if !time.Now().Before(deadEnd) {
					return false, nil
				}
				continue
			}
			if errors.Is(err, io.EOF) {
				if len(line) > 0 {
					if err2 := acc.ObserveLine(line); err2 != nil {
						return false, err2
					}
				}
				return true, nil
			}
			return false, err
		}
		if err := acc.ObserveLine(line); err != nil {
			return false, err
		}
	}
}

// readPhaseWithoutReadDeadline runs when SetReadDeadline is not supported on this reader.
// The read budget is enforced between lines (a blocking ReadBytes may exceed deadEnd).
func readPhaseWithoutReadDeadline(ctx context.Context, br *bufio.Reader, acc *schemaguess.Accumulator, deadEnd time.Time) (eof bool, err error) {
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		if !time.Now().Before(deadEnd) {
			return false, nil
		}

		line, err := br.ReadBytes('\n')
		if len(line) > maxJSONLLineBytes {
			return false, errors.New("jsonl line exceeds max size")
		}
		line = trimCRLF(line)
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(line) > 0 {
					if err2 := acc.ObserveLine(line); err2 != nil {
						return false, err2
					}
				}
				return true, nil
			}
			return false, err
		}
		if err := acc.ObserveLine(line); err != nil {
			return false, err
		}
	}
}

func readPhaseFromPump(ctx context.Context, acc *schemaguess.Accumulator, window time.Duration, lines <-chan []byte, errs <-chan error, startOnNextMsg bool) (eof bool, err error) {
	var (
		timer  *time.Timer
		timerC <-chan time.Time
	)
	if !startOnNextMsg {
		timer = time.NewTimer(window)
		timerC = timer.C
		defer timer.Stop()
	}
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-timerC:
			return false, nil
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				return false, err
			}
		case line, ok := <-lines:
			if !ok {
				return true, nil
			}
			if len(line) == 0 {
				continue
			}
			if startOnNextMsg && timer == nil {
				timer = time.NewTimer(window)
				timerC = timer.C
				defer timer.Stop()
			}
			if err := acc.ObserveLine(line); err != nil {
				return false, err
			}
		}
	}
}
func trimCRLF(line []byte) []byte {
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return line
}

func emitSchema(out io.Writer, acc *schemaguess.Accumulator, cfg GuesschemaConfig) error {
	doc := schemaguess.BuildSchema(acc, cfg.VariantThreshold, time.Now())
	if cfg.NoExtra {
		stripXVendorKeys(doc)
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if _, werr := out.Write(b); werr != nil {
		return werr
	}
	return nil
}

// stripXVendorKeys removes JSON object properties whose names start with "x-" (vendor extensions), recursively.
func stripXVendorKeys(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k := range t {
			if strings.HasPrefix(k, "x-") {
				delete(t, k)
			}
		}
		for _, val := range t {
			stripXVendorKeys(val)
		}
	case []any:
		for _, el := range t {
			stripXVendorKeys(el)
		}
	default:
		return
	}
}

// ValidateGuesschemaFlags checks CLI-derived durations and threshold.
func ValidateGuesschemaFlags(readWindow, every time.Duration, variantT float64, once, periodic bool) error {
	if once && periodic {
		return errors.New("cannot set both --once and --every")
	}
	if readWindow < 0 || every < 0 {
		return errors.New("durations must be non-negative")
	}
	if readWindow > 0 && readWindow < time.Millisecond {
		return errors.New("--read-window must be positive")
	}
	if periodic {
		effective := readWindow
		if effective == 0 {
			effective = defaultReadWindow
		}
		if every < defaultReadWindow {
			return errors.New("--every must be at least 1s when using default --read-window")
		}
		if effective > every {
			return errors.New("effective read-window must be <= --every")
		}
	}
	if variantT <= 0 || variantT >= 1 {
		return errors.New("--variant-threshold must satisfy 0 < T < 1")
	}
	return nil
}
