package preflight

import (
	"bytes"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestInstallCommand_DarwinWithBrew(t *testing.T) {
	restore := stubInstallEnvironment("darwin", map[string]bool{"brew": true}, false)
	defer restore()

	got := installCommand("exiftool")
	want := []string{"brew", "install", "exiftool"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("installCommand mismatch:\nwant: %v\ngot:  %v", want, got)
	}
}

func TestInstallCommand_LinuxAptGet(t *testing.T) {
	restore := stubInstallEnvironment("linux", map[string]bool{"apt-get": true, "sudo": true}, false)
	defer restore()

	got := installCommand("exiftool")
	want := []string{"sudo", "apt-get", "install", "-y", "libimage-exiftool-perl"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("installCommand mismatch:\nwant: %v\ngot:  %v", want, got)
	}
}

func TestInstallCommand_LinuxDnf(t *testing.T) {
	restore := stubInstallEnvironment("linux", map[string]bool{"dnf": true, "sudo": true}, false)
	defer restore()

	got := installCommand("exiftool")
	want := []string{"sudo", "dnf", "install", "-y", "perl-Image-ExifTool"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("installCommand mismatch:\nwant: %v\ngot:  %v", want, got)
	}
}

func TestInstallCommand_LinuxPacman(t *testing.T) {
	restore := stubInstallEnvironment("linux", map[string]bool{"pacman": true, "sudo": true}, false)
	defer restore()

	got := installCommand("exiftool")
	want := []string{"sudo", "pacman", "-S", "--noconfirm", "perl-image-exiftool"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("installCommand mismatch:\nwant: %v\ngot:  %v", want, got)
	}
}

func TestInstallCommand_LinuxNoSupportedManager(t *testing.T) {
	restore := stubInstallEnvironment("linux", map[string]bool{"sudo": true}, false)
	defer restore()

	if got := installCommand("exiftool"); got != nil {
		t.Fatalf("expected nil install command, got %v", got)
	}
}

func TestInstallCommand_LinuxWithoutSudoAndWithoutRoot(t *testing.T) {
	restore := stubInstallEnvironment("linux", map[string]bool{"apt-get": true}, false)
	defer restore()

	if got := installCommand("exiftool"); got != nil {
		t.Fatalf("expected nil install command without sudo and root, got %v", got)
	}
}

func TestInstallCommand_LinuxRootWithoutSudo(t *testing.T) {
	restore := stubInstallEnvironment("linux", map[string]bool{"apt-get": true}, true)
	defer restore()

	got := installCommand("exiftool")
	want := []string{"apt-get", "install", "-y", "libimage-exiftool-perl"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("installCommand mismatch:\nwant: %v\ngot:  %v", want, got)
	}
}

func TestInstallCommand_WindowsWinget(t *testing.T) {
	restore := stubInstallEnvironment("windows", map[string]bool{"winget": true}, false)
	defer restore()

	got := installCommand("exiftool")
	want := []string{
		"winget",
		"install",
		"--id", "OliverBetz.ExifTool",
		"--exact",
		"--accept-package-agreements",
		"--accept-source-agreements",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("installCommand mismatch:\nwant: %v\ngot:  %v", want, got)
	}
}

func TestInstallCommand_WindowsWithoutWinget(t *testing.T) {
	restore := stubInstallEnvironment("windows", map[string]bool{}, false)
	defer restore()

	if got := installCommand("exiftool"); got != nil {
		t.Fatalf("expected nil install command, got %v", got)
	}
}

func TestCheckDependencies_UsesResolver(t *testing.T) {
	origResolver := resolveExiftool
	defer func() {
		resolveExiftool = origResolver
	}()

	resolveExiftool = func() (string, error) {
		return "/usr/bin/exiftool", nil
	}

	got := CheckDependencies()
	if len(got) != 0 {
		t.Fatalf("expected no missing dependencies, got %#v", got)
	}
}

func TestCheckDependencies_MissingExiftoolOnWindowsIncludesWingetCommand(t *testing.T) {
	restore := stubInstallEnvironment("windows", map[string]bool{"winget": true}, false)
	defer restore()

	origResolver := resolveExiftool
	defer func() {
		resolveExiftool = origResolver
	}()

	resolveExiftool = func() (string, error) {
		return "", errors.New("not found")
	}

	got := CheckDependencies()
	if len(got) != 1 {
		t.Fatalf("expected one missing dependency, got %#v", got)
	}
	if got[0].Name != "exiftool" {
		t.Fatalf("unexpected dependency name: %q", got[0].Name)
	}
	wantCmd := []string{
		"winget",
		"install",
		"--id", "OliverBetz.ExifTool",
		"--exact",
		"--accept-package-agreements",
		"--accept-source-agreements",
	}
	if !reflect.DeepEqual(got[0].InstallCmd, wantCmd) {
		t.Fatalf("install command mismatch:\nwant: %v\ngot:  %v", wantCmd, got[0].InstallCmd)
	}
}

func TestInstallDependencies_PassesStdinToCommand(t *testing.T) {
	origRunCommandFn := runCommandFn
	defer func() {
		runCommandFn = origRunCommandFn
	}()

	input := strings.NewReader("password\n")
	var captured *exec.Cmd
	runCommandFn = func(cmd *exec.Cmd) error {
		captured = cmd
		return nil
	}

	var out bytes.Buffer
	err := InstallDependencies([]Dependency{
		{
			Name:       "exiftool",
			InstallCmd: []string{"sudo", "apt-get", "install", "-y", "libimage-exiftool-perl"},
		},
	}, input, &out)
	if err != nil {
		t.Fatalf("InstallDependencies error: %v", err)
	}
	if captured == nil {
		t.Fatalf("expected command to be executed")
	}
	if captured.Stdin != input {
		t.Fatalf("expected command stdin to use provided input reader")
	}
	if captured.Stdout != &out || captured.Stderr != &out {
		t.Fatalf("expected command stdout/stderr to point to provided writer")
	}
	if !strings.Contains(out.String(), "$ sudo apt-get install -y libimage-exiftool-perl") {
		t.Fatalf("expected install command line in output, got: %q", out.String())
	}
}

func stubInstallEnvironment(goos string, available map[string]bool, isRoot bool) func() {
	origGOOS := currentGOOS
	origLookPathFn := lookPathFn
	origIsRootUserFn := isRootUserFn

	currentGOOS = goos
	lookPathFn = func(file string) (string, error) {
		if available[file] {
			return "/usr/bin/" + file, nil
		}
		return "", exec.ErrNotFound
	}
	isRootUserFn = func() bool {
		return isRoot
	}

	return func() {
		currentGOOS = origGOOS
		lookPathFn = origLookPathFn
		isRootUserFn = origIsRootUserFn
	}
}
