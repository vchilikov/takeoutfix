package exiftool

import (
	"bufio"
	"strings"
	"testing"
)

func TestHasErrorLine(t *testing.T) {
	if !hasErrorLine("Warning: x\nError: boom\n") {
		t.Fatalf("expected error line detection")
	}
	if hasErrorLine("1 image files updated\n") {
		t.Fatalf("did not expect error line")
	}
}

func TestHasErrorLine_CaseInsensitive(t *testing.T) {
	if !hasErrorLine("error: boom\n") {
		t.Fatalf("expected lowercase error detection")
	}
	if !hasErrorLine("ERROR: BOOM\n") {
		t.Fatalf("expected uppercase error detection")
	}
}

func TestHasErrorLine_CRLF(t *testing.T) {
	if !hasErrorLine("Warning: x\r\nError: boom\r\n") {
		t.Fatalf("expected error detection with CRLF")
	}
}

func TestFirstErrorLine(t *testing.T) {
	got := firstErrorLine("Warning: x\nError: boom\n")
	if got != "Error: boom" {
		t.Fatalf("firstErrorLine mismatch: got %q", got)
	}
}

func TestFirstErrorLine_NoErrors(t *testing.T) {
	got := firstErrorLine("all good\nno errors here\n")
	if got != "unknown error" {
		t.Fatalf("expected 'unknown error', got %q", got)
	}
}

func TestReadUntilReady(t *testing.T) {
	input := "line1\nline2\n{ready}\n"
	s := &Session{
		stdout: bufio.NewReader(strings.NewReader(input)),
	}

	output, err := s.readUntilReady()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "line1\nline2\n" {
		t.Fatalf("output mismatch: got %q", output)
	}
}

func TestReadUntilReady_EOF(t *testing.T) {
	input := "line1\nline2\n"
	s := &Session{
		stdout: bufio.NewReader(strings.NewReader(input)),
	}

	_, err := s.readUntilReady()
	if err == nil {
		t.Fatalf("expected error on EOF before {ready}")
	}
}

func TestReadUntilReady_ReadyWithNumber(t *testing.T) {
	input := "ok\n{ready42}\n"
	s := &Session{
		stdout: bufio.NewReader(strings.NewReader(input)),
	}

	output, err := s.readUntilReady()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "ok\n" {
		t.Fatalf("output mismatch: got %q", output)
	}
}

func TestRun_MarksClosedOnReadError(t *testing.T) {
	// Create a session with a reader that will EOF (no {ready} marker)
	input := "partial output\n"
	s := &Session{
		stdout: bufio.NewReader(strings.NewReader(input)),
		stdin:  nopWriteCloser{&strings.Builder{}},
	}

	_, err := s.Run([]string{"-ver"})
	if err == nil {
		t.Fatalf("expected error from Run")
	}
	if !s.closed {
		t.Fatalf("session should be marked closed after read error")
	}

	// Verify subsequent Run calls return "session is closed"
	_, err = s.Run([]string{"-ver"})
	if err == nil || err.Error() != "exiftool session is closed" {
		t.Fatalf("expected 'exiftool session is closed', got: %v", err)
	}
}

// nopWriteCloser wraps a strings.Builder to satisfy io.WriteCloser.
type nopWriteCloser struct {
	*strings.Builder
}

func (nopWriteCloser) Close() error { return nil }
