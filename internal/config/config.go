package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	APIVersion = "mailjail.io/v1alpha1"
	Kind       = "MailStack"
)

type Config struct {
	APIVersion string                  `yaml:"apiVersion"`
	Kind       string                  `yaml:"kind"`
	Metadata   Metadata                `yaml:"metadata"`
	Host       Host                    `yaml:"host"`
	Network    Network                 `yaml:"network"`
	TLS        TLS                     `yaml:"tls"`
	Profiles   []string                `yaml:"profiles"`
	Modules    map[string]ModuleConfig `yaml:"modules"`
}

type Metadata struct {
	Name string `yaml:"name"`
}

type Host struct {
	Hostname        string   `yaml:"hostname"`
	ExternalIFace   string   `yaml:"externalInterface"`
	ZFSPool         string   `yaml:"zfsPool"`
	JailDatasetRoot string   `yaml:"jailDatasetRoot"`
	Bastille        Bastille `yaml:"bastille"`
}

type Bastille struct {
	Dataset string `yaml:"dataset"`
	Release string `yaml:"release"`
}

type Network struct {
	Domain      string `yaml:"domain"`
	Bridge      string `yaml:"bridge"`
	JailsSubnet string `yaml:"jailsSubnet"`
	Gateway4    string `yaml:"gateway4"`
}

type TLS struct {
	Mode  string `yaml:"mode"`
	Email string `yaml:"email"`
}

type ModuleConfig struct {
	Enabled bool   `yaml:"enabled"`
	IP4     string `yaml:"ip4"`
	Edge    string `yaml:"edge,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	cfg.ApplyDefaults()
	return &cfg, nil
}

func (c *Config) ApplyDefaults() {
	if c.APIVersion == "" {
		c.APIVersion = APIVersion
	}
	if c.Kind == "" {
		c.Kind = Kind
	}
	if len(c.Profiles) == 0 {
		c.Profiles = []string{"core"}
	}
	if c.Modules == nil {
		c.Modules = make(map[string]ModuleConfig)
	}
	if web := c.Modules["web"]; web.Enabled && web.Edge == "" {
		web.Edge = "angie"
		c.Modules["web"] = web
	}
}
