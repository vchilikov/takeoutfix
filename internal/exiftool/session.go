package exiftool

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/vchilikov/takeout-fix/internal/exifcmd"
)

type Session struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	mu     sync.Mutex
	closed bool
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
		cmd:    cmd,
		stdin:  stdinPipe,
		stdout: bufio.NewReader(stdoutPipe),
	}, nil
}

func (s *Session) Run(args []string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return "", errors.New("exiftool session is closed")
	}

	for _, arg := range args {
		if _, err := io.WriteString(s.stdin, arg+"\n"); err != nil {
			return "", fmt.Errorf("write command: %w", err)
		}
	}
	if _, err := io.WriteString(s.stdin, "-execute\n"); err != nil {
		return "", fmt.Errorf("write execute marker: %w", err)
	}

	output, err := s.readUntilReady()
	if err != nil {
		s.closed = true
		return output, err
	}
	if hasErrorLine(output) {
		return output, fmt.Errorf("exiftool command failed: %s", firstErrorLine(output))
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

func (s *Session) readUntilReady() (string, error) {
	var out strings.Builder
	for {
		line, err := s.stdout.ReadString('\n')
		if err != nil {
			return out.String(), fmt.Errorf("read output: %w", err)
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{ready") {
			break
		}
		out.WriteString(line)
	}
	return out.String(), nil
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
