package exifcmd

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestCandidates(t *testing.T) {
	tests := []struct {
		name string
		goos string
		want []string
	}{
		{
			name: "windows",
			goos: "windows",
			want: []string{"exiftool", "exiftool.exe", "exiftool(-k).exe"},
		},
		{
			name: "linux",
			goos: "linux",
			want: []string{"exiftool"},
		},
		{
			name: "darwin",
			goos: "darwin",
			want: []string{"exiftool"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := Candidates(tc.goos)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Candidates() mismatch:\nwant: %v\ngot:  %v", tc.want, got)
			}
		})
	}
}

func TestResolve_UsesFallbackCandidates(t *testing.T) {
	restore := stubResolverEnvironment("windows", map[string]string{
		"exiftool.exe": "C:\\Program Files\\ExifTool\\exiftool.exe",
	})
	defer restore()

	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	want := "C:\\Program Files\\ExifTool\\exiftool.exe"
	if got != want {
		t.Fatalf("Resolve() mismatch:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestResolve_ReturnsClearErrorWhenNotFound(t *testing.T) {
	restore := stubResolverEnvironment("windows", map[string]string{})
	defer restore()

	_, err := Resolve()
	if err == nil {
		t.Fatalf("expected Resolve() error")
	}
	msg := err.Error()
	for _, candidate := range []string{"exiftool", "exiftool.exe", "exiftool(-k).exe"} {
		if !strings.Contains(msg, candidate) {
			t.Fatalf("expected %q in error message, got: %q", candidate, msg)
		}
	}
}

func stubResolverEnvironment(goos string, available map[string]string) func() {
	origGOOS := currentGOOS
	origLookPathFn := lookPathFn

	currentGOOS = goos
	lookPathFn = func(file string) (string, error) {
		if path, ok := available[file]; ok {
			return path, nil
		}
		return "", exec.ErrNotFound
	}

	return func() {
		currentGOOS = origGOOS
		lookPathFn = origLookPathFn
	}
}
