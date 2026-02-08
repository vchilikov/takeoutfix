package preflight

import (
	"fmt"
	"io"
	"os/exec"
	"os/user"
	"runtime"
	"strings"

	"github.com/vchilikov/takeout-fix/internal/exifcmd"
)

type Dependency struct {
	Name       string
	InstallCmd []string
}

var (
	currentGOOS     = runtime.GOOS
	lookPathFn      = exec.LookPath
	resolveExiftool = exifcmd.Resolve
	isRootUserFn    = currentUserIsRoot
	runCommandFn    = func(cmd *exec.Cmd) error { return cmd.Run() }
)

func CheckDependencies() []Dependency {
	var missing []Dependency

	if _, err := resolveExiftool(); err != nil {
		missing = append(missing, Dependency{Name: "exiftool", InstallCmd: installCommand("exiftool")})
	}

	return missing
}

func installCommand(name string) []string {
	switch currentGOOS {
	case "darwin":
		if commandAvailable("brew") {
			return []string{"brew", "install", name}
		}
	case "linux":
		return linuxInstallCommand(name)
	case "windows":
		return windowsInstallCommand(name)
	}
	return nil
}

func linuxInstallCommand(name string) []string {
	var base []string

	switch {
	case commandAvailable("apt-get"):
		base = []string{"apt-get", "install", "-y", linuxPackageName(name, "apt-get")}
	case commandAvailable("dnf"):
		base = []string{"dnf", "install", "-y", linuxPackageName(name, "dnf")}
	case commandAvailable("pacman"):
		base = []string{"pacman", "-S", "--noconfirm", linuxPackageName(name, "pacman")}
	default:
		return nil
	}

	if isRootUserFn() {
		return base
	}
	if !commandAvailable("sudo") {
		return nil
	}

	return append([]string{"sudo"}, base...)
}

func windowsInstallCommand(name string) []string {
	packageID := windowsPackageID(name)
	if packageID == "" {
		return nil
	}
	if !commandAvailable("winget") {
		return nil
	}

	return []string{
		"winget",
		"install",
		"--id", packageID,
		"--exact",
		"--accept-package-agreements",
		"--accept-source-agreements",
	}
}

func linuxPackageName(name string, manager string) string {
	if name != "exiftool" {
		return name
	}

	switch manager {
	case "apt-get":
		return "libimage-exiftool-perl"
	case "dnf":
		return "perl-Image-ExifTool"
	case "pacman":
		return "perl-image-exiftool"
	default:
		return name
	}
}

func windowsPackageID(name string) string {
	if name == "exiftool" {
		return "OliverBetz.ExifTool"
	}
	return ""
}

func commandAvailable(name string) bool {
	_, err := lookPathFn(name)
	return err == nil
}

func currentUserIsRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		return false
	}
	return currentUser.Uid == "0"
}

func InstallDependencies(deps []Dependency, in io.Reader, out io.Writer) error {
	for _, dep := range deps {
		if len(dep.InstallCmd) == 0 {
			return fmt.Errorf("no install command for %s", dep.Name)
		}
		if _, err := fmt.Fprintf(out, "$ %s\n", strings.Join(dep.InstallCmd, " ")); err != nil {
			return fmt.Errorf("write install command output: %w", err)
		}
		cmd := exec.Command(dep.InstallCmd[0], dep.InstallCmd[1:]...)
		cmd.Stdin = in
		cmd.Stdout = out
		cmd.Stderr = out
		if err := runCommandFn(cmd); err != nil {
			return fmt.Errorf("install %s: %w", dep.Name, err)
		}
	}
	return nil
}
