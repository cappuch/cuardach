package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Engines  EnginesConfig  `yaml:"engines"`
	Privacy  PrivacyConfig  `yaml:"privacy"`
	Cache    CacheConfig    `yaml:"cache"`
	Database DatabaseConfig `yaml:"database"`
	Display  DisplayConfig  `yaml:"display"`
}

type EnginesConfig struct {
	Enabled    []string         `yaml:"enabled"`
	DuckDuckGo DuckDuckGoConfig `yaml:"duckduckgo"`
	Bing       BingConfig       `yaml:"bing"`
}

type DuckDuckGoConfig struct {
	Enabled bool   `yaml:"enabled"`
	BaseURL string `yaml:"base_url"`
}

type BingConfig struct {
	Enabled bool   `yaml:"enabled"`
	BaseURL string `yaml:"base_url"`
}

type PrivacyConfig struct {
	UserAgentRotation bool        `yaml:"user_agent_rotation"`
	StripReferrer     bool        `yaml:"strip_referrer"`
	RandomDelay       DelayConfig `yaml:"random_delay"`
	Proxy             ProxyConfig `yaml:"proxy"`
	DNSOverHTTPS      DNSConfig   `yaml:"dns_over_https"`
}

type DelayConfig struct {
	Enabled bool `yaml:"enabled"`
	MinMs   int  `yaml:"min_ms"`
	MaxMs   int  `yaml:"max_ms"`
}

type ProxyConfig struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
	Type    string `yaml:"type"`
	Rotate  bool   `yaml:"rotate"` // use public proxy pool rotation
}

type DNSConfig struct {
	Enabled     bool   `yaml:"enabled"`
	ResolverURL string `yaml:"resolver_url"`
}

type CacheConfig struct {
	ResultTTL  Duration `yaml:"result_ttl"`
	ContentTTL Duration `yaml:"content_ttl"`
	ContentDir string   `yaml:"content_dir"`
	MaxSizeMB  int      `yaml:"max_size_mb"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type DisplayConfig struct {
	Format     string `yaml:"format"`
	MaxResults int    `yaml:"max_results"`
	Color      bool   `yaml:"color"`
}

// Duration wraps time.Duration for YAML marshaling as a string like "24h".
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = dur
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}

func DefaultConfig() *Config {
	return &Config{
		Engines: EnginesConfig{
			Enabled: []string{"duckduckgo", "bing"},
			DuckDuckGo: DuckDuckGoConfig{
				Enabled: true,
				BaseURL: "https://lite.duckduckgo.com",
			},
			Bing: BingConfig{
				Enabled: true,
				BaseURL: "https://www.bing.com",
			},
		},
		Privacy: PrivacyConfig{
			UserAgentRotation: true,
			StripReferrer:     true,
			RandomDelay: DelayConfig{
				Enabled: true,
				MinMs:   200,
				MaxMs:   2000,
			},
			Proxy: ProxyConfig{
				Enabled: false,
				Address: "socks5://127.0.0.1:9050",
				Type:    "socks5",
			},
			DNSOverHTTPS: DNSConfig{
				Enabled:     false,
				ResolverURL: "https://dns.google/dns-query",
			},
		},
		Cache: CacheConfig{
			ResultTTL:  Duration{24 * time.Hour},
			ContentTTL: Duration{7 * 24 * time.Hour},
			ContentDir: expandPath("~/.cache/cuardach/content"),
			MaxSizeMB:  500,
		},
		Database: DatabaseConfig{
			Path: expandPath("~/.local/share/cuardach/cuardach.db"),
		},
		Display: DisplayConfig{
			Format:     "table",
			MaxResults: 20,
			Color:      true,
		},
	}
}

func ConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "cuardach")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cuardach")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func Load() (*Config, error) {
	cfg := DefaultConfig()
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.Cache.ContentDir = expandPath(cfg.Cache.ContentDir)
	cfg.Database.Path = expandPath(cfg.Database.Path)

	return cfg, nil
}

func Save(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	header := []byte("# cuardach configuration\n# privacy-focused search aggregator\n\n")
	return os.WriteFile(ConfigPath(), append(header, data...), 0600)
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
