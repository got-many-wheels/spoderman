package config

import (
	"encoding/json"
	"os"
)

const (
	DEFAULT_DEPTH   = 2
	DEFAULT_WORKERS = 10
	DEFAULT_BASE    = false
	DEFAULT_VERBOSE = false
)

type Config struct {
	Verbose           *bool    `json:"verbose"`
	Depth             *int     `json:"depth"`
	Workers           *int     `json:"workers"`
	Base              *bool    `json:"base"`
	AllowedDomains    []string `json:"allowedDomains,omitempty"`
	DisallowedDomains []string `json:"disallowedDomains,omitempty"`
}

func Ptr[T any](v T) *T {
	return &v
}

func New() *Config {
	return &Config{
		Workers:           Ptr(DEFAULT_WORKERS),
		Depth:             Ptr(DEFAULT_DEPTH),
		Base:              Ptr(DEFAULT_BASE),
		Verbose:           Ptr(DEFAULT_VERBOSE),
		AllowedDomains:    []string{},
		DisallowedDomains: []string{},
	}
}

func ParseJsonConfig(src string) (*Config, error) {
	cfg := New()

	var tempCfg Config

	f, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(f, &tempCfg); err != nil {
		return nil, err
	}

	if tempCfg.Workers != nil {
		cfg.Workers = tempCfg.Workers
	}
	if tempCfg.Depth != nil {
		cfg.Depth = tempCfg.Depth
	}
	if tempCfg.Base != nil {
		cfg.Base = tempCfg.Base
	}
	if tempCfg.Verbose != nil {
		cfg.Verbose = tempCfg.Verbose
	}

	if len(tempCfg.AllowedDomains) > 0 {
		cfg.AllowedDomains = tempCfg.AllowedDomains
	}
	if len(tempCfg.DisallowedDomains) > 0 {
		cfg.DisallowedDomains = tempCfg.DisallowedDomains
	}

	return cfg, nil
}
