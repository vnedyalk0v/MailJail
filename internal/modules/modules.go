package modules

import (
	"fmt"
	"net/netip"
	"sort"

	"github.com/vnedyalk0v/mailjail/internal/config"
)

type Definition struct {
	Name         string
	Dependencies []string
}

type EnabledModule struct {
	Definition Definition
	IP         netip.Addr
}

type ConnectionRule struct {
	From string
	To   string
}

var known = map[string]Definition{
	"postfix": {Name: "postfix", Dependencies: []string{"dovecot", "rspamd"}},
	"dovecot": {Name: "dovecot"},
	"rspamd":  {Name: "rspamd", Dependencies: []string{"redis"}},
	"redis":   {Name: "redis"},
	"db":      {Name: "db"},
	"web":     {Name: "web"},
}

func Known(name string) (Definition, bool) {
	def, ok := known[name]
	return def, ok
}

func Names() []string {
	names := make([]string, 0, len(known))
	for name := range known {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func Enabled(cfg *config.Config) ([]EnabledModule, error) {
	names := make([]string, 0, len(cfg.Modules))
	for name, module := range cfg.Modules {
		if !module.Enabled {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	enabled := make([]EnabledModule, 0, len(names))
	for _, name := range names {
		def, ok := Known(name)
		if !ok {
			continue
		}

		addr, err := netip.ParseAddr(cfg.Modules[name].IP4)
		if err != nil {
			return nil, fmt.Errorf("parse ip for module %s: %w", name, err)
		}

		enabled = append(enabled, EnabledModule{
			Definition: def,
			IP:         addr,
		})
	}

	return enabled, nil
}

func DependencyConnections(cfg *config.Config) []ConnectionRule {
	names := make([]string, 0, len(cfg.Modules))
	for name, module := range cfg.Modules {
		if module.Enabled {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var rules []ConnectionRule
	for _, name := range names {
		def, ok := Known(name)
		if !ok {
			continue
		}
		for _, dep := range def.Dependencies {
			depConfig, ok := cfg.Modules[dep]
			if !ok || !depConfig.Enabled {
				continue
			}
			rules = append(rules, ConnectionRule{
				From: name,
				To:   dep,
			})
		}
	}

	return rules
}
