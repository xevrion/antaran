package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ScanRoot     string        `toml:"scan_root"`
	ScanInterval time.Duration `toml:"scan_interval"`
	Git          GitConfig     `toml:"git"`
	Process      ProcessConfig `toml:"process"`
}

type GitConfig struct {
	MaxDepth      int      `toml:"max_depth"`
	FetchRemote   bool     `toml:"fetch_remote"`
	StaleAfterDays int     `toml:"stale_after_days"`
	Ignore        []string `toml:"ignore"`
}

type ProcessConfig struct {
	Watch   []string `toml:"watch"`
	Ignore  []uint32 `toml:"ignore_pids"`
}

func Default() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		ScanRoot:     filepath.Join(home, "Coding"),
		ScanInterval: 30 * time.Second,
		Git: GitConfig{
			MaxDepth:       3,
			FetchRemote:    false,
			StaleAfterDays: 14,
		},
		Process: ProcessConfig{
			Watch: []string{
				"node", "cargo", "vite", "next", "webpack",
				"python", "uvicorn", "deno", "bun", "go",
			},
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	_, err := toml.DecodeFile(path, cfg)
	return cfg, err
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "antaran", "antaran.toml")
}
