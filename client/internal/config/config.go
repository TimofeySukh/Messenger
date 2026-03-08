package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Config struct {
	Server   string
	Username string
	Aliases  map[string]string
}

func Default() Config {
	return Config{
		Server:   "127.0.0.1:8080",
		Username: "",
		Aliases:  map[string]string{},
	}
}

func baseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".messenger"), nil
}

func Path() (string, error) {
	dir, err := baseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

func Load() (Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	defer f.Close()

	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}

		k, v, ok := splitKV(line)
		if !ok {
			continue
		}
		value := unquote(v)

		switch section {
		case "aliases":
			if value != "" {
				cfg.Aliases[k] = value
			}
		default:
			switch k {
			case "server":
				cfg.Server = value
			case "username":
				cfg.Username = value
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return cfg, scanErr
	}

	return cfg, nil
}

func Save(cfg Config) error {
	dir, err := baseDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	path := filepath.Join(dir, "config.toml")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "server = %q\n", cfg.Server); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "username = %q\n\n", cfg.Username); err != nil {
		return err
	}
	if _, err := f.WriteString("[aliases]\n"); err != nil {
		return err
	}

	keys := make([]string, 0, len(cfg.Aliases))
	for k := range cfg.Aliases {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if _, err := fmt.Fprintf(f, "%s = %q\n", k, cfg.Aliases[k]); err != nil {
			return err
		}
	}

	return nil
}

func splitKV(line string) (string, string, bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	k := strings.TrimSpace(parts[0])
	v := strings.TrimSpace(parts[1])
	if k == "" {
		return "", "", false
	}
	return k, v, true
}

func unquote(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, `"`)
	return v
}
