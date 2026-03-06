package pf

import (
	"strings"
	"testing"

	"github.com/vnedyalk0v/mailjail/internal/config"
)

func TestRenderBuildsDefaultDenyAnchor(t *testing.T) {
	cfg := &config.Config{
		Metadata: config.Metadata{
			Name: "mx1",
		},
		Host: config.Host{
			Bastille: config.Bastille{
				Release: "15.0-RELEASE",
			},
		},
		Network: config.Network{
			Bridge:      "bridge0",
			JailsSubnet: "10.77.0.0/24",
			Gateway4:    "10.77.0.1",
		},
		Modules: map[string]config.ModuleConfig{
			"postfix": {Enabled: true, IP4: "10.77.0.10"},
			"dovecot": {Enabled: true, IP4: "10.77.0.11"},
			"rspamd":  {Enabled: true, IP4: "10.77.0.12"},
			"redis":   {Enabled: true, IP4: "10.77.0.13"},
			"web":     {Enabled: true, IP4: "10.77.0.15", Edge: "angie"},
		},
	}

	rendered, err := Render(cfg)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if !strings.Contains(rendered, "table <mailjail_nodes> const { 10.77.0.10, 10.77.0.11, 10.77.0.12, 10.77.0.13, 10.77.0.15 }") {
		t.Fatalf("expected module table in rendered PF config, got:\n%s", rendered)
	}

	if !strings.Contains(rendered, "pass quick on bridge0 inet proto tcp from 10.77.0.10 to 10.77.0.11 keep state # postfix -> dovecot") {
		t.Fatalf("expected postfix to dovecot rule, got:\n%s", rendered)
	}

	if !strings.Contains(rendered, "pass quick on bridge0 inet proto tcp from 10.77.0.12 to 10.77.0.13 keep state # rspamd -> redis") {
		t.Fatalf("expected rspamd to redis rule, got:\n%s", rendered)
	}

	if !strings.Contains(rendered, "block drop quick on bridge0 inet from <mailjail_nodes> to <mailjail_nodes>") {
		t.Fatalf("expected default deny rule, got:\n%s", rendered)
	}
}
