package schema

import (
	"testing"

	"github.com/vnedyalk0v/mailjail/internal/config"
)

func TestValidateAcceptsExampleLikeConfig(t *testing.T) {
	cfg := &config.Config{
		APIVersion: config.APIVersion,
		Kind:       config.Kind,
		Metadata: config.Metadata{
			Name: "mx1",
		},
		Host: config.Host{
			Hostname:        "mx1.example.com",
			ExternalIFace:   "vtnet0",
			ZFSPool:         "zroot",
			JailDatasetRoot: "zroot/mailjail",
			Bastille: config.Bastille{
				Dataset: "zroot/bastille",
				Release: "15.0-RELEASE",
			},
		},
		Network: config.Network{
			Domain:      "example.com",
			Bridge:      "bridge0",
			JailsSubnet: "10.77.0.0/24",
			Gateway4:    "10.77.0.1",
		},
		TLS: config.TLS{
			Mode:  "acme",
			Email: "admin@example.com",
		},
		Profiles: []string{"core"},
		Modules: map[string]config.ModuleConfig{
			"postfix": {Enabled: true, IP4: "10.77.0.10"},
			"dovecot": {Enabled: true, IP4: "10.77.0.11"},
			"rspamd":  {Enabled: true, IP4: "10.77.0.12"},
			"redis":   {Enabled: true, IP4: "10.77.0.13"},
			"db":      {Enabled: false, IP4: "10.77.0.14"},
			"web":     {Enabled: true, IP4: "10.77.0.15", Edge: "angie"},
		},
	}

	if issues := Validate(cfg); len(issues) != 0 {
		t.Fatalf("expected no validation issues, got %v", issues)
	}
}

func TestValidateRejectsMissingDependencies(t *testing.T) {
	cfg := &config.Config{
		APIVersion: config.APIVersion,
		Kind:       config.Kind,
		Metadata: config.Metadata{
			Name: "mx1",
		},
		Host: config.Host{
			Hostname:        "mx1.example.com",
			ExternalIFace:   "vtnet0",
			ZFSPool:         "zroot",
			JailDatasetRoot: "zroot/mailjail",
			Bastille: config.Bastille{
				Dataset: "zroot/bastille",
				Release: "15.0-RELEASE",
			},
		},
		Network: config.Network{
			Domain:      "example.com",
			Bridge:      "bridge0",
			JailsSubnet: "10.77.0.0/24",
			Gateway4:    "10.77.0.1",
		},
		TLS: config.TLS{
			Mode:  "acme",
			Email: "admin@example.com",
		},
		Profiles: []string{"core"},
		Modules: map[string]config.ModuleConfig{
			"postfix": {Enabled: true, IP4: "10.77.0.10"},
			"dovecot": {Enabled: true, IP4: "10.77.0.11"},
		},
	}

	if issues := Validate(cfg); len(issues) == 0 {
		t.Fatal("expected validation issues for missing postfix dependencies")
	}
}
