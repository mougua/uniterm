package session

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOpenSSHAliases(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	aliases, err := OpenSSHAliases()
	if err != nil || len(aliases) != 0 {
		t.Fatalf("missing config: got %v, %v", aliases, err)
	}

	dir := filepath.Join(home, ".ssh")
	includeDir := filepath.Join(dir, "config d")
	if err := os.MkdirAll(includeDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config"), []byte(`
Host * !ignored staging STAGING
    User deploy
Include "config d/*.conf" # look for additional hosts
host = "web-1" staging-* # no aliases in comments
`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "hosts.conf"), []byte(`
Include config
Host build
Match user deploy
    Port 2222
`), 0600); err != nil {
		t.Fatal(err)
	}

	aliases, err = OpenSSHAliases()
	want := []string{"build", "staging", "web-1"}
	if err != nil || !reflect.DeepEqual(aliases, want) {
		t.Fatalf("got %v, %v; want %v", aliases, err, want)
	}
}

func TestOpenSSHCommandRejectsOptions(t *testing.T) {
	for _, host := range []string{"-F", "host -p 2200", "host;command", ""} {
		if _, err := openSSHCommand(host); err == nil {
			t.Errorf("accepted invalid alias %q", host)
		}
	}
}
