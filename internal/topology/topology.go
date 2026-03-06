package topology

import (
	"fmt"
	"net/netip"

	"github.com/vnedyalk0v/mailjail/internal/config"
)

func BaseJailName(cfg *config.Config) string {
	return cfg.Metadata.Name + "-base"
}

func DeriveBaseJailIP(cfg *config.Config) (netip.Addr, error) {
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
