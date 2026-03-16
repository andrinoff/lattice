package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for Lattice.
type Config struct {
	// Columns is the number of columns in the grid layout (default: 2).
	Columns int `yaml:"columns"`

	// Modules lists which modules to display, in order.
	// If empty, a sensible default set is used.
	Modules []ModuleConfig `yaml:"modules"`
}

// ModuleConfig configures a single module instance.
type ModuleConfig struct {
	Type   string            `yaml:"type"`
	Config map[string]string `yaml:"config,omitempty"`
}

// DefaultConfig returns the default configuration when no config file exists.
func DefaultConfig() Config {
	return Config{
		Columns: 2,
		Modules: []ModuleConfig{
			{Type: "greeting"},
			{Type: "clock"},
			{Type: "system"},
			{Type: "github"},
			{Type: "weather"},
			{Type: "uptime"},
		},
	}
}

// LoadConfig reads the config from ~/.config/lattice/config.yaml.
// Falls back to DefaultConfig if the file doesn't exist.
func LoadConfig() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultConfig()
	}

	path := filepath.Join(home, ".config", "lattice", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig()
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig()
	}

	if cfg.Columns < 1 {
		cfg.Columns = 2
	}
	if len(cfg.Modules) == 0 {
		cfg.Modules = DefaultConfig().Modules
	}

	return cfg
}

// Get retrieves a config value, falling back to an environment variable,
// then to a default. This lets users put secrets in env vars instead of
// the config file.
func (mc ModuleConfig) Get(key, envVar, fallback string) string {
	if v, ok := mc.Config[key]; ok && v != "" {
		return v
	}
	if envVar != "" {
		if v := os.Getenv(envVar); v != "" {
			return v
		}
	}
	return fallback
}
