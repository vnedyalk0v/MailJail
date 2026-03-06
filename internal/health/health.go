package health

import (
	"sort"

	"github.com/vnedyalk0v/mailjail/internal/config"
)

type ModuleSnapshot struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

type StateSnapshot struct {
	Modules []ModuleSnapshot `json:"modules"`
}

func DefaultSnapshot(cfg *config.Config) StateSnapshot {
	names := make([]string, 0, len(cfg.Modules))
	for name := range cfg.Modules {
		names = append(names, name)
	}
	sort.Strings(names)

	modules := make([]ModuleSnapshot, 0, len(names))
	for _, name := range names {
		module := cfg.Modules[name]
		status := "disabled"
		if module.Enabled {
			status = "planned"
		}
		modules = append(modules, ModuleSnapshot{
			Name:    name,
			Enabled: module.Enabled,
			Status:  status,
		})
	}

	return StateSnapshot{Modules: modules}
}
