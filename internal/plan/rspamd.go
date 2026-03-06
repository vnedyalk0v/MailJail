package plan

import (
	"fmt"

	"github.com/vnedyalk0v/mailjail/internal/config"
	"github.com/vnedyalk0v/mailjail/internal/topology"
)

const (
	rspamdModuleName   = "rspamd"
	rspamdPackageName  = "rspamd"
	rspamdServiceName  = "rspamd"
	rspamdServiceRCVar = "rspamd_enable=YES"
)

func buildRspamdActions(cfg *config.Config) ([]Action, error) {
	module, ok := cfg.Modules[rspamdModuleName]
	if !ok || !module.Enabled {
		return nil, nil
	}
	if module.IP4 == "" {
		return nil, fmt.Errorf("rspamd module is enabled but ip4 is empty")
	}

	jailName := topology.ModuleJailName(cfg, rspamdModuleName)
	ipCIDR, err := topology.ModuleIPCIDR(cfg, rspamdModuleName)
	if err != nil {
		return nil, err
	}

	return []Action{
		{
			Type:   ActionEnsureBaseJail,
			Target: jailName,
			Summary: fmt.Sprintf(
				"ensure rspamd jail %s exists at %s on bridge %s",
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
				"module":  rspamdModuleName,
				"release": cfg.Host.Bastille.Release,
				"ipCIDR":  ipCIDR,
				"iface":   cfg.Network.Bridge,
			},
		},
		{
			Type:           ActionInstallPackages,
			Target:         jailName,
			Summary:        fmt.Sprintf("install rspamd package inside jail %s", jailName),
			CommandPreview: []string{"bastille", "pkg", jailName, "install", "-y", rspamdPackageName},
			Items:          []string{rspamdPackageName},
			Metadata: map[string]string{
				"module": rspamdModuleName,
				"jail":   jailName,
			},
		},
		{
			Type:           ActionEnableService,
			Target:         rspamdServiceName,
			Summary:        fmt.Sprintf("enable rspamd service inside jail %s", jailName),
			CommandPreview: []string{"bastille", "sysrc", jailName, rspamdServiceRCVar},
			Metadata: map[string]string{
				"module":  rspamdModuleName,
				"jail":    jailName,
				"service": rspamdServiceName,
				"rcvar":   rspamdServiceRCVar,
			},
		},
		{
			Type:    ActionRenderModuleConfig,
			Target:  jailName,
			Summary: fmt.Sprintf("render secure rspamd config inside jail %s and reload only on drift", jailName),
			Items: []string{
				"/usr/local/etc/rspamd/local.d/worker-proxy.inc",
				"/usr/local/etc/rspamd/local.d/worker-controller.inc",
				"/usr/local/etc/rspamd/local.d/worker-normal.inc",
				"/usr/local/etc/rspamd/local.d/redis.conf",
			},
			Metadata: map[string]string{
				"module":  rspamdModuleName,
				"jail":    jailName,
				"service": rspamdServiceName,
			},
		},
		{
			Type:           ActionStartService,
			Target:         rspamdServiceName,
			Summary:        fmt.Sprintf("start rspamd service inside jail %s", jailName),
			CommandPreview: []string{"bastille", "service", jailName, rspamdServiceName, "start"},
			Metadata: map[string]string{
				"module":  rspamdModuleName,
				"jail":    jailName,
				"service": rspamdServiceName,
			},
		},
	}, nil
}
