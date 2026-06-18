package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Concurrency    int    `yaml:"concurrency"`
	HTTPTimeoutSec int    `yaml:"http_timeout_sec"`
	HTTPRetries    int    `yaml:"http_retries"`
	LogLevel       string `yaml:"log_level"`
	ReleasesBranch string `yaml:"releases_branch"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		Concurrency:    16,
		HTTPTimeoutSec: 30,
		HTTPRetries:    2,
		LogLevel:       "info",
		ReleasesBranch: "release",
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.Validate()
	return cfg, nil
}

func (c *Config) Validate() {
	if c.Concurrency < 1 {
		c.Concurrency = 1
	}
	if c.HTTPTimeoutSec < 1 {
		c.HTTPTimeoutSec = 30
	}
	if c.HTTPRetries < 0 {
		c.HTTPRetries = 0
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.ReleasesBranch == "" {
		c.ReleasesBranch = "release"
	}
}
