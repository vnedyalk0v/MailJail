package plan

import (
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/vnedyalk0v/mailjail/internal/config"
)

type ActionType string

const (
	ActionEnsureDataset       ActionType = "EnsureDataset"
	ActionEnsureBastilleSetup ActionType = "EnsureBastilleRelease"
	ActionEnsureBaseJail      ActionType = "EnsureJail"
)

type Action struct {
	Type           ActionType `json:"type"`
	Target         string     `json:"target"`
	Summary        string     `json:"summary"`
	CommandPreview []string   `json:"commandPreview,omitempty"`
}

func (a Action) CommandString() string {
	return strings.Join(a.CommandPreview, " ")
}

type Plan struct {
	GeneratedAt time.Time `json:"generatedAt"`
	ConfigName  string    `json:"configName"`
	Actions     []Action  `json:"actions"`
}

func Build(cfg *config.Config) (*Plan, error) {
	baseJailIP, err := deriveBaseJailIP(cfg)
	if err != nil {
		return nil, err
	}

	baseJailTarget := BaseJailName(cfg)
	release := cfg.Host.Bastille.Release
	prefixBits, _ := netip.ParsePrefix(cfg.Network.JailsSubnet)

	actions := []Action{
		{
			Type:           ActionEnsureDataset,
			Target:         cfg.Host.Bastille.Dataset,
			Summary:        fmt.Sprintf("ensure Bastille dataset %s exists", cfg.Host.Bastille.Dataset),
			CommandPreview: []string{"zfs", "create", "-p", cfg.Host.Bastille.Dataset},
		},
		{
			Type:           ActionEnsureBastilleSetup,
			Target:         release,
			Summary:        fmt.Sprintf("ensure Bastille release %s is bootstrapped", release),
			CommandPreview: []string{"bastille", "bootstrap", release},
		},
		{
			Type:           ActionEnsureDataset,
			Target:         cfg.Host.JailDatasetRoot,
			Summary:        fmt.Sprintf("ensure MailJail dataset root %s exists", cfg.Host.JailDatasetRoot),
			CommandPreview: []string{"zfs", "create", "-p", cfg.Host.JailDatasetRoot},
		},
		{
			Type:   ActionEnsureBaseJail,
			Target: baseJailTarget,
			Summary: fmt.Sprintf(
				"ensure base jail %s exists at %s/%d on bridge %s",
				baseJailTarget,
				baseJailIP.String(),
				prefixBits.Bits(),
				cfg.Network.Bridge,
			),
			CommandPreview: []string{
				"bastille",
				"create",
				baseJailTarget,
				release,
				fmt.Sprintf("%s/%d", baseJailIP.String(), prefixBits.Bits()),
				cfg.Network.Bridge,
			},
		},
	}

	return &Plan{
		GeneratedAt: time.Now().UTC(),
		ConfigName:  cfg.Metadata.Name,
		Actions:     actions,
	}, nil
}

func BaseJailName(cfg *config.Config) string {
	return cfg.Metadata.Name + "-base"
}

func deriveBaseJailIP(cfg *config.Config) (netip.Addr, error) {
	prefix, err := netip.ParsePrefix(cfg.Network.JailsSubnet)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("parse subnet: %w", err)
	}
	gateway, err := netip.ParseAddr(cfg.Network.Gateway4)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("parse gateway: %w", err)
	}

	used := map[netip.Addr]struct{}{
		gateway: {},
	}
	for _, module := range cfg.Modules {
		if !module.Enabled || module.IP4 == "" {
			continue
		}
		addr, parseErr := netip.ParseAddr(module.IP4)
		if parseErr != nil {
			continue
		}
		used[addr] = struct{}{}
	}

	for candidate := prefix.Addr().Next(); prefix.Contains(candidate); candidate = candidate.Next() {
		if !candidate.Is4() || candidate == prefix.Addr() {
			continue
		}
		if _, exists := used[candidate]; exists {
			continue
		}
		return candidate, nil
	}

	return netip.Addr{}, fmt.Errorf("no free IPs available in %s for the base jail", cfg.Network.JailsSubnet)
}
