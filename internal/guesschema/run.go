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

// Run reads JSONL from in, writes one JSON Schema 2020-12 document to out,
// and returns ctx.Err() when ctx is canceled.
func (g *Guesser) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	acc := NewAccumulator()
	var (
		eof bool
		err error
	)
	if _, ok := probeReadDeadlineSupport(in); ok {
		eof, err = readPhase(ctx, in, acc, g.cfg.readWindow)
	} else {
		pump := startLinePump(ctx, in)
		eof, err = readPhaseFromPump(ctx, acc, g.cfg.readWindow, pump.lines, pump.errs, g.cfg.startOnNextMsg)
	}
	if err != nil {
		return err
	}
	if err := g.emitSchema(out, acc); err != nil {
		return err
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

func readPhase(ctx context.Context, in io.Reader, acc *Accumulator, window time.Duration) (eof bool, err error) {
	deadEnd := time.Now().Add(window)
	br := bufio.NewReaderSize(in, 64*1024)
	fin, isFile := in.(*os.File)

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

func readPhaseWithReadDeadline(ctx context.Context, br *bufio.Reader, fin *os.File, acc *Accumulator, deadEnd time.Time) (eof bool, err error) {
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

func readPhaseWithoutReadDeadline(ctx context.Context, br *bufio.Reader, acc *Accumulator, deadEnd time.Time) (eof bool, err error) {
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

func (g *Guesser) emitSchema(out io.Writer, acc *Accumulator) error {
	doc := BuildSchema(acc, g.cfg.variantThreshold, time.Now())
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
