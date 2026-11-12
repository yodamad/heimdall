package utils

import (
	"strings"
	"testing"
)

// collect returns a writer and the lines it emitted
func collect() (*LineWriter, *[]string) {
	var lines []string
	return NewLineWriter(func(s string) { lines = append(lines, s) }), &lines
}

func TestWriteSplitsOnBothEndings(t *testing.T) {
	w, lines := collect()
	if n, err := w.Write([]byte("first\nsecond\rthird\r\n")); n != 20 || err != nil {
		t.Fatalf("Write reported %d, %v", n, err)
	}

	if want := []string{"first", "second", "third"}; strings.Join(*lines, "|") != strings.Join(want, "|") {
		t.Errorf("lines are %v, expected %v", *lines, want)
	}
}

func TestWriteKeepsAPartialLineForLater(t *testing.T) {
	w, lines := collect()
	w.Write([]byte("Counting objects:  4"))
	if len(*lines) != 0 {
		t.Fatalf("expected nothing emitted yet, got %v", *lines)
	}

	w.Write([]byte("3%\rCounting objects: 100%"))
	if len(*lines) != 1 || (*lines)[0] != "Counting objects:  43%" {
		t.Fatalf("unexpected lines : %v", *lines)
	}

	// What is written without an ending comes out on Close, as the last line of a clone does
	if err := w.Close(); err != nil {
		t.Fatalf("Close returned %v", err)
	}
	if len(*lines) != 2 || (*lines)[1] != "Counting objects: 100%" {
		t.Errorf("unexpected lines after Close : %v", *lines)
	}
}

func TestEmptySegmentsAreDropped(t *testing.T) {
	w, lines := collect()
	w.Write([]byte("\r\n\n  \r\none\n"))
	w.Close()

	if len(*lines) != 1 || (*lines)[0] != "one" {
		t.Errorf("expected only the one line, got %v", *lines)
	}
}

func TestCloseWithoutAnythingWritten(t *testing.T) {
	w, lines := collect()
	if err := w.Close(); err != nil {
		t.Fatalf("Close returned %v", err)
	}
	if len(*lines) != 0 {
		t.Errorf("expected no line, got %v", *lines)
	}
}
