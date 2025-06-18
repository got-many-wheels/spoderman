package crawler

import (
	"net/url"

	"github.com/ganbarodigital/go_glob"
)

type urlFilter interface {
	allow(u string) bool
}

type chainedFilters struct {
	filters []urlFilter
}

type allowedFilter struct {
	allowed []string
}

type disallowedFilter struct {
	disallowed []string
}

func (a *allowedFilter) allow(u string) bool {
	uf, err := url.Parse(u)
	if err != nil {
		return false
	}
	for _, pattern := range a.allowed {
		g := glob.NewGlob(pattern)
		ok, err := g.Match(uf.Hostname())
		if err != nil {
			return false
		}
		if ok {
			return true
		}
	}
	return false
}

func (d *disallowedFilter) allow(u string) bool {
	uf, err := url.Parse(u)
	if err != nil {
		return false
	}
	for _, pattern := range d.disallowed {
		g := glob.NewGlob(pattern)
		ok, err := g.Match(uf.Hostname())
		if err != nil {
			return false
		}
		if ok {
			return false
		}
	}
	return true
}

func (c *chainedFilters) allow(u string) bool {
	for _, f := range c.filters {
		if !f.allow(u) {
			return false
		}
	}
	return true
}
