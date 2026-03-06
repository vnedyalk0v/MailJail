package plan

import (
	"fmt"

	"github.com/vnedyalk0v/mailjail/internal/config"
	"github.com/vnedyalk0v/mailjail/internal/topology"
)

const (
	redisModuleName   = "redis"
	redisPackageName  = "redis"
	redisServiceName  = "redis"
	redisServiceRCVar = "redis_enable=YES"
)

func buildRedisActions(cfg *config.Config) ([]Action, error) {
	module, ok := cfg.Modules[redisModuleName]
	if !ok || !module.Enabled {
		return nil, nil
	}
	if module.IP4 == "" {
		return nil, fmt.Errorf("redis module is enabled but ip4 is empty")
	}

	jailName := topology.ModuleJailName(cfg, redisModuleName)
	ipCIDR, err := topology.ModuleIPCIDR(cfg, redisModuleName)
	if err != nil {
		return nil, err
	}

	return []Action{
		{
			Type:   ActionEnsureBaseJail,
			Target: jailName,
			Summary: fmt.Sprintf(
				"ensure redis jail %s exists at %s on bridge %s",
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
				"module":  redisModuleName,
				"release": cfg.Host.Bastille.Release,
				"ipCIDR":  ipCIDR,
				"iface":   cfg.Network.Bridge,
			},
		},
		{
			Type:           ActionInstallPackages,
			Target:         jailName,
			Summary:        fmt.Sprintf("install redis package inside jail %s", jailName),
			CommandPreview: []string{"bastille", "pkg", jailName, "install", "-y", redisPackageName},
			Items:          []string{redisPackageName},
			Metadata: map[string]string{
				"module": redisModuleName,
				"jail":   jailName,
			},
		},
		{
			Type:           ActionEnableService,
			Target:         redisServiceName,
			Summary:        fmt.Sprintf("enable redis service inside jail %s", jailName),
			CommandPreview: []string{"bastille", "sysrc", jailName, redisServiceRCVar},
			Metadata: map[string]string{
				"module":  redisModuleName,
				"jail":    jailName,
				"service": redisServiceName,
				"rcvar":   redisServiceRCVar,
			},
		},
		{
			Type:    ActionRenderModuleConfig,
			Target:  jailName,
			Summary: fmt.Sprintf("render secure redis config inside jail %s and reload only on drift", jailName),
			Items:   []string{"/usr/local/etc/redis.conf"},
			Metadata: map[string]string{
				"module":  redisModuleName,
				"jail":    jailName,
				"service": redisServiceName,
			},
		},
		{
			Type:           ActionStartService,
			Target:         redisServiceName,
			Summary:        fmt.Sprintf("start redis service inside jail %s", jailName),
			CommandPreview: []string{"bastille", "service", jailName, redisServiceName, "start"},
			Metadata: map[string]string{
				"module":  redisModuleName,
				"jail":    jailName,
				"service": redisServiceName,
			},
		},
	}, nil
}
