package utils

import "strings"

// LineWriter turns a byte stream into whole lines handed to emit.
//
// It splits on \r as well as on \n, because the progress of git is a sideband which redraws itself
// in place : "Counting objects:  43%\rCounting objects:  57%\r…". Feeding that raw to a line based
// output would wreck it.
//
// It is not safe for concurrent use : one clone at a time writes to it.
type LineWriter struct {
	buf  []byte
	emit func(string)
}

// NewLineWriter returns a writer handing each line it reads to emit
func NewLineWriter(emit func(string)) *LineWriter {
	return &LineWriter{emit: emit}
}

// Write always reports every byte written : a writer refusing bytes would abort the clone, and a
// line of progress is not worth that.
func (w *LineWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)

	for {
		at := -1
		for i, b := range w.buf {
			if b == '\n' || b == '\r' {
				at = i
				break
			}
		}
		if at < 0 {
			return len(p), nil
		}
		w.line(string(w.buf[:at]))
		w.buf = w.buf[at+1:]
	}
}

// Close hands over what was written without a line ending, as the last line of a clone often is
func (w *LineWriter) Close() error {
	w.line(string(w.buf))
	w.buf = nil
	return nil
}

// line hands a line over, unless there is nothing left of it once trimmed : splitting on both \r
// and \n leaves an empty segment on every "\r\n"
func (w *LineWriter) line(s string) {
	if s = strings.TrimSpace(s); s != "" && w.emit != nil {
		w.emit(s)
	}
}
