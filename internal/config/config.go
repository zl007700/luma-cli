package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	envConfigDir = "LUMA_CONFIG_DIR"
	envCardKey   = "LUMA_CARD_KEY"
)

// Config is the local CLI configuration.
type Config struct {
	CardKey string `json:"card_key"`
}

// Dir returns the configuration directory.
func Dir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv(envConfigDir)); dir != "" {
		return filepath.Abs(dir)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".luma"), nil
}

// Path returns the configuration file path.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads configuration, with environment variables taking precedence.
func Load() (*Config, error) {
	if key := strings.TrimSpace(os.Getenv(envCardKey)); key != "" {
		return &Config{CardKey: key}, nil
	}

	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.CardKey = strings.TrimSpace(cfg.CardKey)
	if cfg.CardKey == "" {
		return nil, nil
	}
	return &cfg, nil
}

// SaveCardKey stores the user card key in the local config file.
func SaveCardKey(cardKey string) error {
	cardKey = strings.TrimSpace(cardKey)
	if cardKey == "" {
		return fmt.Errorf("card key cannot be empty")
	}

	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(Config{CardKey: cardKey}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	path := filepath.Join(dir, "config.json")
	return os.WriteFile(path, data, 0600)
}

// MaskKey returns a display-safe key preview.
func MaskKey(cardKey string) string {
	cardKey = strings.TrimSpace(cardKey)
	if len(cardKey) <= 8 {
		return strings.Repeat("*", len(cardKey))
	}
	return cardKey[:4] + "..." + cardKey[len(cardKey)-4:]
}
