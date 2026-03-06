package plan

import (
	"fmt"

	"github.com/vnedyalk0v/mailjail/internal/config"
	"github.com/vnedyalk0v/mailjail/internal/topology"
)

const (
	dovecotModuleName   = "dovecot"
	dovecotPackageName  = "dovecot"
	dovecotServiceName  = "dovecot"
	dovecotServiceRCVar = "dovecot_enable=YES"
)

func buildDovecotActions(cfg *config.Config) ([]Action, error) {
	module, ok := cfg.Modules[dovecotModuleName]
	if !ok || !module.Enabled {
		return nil, nil
	}
	if module.IP4 == "" {
		return nil, fmt.Errorf("dovecot module is enabled but ip4 is empty")
	}

	jailName := topology.ModuleJailName(cfg, dovecotModuleName)
	ipCIDR, err := topology.ModuleIPCIDR(cfg, dovecotModuleName)
	if err != nil {
		return nil, err
	}

	return []Action{
		{
			Type:   ActionEnsureBaseJail,
			Target: jailName,
			Summary: fmt.Sprintf(
				"ensure dovecot jail %s exists at %s on bridge %s",
				jailName,
				ipCIDR,
				cfg.Network.Bridge,
			),
			CommandPreview: []string{
				"bastille",
				"create",
				jailName,
				cfg.Host.Bastille.Release,
				ipCIDR,
				cfg.Network.Bridge,
			},
			Metadata: map[string]string{
				"module":  dovecotModuleName,
				"release": cfg.Host.Bastille.Release,
				"ipCIDR":  ipCIDR,
				"iface":   cfg.Network.Bridge,
			},
		},
		{
			Type:           ActionInstallPackages,
			Target:         jailName,
			Summary:        fmt.Sprintf("install dovecot package inside jail %s", jailName),
			CommandPreview: []string{"bastille", "pkg", jailName, "install", "-y", dovecotPackageName},
			Items:          []string{dovecotPackageName},
			Metadata: map[string]string{
				"module": dovecotModuleName,
				"jail":   jailName,
			},
		},
		{
			Type:           ActionEnableService,
			Target:         dovecotServiceName,
			Summary:        fmt.Sprintf("enable dovecot service inside jail %s", jailName),
			CommandPreview: []string{"bastille", "sysrc", jailName, dovecotServiceRCVar},
			Metadata: map[string]string{
				"module":  dovecotModuleName,
				"jail":    jailName,
				"service": dovecotServiceName,
				"rcvar":   dovecotServiceRCVar,
			},
		},
		{
			Type:           ActionStartService,
			Target:         dovecotServiceName,
			Summary:        fmt.Sprintf("start dovecot service inside jail %s", jailName),
			CommandPreview: []string{"bastille", "service", jailName, dovecotServiceName, "start"},
			Metadata: map[string]string{
				"module":  dovecotModuleName,
				"jail":    jailName,
				"service": dovecotServiceName,
			},
		},
	}, nil
}
