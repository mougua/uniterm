//go:build windows
// +build windows

package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindGitBashInPath(t *testing.T) {
	root := t.TempDir()
	sys32 := filepath.Join(root, "Windows", "System32")
	gitBin := filepath.Join(root, "Git", "bin")
	empty := filepath.Join(root, "empty")
	for _, d := range []string{sys32, gitBin, empty} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(sys32, "bash.exe"), []byte("wsl"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitBin, "bash.exe"), []byte("git"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("skips WSL shim and finds Git Bash", func(t *testing.T) {
		// System32 sorts before Git's bin dir, so exec.LookPath would resolve
		// to the WSL shim — FindGitBashInPath must skip it.
		t.Setenv("PATH", sys32+";"+gitBin)
		got, ok := FindGitBashInPath()
		want := filepath.Join(gitBin, "bash.exe")
		if !ok || got != want {
			t.Fatalf("FindGitBashInPath() = %q, %v; want %q, true", got, ok, want)
		}
	})

	t.Run("only WSL shim on PATH", func(t *testing.T) {
		t.Setenv("PATH", sys32)
		if got, ok := FindGitBashInPath(); ok {
			t.Fatalf("FindGitBashInPath() = %q, true; want no Git Bash when only the WSL shim exists", got)
		}
	})

	t.Run("no bash.exe on PATH", func(t *testing.T) {
		t.Setenv("PATH", empty)
		if got, ok := FindGitBashInPath(); ok {
			t.Fatalf("FindGitBashInPath() = %q, true; want no result", got)
		}
	})

	t.Run("Git Bash before WSL shim", func(t *testing.T) {
		t.Setenv("PATH", gitBin+";"+sys32)
		got, ok := FindGitBashInPath()
		want := filepath.Join(gitBin, "bash.exe")
		if !ok || got != want {
			t.Fatalf("FindGitBashInPath() = %q, %v; want %q, true", got, ok, want)
		}
	})

	t.Run("forward-slash PATH entry", func(t *testing.T) {
		// PATH entries may use forward slashes; the shim must still be skipped.
		t.Setenv("PATH", strings.ReplaceAll(sys32, `\`, "/")+";"+gitBin)
		got, ok := FindGitBashInPath()
		want := filepath.Join(gitBin, "bash.exe")
		if !ok || got != want {
			t.Fatalf("FindGitBashInPath() = %q, %v; want %q, true", got, ok, want)
		}
	})

	t.Run("dir name merely contains windows", func(t *testing.T) {
		// A custom dir whose name contains "windows" but is not a system dir
		// must not be skipped.
		custom := filepath.Join(root, "windows-tools", "Git", "bin")
		if err := os.MkdirAll(custom, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(custom, "bash.exe"), []byte("git"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", sys32+";"+custom)
		got, ok := FindGitBashInPath()
		want := filepath.Join(custom, "bash.exe")
		if !ok || got != want {
			t.Fatalf("FindGitBashInPath() = %q, %v; want %q, true", got, ok, want)
		}
	})

	t.Run("WindowsApps App Execution Alias shim", func(t *testing.T) {
		// WindowsApps holds zero-byte App Execution Alias stubs (bash.exe)
		// that launch WSL; they sort before Git's bin dir on many machines
		// and must be skipped like System32.
		winApps := filepath.Join(root, "Users", "me", "AppData", "Local", "Microsoft", "WindowsApps")
		if err := os.MkdirAll(winApps, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(winApps, "bash.exe"), []byte("wsl-alias"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", winApps+";"+gitBin)
		got, ok := FindGitBashInPath()
		want := filepath.Join(gitBin, "bash.exe")
		if !ok || got != want {
			t.Fatalf("FindGitBashInPath() = %q, %v; want %q, true", got, ok, want)
		}
	})
}

func TestIsWSLBashPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{`C:\Windows\System32\bash.exe`, true},
		{`c:\windows\system32\bash.exe`, true},
		{`C:\Users\me\AppData\Local\Microsoft\WindowsApps\bash.exe`, true},
		{`C:\Users\me\AppData\Local\Microsoft\WindowsApps\bash.exe`, true},
		{`D:\wsl\bash.exe`, true},
		{`wsl://Ubuntu`, true},
		{`D:\Program Files\Git\bin\bash.exe`, false},
		{`C:\Program Files\Git\bin\bash.exe`, false},
		{`C:\Program Files\Git\usr\bin\bash.exe`, false},
	}
	for _, c := range cases {
		if got := isWSLBashPath(strings.ToLower(c.path)); got != c.want {
			t.Errorf("isWSLBashPath(%q) = %v; want %v", c.path, got, c.want)
		}
	}
}
