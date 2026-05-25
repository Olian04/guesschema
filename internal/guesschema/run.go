package guesschema

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

const maxJSONLLineBytes = 16 << 20

// Run reads JSONL from in, writes one JSON Schema 2020-12 document to out.
// If ctx is canceled (e.g. SIGINT), reading stops and the schema built so far is still written.
func (g *Guesser) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	acc := NewAccumulator()
	var (
		eof bool
		err error
	)
	if _, isFile := in.(*os.File); isFile {
		eof, err = readPhase(ctx, in, acc, g.cfg.readWindow)
	} else {
		pump := startLinePump(ctx, in)
		eof, err = readPhaseFromPump(ctx, acc, g.cfg.readWindow, pump.lines, pump.errs, g.cfg.startOnNextMsg)
	}
	readCanceled := err != nil && errors.Is(err, context.Canceled)
	if err != nil && !readCanceled {
		return err
	}
	emitCtx := ctx
	if readCanceled {
		// Reading stopped because ctx was canceled; still materialize lines observed so far.
		// WithoutCancel keeps deadlines but ignores cancel so emit is not aborted at buildSchema entry.
		// materializeAt still checks the original ctx for prompt abort during a long materialize.
		emitCtx = context.WithoutCancel(ctx)
	}
	emitErr := g.emitSchema(emitCtx, out, acc)
	emitCanceled := emitErr != nil && errors.Is(emitErr, context.Canceled)
	if emitErr != nil && !emitCanceled {
		return emitErr
	}
	if emitCanceled {
		readCanceled = true
	}
	if readCanceled {
		g.cfg.logger.Debug("guesschema: interrupted, emitting partial schema")
		return nil
	}
	if !eof {
		g.cfg.logger.Debug("guesschema: read window elapsed before EOF")
	}
	return nil
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

type linePump struct {
	lines <-chan []byte
	errs  <-chan error
}

func startLinePump(ctx context.Context, in io.Reader) linePump {
	lines := make(chan []byte, 16)
	errs := make(chan error, 1)
	br := bufio.NewReaderSize(in, 64*1024)
	fin, isFile := in.(*os.File)
	go func() {
		defer close(lines)
		defer close(errs)
		if isFile && fileReadDeadlineOK(fin) {
			pumpFileWithDeadline(ctx, br, fin, lines, errs)
			return
		}
		pumpReader(ctx, br, lines, errs)
	}()
	return linePump{lines: lines, errs: errs}
}

func pumpReader(ctx context.Context, br *bufio.Reader, lines chan<- []byte, errs chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
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
}

func pumpFileWithDeadline(ctx context.Context, br *bufio.Reader, fin *os.File, lines chan<- []byte, errs chan<- error) {
	defer func() { _ = fin.SetReadDeadline(time.Time{}) }()
	const slice = 100 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := fin.SetReadDeadline(time.Now().Add(slice)); err != nil {
			errs <- err
			return
		}
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
			if isTimeout(err) {
				select {
				case <-ctx.Done():
					return
				default:
				}
				continue
			}
			if errors.Is(err, io.EOF) {
				return
			}
			errs <- err
			return
		}
	}
}

func fileReadDeadlineOK(fin *os.File) bool {
	if err := fin.SetReadDeadline(time.Now().Add(time.Millisecond)); err != nil {
		_ = fin.SetReadDeadline(time.Time{})
		return false
	}
	_ = fin.SetReadDeadline(time.Time{})
	return true
}

func readPhase(ctx context.Context, in io.Reader, acc *Accumulator, window time.Duration) (eof bool, err error) {
	fin, ok := in.(*os.File)
	if !ok {
		return false, errors.New("readPhase requires *os.File")
	}
	deadEnd := time.Now().Add(window)
	br := bufio.NewReaderSize(in, 64*1024)
	deadlineOK := fileReadDeadlineOK(fin)
	if deadlineOK {
		defer func() { _ = fin.SetReadDeadline(time.Time{}) }()
		return readPhaseWithReadDeadline(ctx, br, fin, acc, deadEnd)
	}
	return readPhaseBlocking(ctx, br, acc, deadEnd)
}

func readPhaseBlocking(ctx context.Context, br *bufio.Reader, acc *Accumulator, deadEnd time.Time) (eof bool, err error) {
	for {
		select {
		case <-ctx.Done():
			return false, context.Canceled
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
		if ctx.Err() != nil {
			return false, context.Canceled
		}
	}
}

func readPhaseWithReadDeadline(ctx context.Context, br *bufio.Reader, fin *os.File, acc *Accumulator, deadEnd time.Time) (eof bool, err error) {
	for {
		select {
		case <-ctx.Done():
			return false, context.Canceled
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
		select {
		case <-ctx.Done():
			return false, context.Canceled
		default:
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
		if ctx.Err() != nil {
			return false, context.Canceled
		}
	}
}

func readPhaseFromPump(ctx context.Context, acc *Accumulator, window time.Duration, lines <-chan []byte, errs <-chan error, startOnNextMsg bool) (eof bool, err error) {
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
			return false, context.Canceled
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

func (g *Guesser) emitSchema(ctx context.Context, out io.Writer, acc *Accumulator) error {
	doc, err := buildSchema(ctx, acc, g.cfg.variantThreshold, time.Now())
	if err != nil {
		return err
	}
	if g.cfg.omitVendorExt {
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
