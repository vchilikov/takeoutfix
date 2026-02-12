package exiftool

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
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

	result, err := s.readUntilReady()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.output != "line1\nline2\n" {
		t.Fatalf("output mismatch: got %q", result.output)
	}
	if result.statusFound {
		t.Fatalf("did not expect status marker in output: %+v", result)
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

	result, err := s.readUntilReady()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.output != "ok\n" {
		t.Fatalf("output mismatch: got %q", result.output)
	}
	if result.statusFound {
		t.Fatalf("did not expect status marker in output: %+v", result)
	}
}

func TestReadUntilReady_ParsesStatusMarker(t *testing.T) {
	input := "ok\n" + statusMarkerPrefix + "0\n{ready}\n"
	s := &Session{
		stdout: bufio.NewReader(strings.NewReader(input)),
	}

	result, err := s.readUntilReady()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.output != "ok\n" {
		t.Fatalf("output mismatch: got %q", result.output)
	}
	if !result.statusFound || result.status != 0 {
		t.Fatalf("expected parsed status=0, got %+v", result)
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

func TestRun_MarksClosedOnReadTimeout(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	var in strings.Builder
	s := &Session{
		stdout:       bufio.NewReader(blockingReader{release: release}),
		stdin:        nopWriteCloser{&in},
		readyTimeout: 10 * time.Millisecond,
	}

	_, err := s.Run([]string{"-ver"})
	if err == nil {
		t.Fatalf("expected timeout error from Run")
	}
	if !errors.Is(err, errReadTimeout) {
		t.Fatalf("expected read timeout error, got: %v", err)
	}
	if !s.closed {
		t.Fatalf("session should be marked closed after timeout")
	}

	_, err = s.Run([]string{"-ver"})
	if err == nil || err.Error() != "exiftool session is closed" {
		t.Fatalf("expected 'exiftool session is closed', got: %v", err)
	}
}

func TestRun_RejectsUnsafeArguments(t *testing.T) {
	tests := map[string]string{
		"newline":         "photo\n.jpg",
		"carriage-return": "photo\r.jpg",
		"null-byte":       "photo\x00.jpg",
	}

	for name, arg := range tests {
		t.Run(name, func(t *testing.T) {
			var in strings.Builder
			s := &Session{
				stdout: bufio.NewReader(strings.NewReader("{ready}\n")),
				stdin:  nopWriteCloser{&in},
			}

			_, err := s.Run([]string{arg})
			if err == nil {
				t.Fatalf("expected validation error for arg %q", arg)
			}
			if got := in.String(); got != "" {
				t.Fatalf("expected no stdin writes for invalid arg, got %q", got)
			}
		})
	}
}

func TestRun_WritesExecuteForValidArguments(t *testing.T) {
	var in strings.Builder
	s := &Session{
		stdout: bufio.NewReader(strings.NewReader(statusMarkerPrefix + "0\n{ready}\n")),
		stdin:  nopWriteCloser{&in},
	}

	if _, err := s.Run([]string{"-ver"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := in.String(); got != "-ver\n-echo3\n"+statusMarkerPrefix+"${status}\n-execute\n" {
		t.Fatalf("stdin mismatch: got %q", got)
	}
}

func TestRun_StatusZeroAllowsErrorLikeOutput(t *testing.T) {
	input := "Error: this is regular output\n" + statusMarkerPrefix + "0\n{ready}\n"
	s := &Session{
		stdout: bufio.NewReader(strings.NewReader(input)),
		stdin:  nopWriteCloser{&strings.Builder{}},
	}

	output, err := s.Run([]string{"-ver"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "Error: this is regular output\n" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRun_StatusNonZeroReturnsError(t *testing.T) {
	input := "Warning: x\nError: boom\n" + statusMarkerPrefix + "1\n{ready}\n"
	s := &Session{
		stdout: bufio.NewReader(strings.NewReader(input)),
		stdin:  nopWriteCloser{&strings.Builder{}},
	}

	output, err := s.Run([]string{"-ver"})
	if err == nil {
		t.Fatalf("expected status error")
	}
	if !strings.Contains(err.Error(), "status 1") {
		t.Fatalf("expected status in error, got %v", err)
	}
	if output != "Warning: x\nError: boom\n" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRun_FallbackToErrorLineWhenStatusMarkerMissing(t *testing.T) {
	input := "Warning: x\nError: boom\n{ready}\n"
	s := &Session{
		stdout: bufio.NewReader(strings.NewReader(input)),
		stdin:  nopWriteCloser{&strings.Builder{}},
	}

	_, err := s.Run([]string{"-ver"})
	if err == nil {
		t.Fatalf("expected fallback error when status marker missing")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "status 1") {
		t.Fatalf("expected synthesized status in fallback error, got %v", err)
	}
}

// nopWriteCloser wraps a strings.Builder to satisfy io.WriteCloser.
type nopWriteCloser struct {
	*strings.Builder
}

func (nopWriteCloser) Close() error { return nil }

type blockingReader struct {
	release <-chan struct{}
}

func (b blockingReader) Read(_ []byte) (int, error) {
	<-b.release
	return 0, io.EOF
}
