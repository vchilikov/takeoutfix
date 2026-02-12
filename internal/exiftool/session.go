package exiftool

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vchilikov/takeout-fix/internal/exifcmd"
)

const defaultReadReadyTimeout = 60 * time.Second

var errReadTimeout = errors.New("timeout waiting for exiftool ready marker")

const statusMarkerPrefix = "__TAKEOUTFIX_STATUS__:"

type Session struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	mu           sync.Mutex
	closed       bool
	readyTimeout time.Duration
}

func Start() (*Session, error) {
	bin, err := exifcmd.Resolve()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(bin, "-stay_open", "True", "-@", "-")
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	// Merge stderr into the same pipe so we can detect error messages from exiftool.
	cmd.Stderr = cmd.Stdout

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start exiftool: %w", err)
	}

	return &Session{
		cmd:          cmd,
		stdin:        stdinPipe,
		stdout:       bufio.NewReader(stdoutPipe),
		readyTimeout: defaultReadReadyTimeout,
	}, nil
}

func (s *Session) Run(args []string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return "", errors.New("exiftool session is closed")
	}
	if err := validateArgs(args); err != nil {
		return "", err
	}

	for _, arg := range args {
		if _, err := io.WriteString(s.stdin, arg+"\n"); err != nil {
			return "", fmt.Errorf("write command: %w", err)
		}
	}
	if _, err := io.WriteString(s.stdin, "-echo3\n"); err != nil {
		return "", fmt.Errorf("write status option: %w", err)
	}
	if _, err := io.WriteString(s.stdin, statusMarkerPrefix+"${status}\n"); err != nil {
		return "", fmt.Errorf("write status marker: %w", err)
	}
	if _, err := io.WriteString(s.stdin, "-execute\n"); err != nil {
		return "", fmt.Errorf("write execute marker: %w", err)
	}

	result, err := s.readUntilReady()
	if err != nil {
		if errors.Is(err, errReadTimeout) {
			s.terminateLocked()
		}
		s.closed = true
		return result.output, err
	}

	output := result.output
	if result.statusFound {
		if result.status != 0 {
			return output, buildStatusError(result.status, output)
		}
		return output, nil
	}

	if hasErrorLine(output) {
		return output, buildStatusError(1, output)
	}
	return output, nil
}

func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	if _, err := io.WriteString(s.stdin, "-stay_open\nFalse\n"); err != nil {
		_ = s.stdin.Close()
		_ = s.cmd.Wait()
		return fmt.Errorf("write close marker: %w", err)
	}
	if err := s.stdin.Close(); err != nil {
		_ = s.cmd.Wait()
		return fmt.Errorf("close stdin: %w", err)
	}
	if err := s.cmd.Wait(); err != nil {
		return fmt.Errorf("wait exiftool: %w", err)
	}
	return nil
}

type readUntilReadyResult struct {
	output      string
	statusFound bool
	status      int
}

func (s *Session) readUntilReady() (readUntilReadyResult, error) {
	result := readUntilReadyResult{}
	var out strings.Builder
	timeout := s.readyTimeout
	if timeout <= 0 {
		timeout = defaultReadReadyTimeout
	}

	for {
		line, err := s.readLineWithTimeout(timeout)
		if err != nil {
			result.output = out.String()
			return result, fmt.Errorf("read output: %w", err)
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{ready") {
			break
		}

		if strings.HasPrefix(trimmed, statusMarkerPrefix) {
			statusValue := strings.TrimSpace(strings.TrimPrefix(trimmed, statusMarkerPrefix))
			if status, convErr := strconv.Atoi(statusValue); convErr == nil {
				result.statusFound = true
				result.status = status
				continue
			}
		}

		out.WriteString(line)
	}
	result.output = out.String()
	return result, nil
}

func (s *Session) readLineWithTimeout(timeout time.Duration) (string, error) {
	type readResult struct {
		line string
		err  error
	}

	ch := make(chan readResult, 1)
	go func() {
		line, err := s.stdout.ReadString('\n')
		ch <- readResult{line: line, err: err}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result := <-ch:
		return result.line, result.err
	case <-timer.C:
		return "", errReadTimeout
	}
}

func (s *Session) terminateLocked() {
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	if s.cmd != nil {
		_ = s.cmd.Wait()
	}
}

func buildStatusError(status int, output string) error {
	if msg := firstErrorLine(output); msg != "unknown error" {
		return fmt.Errorf("exiftool command failed (status %d): %s", status, msg)
	}
	return fmt.Errorf("exiftool command failed with status %d", status)
}

func hasErrorLine(output string) bool {
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "error:") {
			return true
		}
	}
	return false
}

func firstErrorLine(output string) string {
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "error:") {
			return trimmed
		}
	}
	return "unknown error"
}

func validateArgs(args []string) error {
	for _, arg := range args {
		if strings.ContainsAny(arg, "\r\n") || strings.IndexByte(arg, 0) >= 0 {
			return fmt.Errorf("invalid exiftool argument: contains newline or null byte")
		}
	}
	return nil
}
