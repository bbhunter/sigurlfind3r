package sources

import (
	"github.com/hueristiq/xurlfind3r/pkg/xurlfind3r/collector/filter"
)

type Keys struct {
	GitHub []string `json:"github"`
	// unused, add for the purpose of adding an asterisk `*` on listing sources
	Intelx     string `json:"intelx"`
	IntelXHost string `json:"intelXHost"`
	IntelXKey  string `json:"intelXKey"`
}

type URL struct {
	Source string
	Value  string
}

// Source is an interface inherited by each passive source
type Source interface {
	Run(keys Keys, ftr filter.Filter) chan URL
	Name() string
}

// List contains the list of all sources. These sources are used by default.
var List = []string{
	"commoncrawl",
	"github",
	"intelx",
	"otx",
	"urlscan",
	"wayback",
	"waybackrobots",
}
