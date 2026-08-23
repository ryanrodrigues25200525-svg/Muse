package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Config holds user preferences for the Muse player.
type Config struct {
	DefaultShuffle bool   `json:"default_shuffle"`
	DefaultRepeat  string `json:"default_repeat"`
	LyricsProvider string `json:"lyrics_provider"`
	ArtSize        string `json:"art_size"`
	MCPPort        int    `json:"mcp_port"`
}

// Defaults returns a Config populated with sensible default values.
func Defaults() Config {
	return Config{
		DefaultShuffle: false,
		DefaultRepeat:  "off",
		LyricsProvider: "lrclib",
		ArtSize:        "medium",
		MCPPort:        0,
	}
}

// configDir returns ~/.config/muse, creating it if necessary.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".config", "muse")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}
	return dir, nil
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config from disk. If the file does not exist, it returns
// Defaults(). Returns an error only on actual read/parse failures.
func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Defaults(), nil // dir missing → use defaults
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Defaults(), nil
		}
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	cfg := Defaults()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

// Save writes the config to disk.
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// Get returns the value of a single config key.
func Get(key string) (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}
	return getFieldValue(cfg, key)
}

// Set updates a single config key and saves.
func Set(key, value string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if err := setFieldValue(&cfg, key, value); err != nil {
		return err
	}
	return Save(cfg)
}

// List returns all config keys sorted alphabetically.
func List() map[string]string {
	cfg, err := Load()
	if err != nil {
		return nil
	}
	return map[string]string{
		"default_shuffle": fmt.Sprintf("%t", cfg.DefaultShuffle),
		"default_repeat":  cfg.DefaultRepeat,
		"lyrics_provider": cfg.LyricsProvider,
		"art_size":        cfg.ArtSize,
		"mcp_port":        fmt.Sprintf("%d", cfg.MCPPort),
	}
}

// ListSorted returns keys and values in sorted key order.
func ListSorted() ([]string, []string) {
	m := List()
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vals := make([]string, len(keys))
	for i, k := range keys {
		vals[i] = m[k]
	}
	return keys, vals
}

// Path returns the absolute path to the config file.
func Path() string {
	p, err := configPath()
	if err != nil {
		return "(error: " + err.Error() + ")"
	}
	return p
}

func getFieldValue(cfg Config, key string) (string, error) {
	switch strings.ToLower(key) {
	case "default_shuffle":
		return fmt.Sprintf("%t", cfg.DefaultShuffle), nil
	case "default_repeat":
		return cfg.DefaultRepeat, nil
	case "lyrics_provider":
		return cfg.LyricsProvider, nil
	case "art_size":
		return cfg.ArtSize, nil
	case "mcp_port":
		return fmt.Sprintf("%d", cfg.MCPPort), nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

func setFieldValue(cfg *Config, key, value string) error {
	switch strings.ToLower(key) {
	case "default_shuffle":
		switch strings.ToLower(value) {
		case "true", "1", "yes", "on":
			cfg.DefaultShuffle = true
		case "false", "0", "no", "off":
			cfg.DefaultShuffle = false
		default:
			return fmt.Errorf("default_shuffle must be true/false, got %q", value)
		}
	case "default_repeat":
		v := strings.ToLower(value)
		if v != "off" && v != "one" && v != "all" {
			return fmt.Errorf("default_repeat must be off/one/all, got %q", value)
		}
		cfg.DefaultRepeat = v
	case "lyrics_provider":
		cfg.LyricsProvider = value
	case "art_size":
		cfg.ArtSize = value
	case "mcp_port":
		var port int
		if _, err := fmt.Sscanf(value, "%d", &port); err != nil {
			return fmt.Errorf("mcp_port must be an integer, got %q", value)
		}
		if port < 0 || port > 65535 {
			return fmt.Errorf("mcp_port must be 0-65535, got %d", port)
		}
		cfg.MCPPort = port
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}
