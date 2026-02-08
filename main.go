package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vchilikov/takeout-fix/internal/wizard"
)

func main() {
	workDir, err := resolveWorkDir(os.Args[1:], os.Getwd, os.Stat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid arguments: %v\n", err)
		fmt.Fprintln(os.Stderr, "usage: takeoutfix [--workdir /path/to/folder]")
		os.Exit(wizard.ExitRuntimeFail)
	}

	code := wizard.Run(workDir, os.Stdin, os.Stdout)
	os.Exit(code)
}

func resolveWorkDir(
	args []string,
	getwd func() (string, error),
	statFn func(string) (os.FileInfo, error),
) (string, error) {
	fs := flag.NewFlagSet("takeoutfix", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	workdir := fs.String("workdir", "", "working directory")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() > 0 {
		return "", fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}

	resolved := strings.TrimSpace(*workdir)
	if resolved == "" {
		cwd, err := getwd()
		if err != nil {
			return "", fmt.Errorf("get current working directory: %w", err)
		}
		resolved = cwd
	}

	info, err := statFn(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("workdir %q does not exist", resolved)
		}
		return "", fmt.Errorf("stat workdir %q: %w", resolved, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workdir %q is not a directory", resolved)
	}

	return resolved, nil
}
