package plan

import (
	"testing"

	"github.com/vnedyalk0v/mailjail/internal/config"
)

func TestBuildCreatesFirstSliceActions(t *testing.T) {
	cfg := &config.Config{
		Metadata: config.Metadata{
			Name: "mx1",
		},
		Host: config.Host{
			JailDatasetRoot: "zroot/mailjail",
			Bastille: config.Bastille{
				Dataset: "zroot/bastille",
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
		},
	}

	pl, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if got, want := len(pl.Actions), 4; got != want {
		t.Fatalf("expected %d actions, got %d", want, got)
	}

	if got, want := pl.Actions[3].Target, "mx1-base"; got != want {
		t.Fatalf("expected base jail target %q, got %q", want, got)
	}

	if got, want := pl.Actions[3].CommandPreview[4], "10.77.0.2/24"; got != want {
		t.Fatalf("expected base jail IP %q, got %q", want, got)
	}
}
