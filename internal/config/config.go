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
	envAPIURL    = "LUMA_API_URL"
)

// Config is the local CLI configuration.
type Config struct {
	CardKey               string `json:"card_key"`
	APIURL                string `json:"api_url,omitempty"`
	Environment           string `json:"environment,omitempty"`
	PendingAuthDeviceCode string `json:"pending_auth_device_code,omitempty"`
	PendingAuthUserCode   string `json:"pending_auth_user_code,omitempty"`
	PendingAuthVerifyURL  string `json:"pending_auth_verify_url,omitempty"`
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

func readFileConfig() (*Config, error) {
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
	cfg.APIURL = strings.TrimSpace(cfg.APIURL)
	cfg.Environment = strings.TrimSpace(cfg.Environment)
	cfg.PendingAuthDeviceCode = strings.TrimSpace(cfg.PendingAuthDeviceCode)
	cfg.PendingAuthUserCode = strings.TrimSpace(cfg.PendingAuthUserCode)
	cfg.PendingAuthVerifyURL = strings.TrimSpace(cfg.PendingAuthVerifyURL)
	if cfg.CardKey == "" && cfg.APIURL == "" && cfg.Environment == "" && cfg.PendingAuthDeviceCode == "" && cfg.PendingAuthUserCode == "" && cfg.PendingAuthVerifyURL == "" {
		return nil, nil
	}
	return &cfg, nil
}

// Load reads configuration, with environment variables taking precedence.
func Load() (*Config, error) {
	cfg, err := readFileConfig()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = &Config{}
	}
	if key := strings.TrimSpace(os.Getenv(envCardKey)); key != "" {
		cfg.CardKey = key
	}
	if apiURL := strings.TrimSpace(os.Getenv(envAPIURL)); apiURL != "" {
		cfg.APIURL = normalizeAPIURL(apiURL)
		cfg.Environment = EnvironmentName(cfg.APIURL, "")
	}
	if cfg.CardKey == "" && cfg.APIURL == "" && cfg.Environment == "" && cfg.PendingAuthDeviceCode == "" && cfg.PendingAuthUserCode == "" && cfg.PendingAuthVerifyURL == "" {
		return nil, nil
	}
	return cfg, nil
}

func writeConfig(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	path := filepath.Join(dir, "config.json")
	return os.WriteFile(path, data, 0600)
}

// SaveCardKey stores the user card key in the local config file.
func SaveCardKey(cardKey string) error {
	cardKey = strings.TrimSpace(cardKey)
	if cardKey == "" {
		return fmt.Errorf("card key cannot be empty")
	}

	cfg, err := readFileConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.CardKey = cardKey
	cfg.PendingAuthDeviceCode = ""
	cfg.PendingAuthUserCode = ""
	cfg.PendingAuthVerifyURL = ""
	return writeConfig(*cfg)
}

func SavePendingDeviceAuth(deviceCode, userCode, verifyURL string) error {
	deviceCode = strings.TrimSpace(deviceCode)
	if deviceCode == "" {
		return fmt.Errorf("device code cannot be empty")
	}
	cfg, err := readFileConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.PendingAuthDeviceCode = deviceCode
	cfg.PendingAuthUserCode = strings.TrimSpace(userCode)
	cfg.PendingAuthVerifyURL = strings.TrimSpace(verifyURL)
	return writeConfig(*cfg)
}

func ClearPendingDeviceAuth() error {
	cfg, err := readFileConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		return nil
	}
	cfg.PendingAuthDeviceCode = ""
	cfg.PendingAuthUserCode = ""
	cfg.PendingAuthVerifyURL = ""
	return writeConfig(*cfg)
}

func SaveEnvironment(environment, apiURL string) error {
	environment = strings.TrimSpace(environment)
	apiURL = normalizeAPIURL(apiURL)
	if apiURL == "" {
		return fmt.Errorf("api url cannot be empty")
	}
	cfg, err := readFileConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.APIURL = apiURL
	cfg.Environment = EnvironmentName(apiURL, environment)
	return writeConfig(*cfg)
}

func APIBaseURL(defaultURL string) string {
	if apiURL := strings.TrimSpace(os.Getenv(envAPIURL)); apiURL != "" {
		return normalizeAPIURL(apiURL)
	}
	cfg, err := readFileConfig()
	if err == nil && cfg != nil && cfg.APIURL != "" {
		return normalizeAPIURL(cfg.APIURL)
	}
	return normalizeAPIURL(defaultURL)
}

func EnvironmentName(apiURL, configured string) string {
	if configured = strings.TrimSpace(configured); configured != "" {
		return configured
	}
	apiURL = strings.ToLower(strings.TrimSpace(apiURL))
	switch {
	case strings.Contains(apiURL, "localhost") || strings.Contains(apiURL, "127.0.0.1"):
		return "test"
	case apiURL != "":
		return "prod"
	default:
		return ""
	}
}

func normalizeAPIURL(apiURL string) string {
	return strings.TrimRight(strings.TrimSpace(apiURL), "/")
}

// MaskKey returns a display-safe key preview.
func MaskKey(cardKey string) string {
	cardKey = strings.TrimSpace(cardKey)
	if len(cardKey) <= 8 {
		return strings.Repeat("*", len(cardKey))
	}
	return cardKey[:4] + "..." + cardKey[len(cardKey)-4:]
}
