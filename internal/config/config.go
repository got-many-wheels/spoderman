package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DEFAULT_DEPTH   = 2
	DEFAULT_WORKERS = 10
	DEFAULT_BASE    = false
	DEFAULT_VERBOSE = false
)

type Rule struct {
	Name    string `json:"name"    yaml:"name"`
	Pattern string `json:"pattern" yaml:"pattern"`
}

type Config struct {
	Verbose           *bool    `json:"verbose"            yaml:"verbose"`
	Depth             *int     `json:"depth"              yaml:"depth"`
	Workers           *int     `json:"workers"            yaml:"workers"`
	Base              *bool    `json:"base"               yaml:"base"`
	AllowedDomains    []string `json:"allowedDomains,omitempty"    yaml:"allowedDomains,omitempty"`
	DisallowedDomains []string `json:"disallowedDomains,omitempty" yaml:"disallowedDomains,omitempty"`
	Output            string   `json:"output"             yaml:"output"`
	Rules             []Rule   `json:"rules"              yaml:"rules"`
}

func Ptr[T any](v T) *T { return &v }

func New() *Config {
	return &Config{
		Workers:           Ptr(DEFAULT_WORKERS),
		Depth:             Ptr(DEFAULT_DEPTH),
		Base:              Ptr(DEFAULT_BASE),
		Verbose:           Ptr(DEFAULT_VERBOSE),
		AllowedDomains:    []string{},
		DisallowedDomains: []string{},
		Output:            "",
		Rules:             []Rule{},
	}
}

func UnmarshalConfig(src string) (*Config, error) {
	raw, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}

	temp := Config{}
	ext := filepath.Ext(src)

	if ext != ".yaml" && ext != ".yml" {
		return nil, errors.New("unsupported config format: " + ext)
	}

	if err := yaml.Unmarshal(raw, &temp); err != nil {
		return nil, err
	}

	// merge defaults + overrides
	cfg := New()

	if temp.Workers != nil {
		cfg.Workers = temp.Workers
	}
	if temp.Depth != nil {
		cfg.Depth = temp.Depth
	}
	if temp.Base != nil {
		cfg.Base = temp.Base
	}
	if temp.Verbose != nil {
		cfg.Verbose = temp.Verbose
	}

	if len(temp.AllowedDomains) > 0 {
		cfg.AllowedDomains = temp.AllowedDomains
	}
	if len(temp.DisallowedDomains) > 0 {
		cfg.DisallowedDomains = temp.DisallowedDomains
	}

	if temp.Output != "" {
		cfg.Output = temp.Output
	}
	if len(temp.Rules) > 0 {
		cfg.Rules = temp.Rules
	}

	return cfg, nil
}
