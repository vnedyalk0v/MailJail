package plan

import (
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/vnedyalk0v/mailjail/internal/config"
	"github.com/vnedyalk0v/mailjail/internal/host/pf"
	"github.com/vnedyalk0v/mailjail/internal/topology"
)

type ActionType string

const (
	ActionEnsureDataset       ActionType = "EnsureDataset"
	ActionEnsureBastilleSetup ActionType = "EnsureBastilleRelease"
	ActionEnsurePFAnchor      ActionType = "EnsurePFAnchor"
	ActionEnsureBaseJail      ActionType = "EnsureJail"
	ActionInstallPackages     ActionType = "InstallPackages"
	ActionEnableService       ActionType = "EnableService"
	ActionStartService        ActionType = "StartService"
)

type Action struct {
	Type           ActionType        `json:"type"`
	Target         string            `json:"target"`
	Summary        string            `json:"summary"`
	CommandPreview []string          `json:"commandPreview,omitempty"`
	Items          []string          `json:"items,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
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
	baseJailIP, err := topology.DeriveBaseJailIP(cfg)
	if err != nil {
		return nil, err
	}

	baseJailTarget := topology.BaseJailName(cfg)
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
			Type:    ActionEnsurePFAnchor,
			Target:  pf.AnchorName(cfg),
			Summary: fmt.Sprintf("ensure PF anchor %s is rendered and loaded", pf.AnchorName(cfg)),
			CommandPreview: []string{
				"pfctl",
				"-a",
				pf.AnchorName(cfg),
				"-f",
				pf.AnchorFilePath(cfg),
			},
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

	redisActions, err := buildRedisActions(cfg)
	if err != nil {
		return nil, err
	}
	actions = append(actions, redisActions...)

	rspamdActions, err := buildRspamdActions(cfg)
	if err != nil {
		return nil, err
	}
	actions = append(actions, rspamdActions...)

	dovecotActions, err := buildDovecotActions(cfg)
	if err != nil {
		return nil, err
	}
	actions = append(actions, dovecotActions...)

	postfixActions, err := buildPostfixActions(cfg)
	if err != nil {
		return nil, err
	}
	actions = append(actions, postfixActions...)

	return &Plan{
		GeneratedAt: time.Now().UTC(),
		ConfigName:  cfg.Metadata.Name,
		Actions:     actions,
	}, nil
}
