package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is persisted as JSON under the user config dir.
type Config struct {
	Theme     string `json:"theme"`
	RefreshMs int    `json:"refresh_ms"`
	Iface     string `json:"iface"`
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "system-critters", "config.json")
}

func loadConfig() Config {
	c := Config{Theme: themes[0].Name, RefreshMs: 1000}
	if p := configPath(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			_ = json.Unmarshal(b, &c)
		}
	}
	if c.RefreshMs < 250 {
		c.RefreshMs = 250
	}
	if c.RefreshMs > 10000 {
		c.RefreshMs = 10000
	}
	return c
}

func (c Config) save() {
	p := configPath()
	if p == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	if b, err := json.MarshalIndent(c, "", "  "); err == nil {
		_ = os.WriteFile(p, b, 0o644)
	}
}
