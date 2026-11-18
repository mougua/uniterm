package session

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var sshConfigDirective = regexp.MustCompile(`(?i)^\s*(host|include)(?:\s*=\s*|\s+)(.*)$`)
var sshConfigWord = regexp.MustCompile(`#.*|"[^"]*"|[^\s"#]+`)
var sshAlias = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]*$`)

// OpenSSHAliases discovers literal host names. OpenSSH itself resolves the
// effective configuration when the user connects.
func OpenSSHAliases() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".ssh")
	aliases := map[string]string{}
	err = scanSSHConfig(filepath.Join(dir, "config"), dir, aliases, map[string]bool{}, 0)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	result := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		result = append(result, alias)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})
	return result, nil
}

func scanSSHConfig(path, dir string, aliases map[string]string, visited map[string]bool, depth int) error {
	if depth > 16 || visited[path] {
		return nil
	}
	visited[path] = true
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("SSH config is not a regular file: %s", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := sshConfigDirective.FindStringSubmatch(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == nil {
			continue
		}
		for _, token := range sshConfigWord.FindAllString(line[2], -1) {
			if strings.HasPrefix(token, "#") {
				break
			}
			word := strings.Trim(token, `"`)
			if strings.EqualFold(line[1], "host") {
				if sshAlias.MatchString(word) {
					key := strings.ToLower(word)
					if _, exists := aliases[key]; !exists {
						aliases[key] = word
					}
				}
				continue
			}
			if strings.HasPrefix(word, "~/") {
				word = filepath.Join(filepath.Dir(dir), word[2:])
			} else if !filepath.IsAbs(word) {
				word = filepath.Join(dir, word)
			}
			matches, err := filepath.Glob(word)
			if err != nil {
				return err
			}
			for _, match := range matches {
				if err := scanSSHConfig(match, dir, aliases, visited, depth+1); err != nil && !errors.Is(err, os.ErrNotExist) {
					return err
				}
			}
		}
	}
	return scanner.Err()
}

func openSSHCommand(alias string) (*exec.Cmd, error) {
	if !sshAlias.MatchString(alias) {
		return nil, fmt.Errorf("invalid OpenSSH host alias: %q", alias)
	}
	path, err := exec.LookPath("ssh")
	if err != nil {
		return nil, fmt.Errorf("OpenSSH client (ssh) is required: %w", err)
	}
	return exec.Command(path, "--", alias), nil
}
