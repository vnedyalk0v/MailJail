package schema

import (
	"fmt"
	"net/netip"
	"regexp"
	"slices"
	"strings"

	"github.com/vnedyalk0v/mailjail/internal/config"
	"github.com/vnedyalk0v/mailjail/internal/modules"
)

var stackNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

type Issue struct {
	Path    string
	Message string
}

func Validate(cfg *config.Config) []Issue {
	var issues []Issue

	require := func(path, value string) {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, Issue{Path: path, Message: "value is required"})
		}
	}

	if cfg.APIVersion != config.APIVersion {
		issues = append(issues, Issue{
			Path:    "apiVersion",
			Message: fmt.Sprintf("must be %q", config.APIVersion),
		})
	}
	if cfg.Kind != config.Kind {
		issues = append(issues, Issue{
			Path:    "kind",
			Message: fmt.Sprintf("must be %q", config.Kind),
		})
	}

	require("metadata.name", cfg.Metadata.Name)
	if cfg.Metadata.Name != "" && !stackNamePattern.MatchString(cfg.Metadata.Name) {
		issues = append(issues, Issue{
			Path:    "metadata.name",
			Message: "must start with a lowercase letter or digit and contain only lowercase letters, digits, and hyphens",
		})
	}

	require("host.hostname", cfg.Host.Hostname)
	require("host.externalInterface", cfg.Host.ExternalIFace)
	require("host.zfsPool", cfg.Host.ZFSPool)
	require("host.jailDatasetRoot", cfg.Host.JailDatasetRoot)
	require("host.bastille.dataset", cfg.Host.Bastille.Dataset)
	require("host.bastille.release", cfg.Host.Bastille.Release)

	require("network.domain", cfg.Network.Domain)
	require("network.bridge", cfg.Network.Bridge)
	require("network.jailsSubnet", cfg.Network.JailsSubnet)
	require("network.gateway4", cfg.Network.Gateway4)
	require("tls.mode", cfg.TLS.Mode)

	if cfg.TLS.Mode == "acme" {
		require("tls.email", cfg.TLS.Email)
	}

	if len(cfg.Profiles) == 0 {
		issues = append(issues, Issue{Path: "profiles", Message: "at least one profile is required"})
	} else {
		for idx, profile := range cfg.Profiles {
			if !slices.Contains([]string{"core", "groupware"}, profile) {
				issues = append(issues, Issue{
					Path:    fmt.Sprintf("profiles[%d]", idx),
					Message: "must be one of: core, groupware",
				})
			}
		}
	}

	prefix, prefixErr := netip.ParsePrefix(cfg.Network.JailsSubnet)
	if prefixErr != nil {
		issues = append(issues, Issue{
			Path:    "network.jailsSubnet",
			Message: "must be a valid IPv4 CIDR prefix",
		})
	}

	gateway, gatewayErr := netip.ParseAddr(cfg.Network.Gateway4)
	if gatewayErr != nil {
		issues = append(issues, Issue{
			Path:    "network.gateway4",
			Message: "must be a valid IPv4 address",
		})
	} else if prefixErr == nil && !prefix.Contains(gateway) {
		issues = append(issues, Issue{
			Path:    "network.gateway4",
			Message: "must be inside network.jailsSubnet",
		})
	}

	seenIPs := map[netip.Addr]string{}
	for name, module := range cfg.Modules {
		def, ok := modules.Known(name)
		if !ok {
			issues = append(issues, Issue{
				Path:    fmt.Sprintf("modules.%s", name),
				Message: fmt.Sprintf("unknown module name; expected one of: %s", strings.Join(modules.Names(), ", ")),
			})
			continue
		}

		if !module.Enabled {
			continue
		}

		if strings.TrimSpace(module.IP4) == "" {
			issues = append(issues, Issue{
				Path:    fmt.Sprintf("modules.%s.ip4", name),
				Message: "enabled modules must define ip4",
			})
		} else {
			addr, err := netip.ParseAddr(module.IP4)
			if err != nil {
				issues = append(issues, Issue{
					Path:    fmt.Sprintf("modules.%s.ip4", name),
					Message: "must be a valid IPv4 address",
				})
			} else {
				if prefixErr == nil && !prefix.Contains(addr) {
					issues = append(issues, Issue{
						Path:    fmt.Sprintf("modules.%s.ip4", name),
						Message: "must be inside network.jailsSubnet",
					})
				}
				if owner, exists := seenIPs[addr]; exists {
					issues = append(issues, Issue{
						Path:    fmt.Sprintf("modules.%s.ip4", name),
						Message: fmt.Sprintf("duplicates the address already assigned to %s", owner),
					})
				} else {
					seenIPs[addr] = name
				}
				if gatewayErr == nil && addr == gateway {
					issues = append(issues, Issue{
						Path:    fmt.Sprintf("modules.%s.ip4", name),
						Message: "must not match network.gateway4",
					})
				}
			}
		}

		if name == "web" && module.Edge != "" && module.Edge != "angie" {
			issues = append(issues, Issue{
				Path:    "modules.web.edge",
				Message: "must be \"angie\" in the current architecture",
			})
		}

		for _, dep := range def.Dependencies {
			depConfig, ok := cfg.Modules[dep]
			if !ok || !depConfig.Enabled {
				issues = append(issues, Issue{
					Path:    fmt.Sprintf("modules.%s", name),
					Message: fmt.Sprintf("requires enabled module %q", dep),
				})
			}
		}
	}

	return issues
}
