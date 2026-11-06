// Command configcheck validates the single runtime settings contract without
// initializing any Cloud, LINE or AI SDK or reading a secret value into logs.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/963sup/linebot-platform/internal/bootstrap"
)

const (
	documentPath = "docs/operations/configuration.md"
	examplePath  = ".env.example"
)

func checkExample(data []byte) error {
	known := make(map[string]bootstrap.SettingSpec)
	for _, spec := range bootstrap.SettingsContract() {
		if _, exists := known[spec.Key]; exists {
			return fmt.Errorf("duplicate settings contract key %s", spec.Key)
		}
		known[spec.Key] = spec
	}
	seen := make(map[string]bool)
	values := make(map[string]string, len(known))
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return errors.New("invalid .env.example assignment")
		}
		spec, exists := known[key]
		if !exists {
			return fmt.Errorf(".env.example contains unknown key %s", key)
		}
		if seen[key] {
			return fmt.Errorf(".env.example duplicates key %s", key)
		}
		seen[key] = true
		if spec.Secret && strings.TrimSpace(value) != "" {
			return fmt.Errorf(".env.example must not contain credential values for %s", key)
		}
		values[key] = strings.TrimSpace(value)
	}
	for _, key := range []string{
		bootstrap.KeyAppEnv, bootstrap.KeyPort, bootstrap.KeyEventMode,
		bootstrap.KeyLINETimeout, bootstrap.KeyLogLevel, bootstrap.KeyLogFormat,
	} {
		if !seen[key] {
			return fmt.Errorf(".env.example is missing public setting %s", key)
		}
	}
	// Validate the example against the same typed runtime parser as the
	// webhook. Key-only checks would accept invalid ports, log levels or
	// deployment profiles, then advertise a configuration that cannot start.
	cfg, err := bootstrap.LoadConfig(func(key string) string { return values[key] })
	if err != nil {
		return fmt.Errorf(".env.example runtime settings: %w", err)
	}
	if cfg.Environment != bootstrap.EnvLocal || cfg.EventMode != "sync" || cfg.LiveLINE() {
		return errors.New(".env.example must describe a credential-free local sync webhook")
	}
	return nil
}

func run(args []string, getenv func(string) string, output io.Writer) error {
	flags := flag.NewFlagSet("configcheck", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	check := flags.Bool("check", false, "verify generated docs and safe .env.example")
	write := flags.Bool("write-doc", false, "regenerate docs from Go contract")
	validate := flags.Bool("validate", false, "validate role configuration without network calls")
	role := flags.String("role", "webhook", "role to validate: webhook or worker")
	if err := flags.Parse(args); err != nil {
		return err
	}
	selected := 0
	for _, action := range []bool{*check, *write, *validate} {
		if action {
			selected++
		}
	}
	if selected != 1 || len(flags.Args()) != 0 {
		return errors.New("specify exactly one of --check, --write-doc, --validate")
	}
	if *validate {
		var selectedRole bootstrap.RuntimeRole
		var environment bootstrap.Environment
		var mode, port string
		var logging bootstrap.LogConfig
		switch *role {
		case "webhook":
			cfg, err := bootstrap.LoadConfig(getenv)
			if err != nil {
				return err
			}
			selectedRole, environment, mode, port, logging = cfg.Role, cfg.Environment, cfg.EventMode, cfg.Port, cfg.Log
		case "worker":
			cfg, err := bootstrap.LoadWorkerConfig(getenv)
			if err != nil {
				return err
			}
			selectedRole, environment, mode, port, logging = cfg.Role, cfg.Environment, cfg.EventMode, cfg.Port, cfg.Log
		case "miniapp":
			cfg, err := bootstrap.LoadMiniAppConfig(getenv)
			if err != nil {
				return err
			}
			selectedRole, environment, mode, port, logging = cfg.Role, cfg.Environment, "n/a", cfg.Port, cfg.Log
		default:
			return errors.New("--role must be webhook, worker or miniapp")
		}
		_, err := fmt.Fprintf(output, "valid role=%s profile=%s mode=%s port=%s log=%s/%s\n",
			selectedRole, environment, mode, port, logging.Level, logging.Format)
		return err
	}
	if *write {
		return os.WriteFile(documentPath, []byte(bootstrap.ConfigurationMarkdown()), 0o644)
	}
	actual, err := os.ReadFile(filepath.Clean(documentPath))
	if err != nil {
		return err
	}
	if !bytes.Equal(actual, []byte(bootstrap.ConfigurationMarkdown())) {
		return errors.New("configuration docs drift: run go run ./cmd/configcheck --write-doc")
	}
	example, err := os.ReadFile(filepath.Clean(examplePath))
	if err != nil {
		return err
	}
	if err := checkExample(example); err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, "configuration contract OK")
	return err
}

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout); err != nil {
		// Errors contain only well-known config names; never print env values.
		fmt.Fprintln(os.Stderr, "configcheck:", err)
		os.Exit(1)
	}
}
