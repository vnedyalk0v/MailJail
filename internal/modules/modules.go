package modules

import "sort"

type Definition struct {
	Name         string
	Dependencies []string
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
