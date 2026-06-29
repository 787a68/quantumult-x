package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Concurrency    int               `yaml:"concurrency"`
	HTTPTimeoutSec int               `yaml:"http_timeout_sec"`
	HTTPRetries    int               `yaml:"http_retries"`
	LogLevel       string            `yaml:"log_level"`
	ReleasesBranch string            `yaml:"releases_branch"`
	Policies       map[string]string `yaml:"policies"`
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
		Policies: map[string]string{
			"direct": "direct",
			"proxy":  "proxy",
			"reject": "reject",
		},
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.Validate()
	cfg.applyPolicyEnvOverrides()
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
	if c.Policies == nil {
		c.Policies = map[string]string{}
	}
	for _, key := range []string{"direct", "proxy", "reject"} {
		if c.Policies[key] == "" {
			c.Policies[key] = key
		}
	}
}

func (c *Config) applyPolicyEnvOverrides() {
	for _, key := range []string{"direct", "proxy", "reject"} {
		if val := os.Getenv("POLICY_" + strings.ToUpper(key)); val != "" {
			c.Policies[key] = val
		}
	}
}
