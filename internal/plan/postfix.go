package plan

import (
	"fmt"

	"github.com/vnedyalk0v/mailjail/internal/config"
	"github.com/vnedyalk0v/mailjail/internal/topology"
)

const (
	postfixModuleName   = "postfix"
	postfixPackageName  = "postfix"
	postfixServiceName  = "postfix"
	postfixServiceRCVar = "postfix_enable=YES"
)

func buildPostfixActions(cfg *config.Config) ([]Action, error) {
	module, ok := cfg.Modules[postfixModuleName]
	if !ok || !module.Enabled {
		return nil, nil
	}
	if module.IP4 == "" {
		return nil, fmt.Errorf("postfix module is enabled but ip4 is empty")
	}

	jailName := topology.ModuleJailName(cfg, postfixModuleName)
	ipCIDR, err := topology.ModuleIPCIDR(cfg, postfixModuleName)
	if err != nil {
		return nil, err
	}

	return []Action{
		{
			Type:   ActionEnsureBaseJail,
			Target: jailName,
			Summary: fmt.Sprintf(
				"ensure postfix jail %s exists at %s on bridge %s",
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
				"module":  postfixModuleName,
				"release": cfg.Host.Bastille.Release,
				"ipCIDR":  ipCIDR,
				"iface":   cfg.Network.Bridge,
			},
		},
		{
			Type:           ActionInstallPackages,
			Target:         jailName,
			Summary:        fmt.Sprintf("install postfix package inside jail %s", jailName),
			CommandPreview: []string{"bastille", "pkg", jailName, "install", "-y", postfixPackageName},
			Items:          []string{postfixPackageName},
			Metadata: map[string]string{
				"module": postfixModuleName,
				"jail":   jailName,
			},
		},
		{
			Type:           ActionEnableService,
			Target:         postfixServiceName,
			Summary:        fmt.Sprintf("enable postfix service inside jail %s", jailName),
			CommandPreview: []string{"bastille", "sysrc", jailName, postfixServiceRCVar},
			Metadata: map[string]string{
				"module":  postfixModuleName,
				"jail":    jailName,
				"service": postfixServiceName,
				"rcvar":   postfixServiceRCVar,
			},
		},
		{
			Type:    ActionRenderModuleConfig,
			Target:  jailName,
			Summary: fmt.Sprintf("render secure postfix baseline config inside jail %s and reload only on drift", jailName),
			Items:   []string{"/usr/local/etc/postfix/main.cf"},
			Metadata: map[string]string{
				"module":  postfixModuleName,
				"jail":    jailName,
				"service": postfixServiceName,
			},
		},
		{
			Type:           ActionStartService,
			Target:         postfixServiceName,
			Summary:        fmt.Sprintf("start postfix service inside jail %s", jailName),
			CommandPreview: []string{"bastille", "service", jailName, postfixServiceName, "start"},
			Metadata: map[string]string{
				"module":  postfixModuleName,
				"jail":    jailName,
				"service": postfixServiceName,
			},
		},
	}, nil
}
