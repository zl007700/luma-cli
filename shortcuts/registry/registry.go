package registry

import (
	"sort"

	"github.com/luma-cli/lumer-cli/shortcuts/asset"
	"github.com/luma-cli/lumer-cli/shortcuts/common"
	"github.com/luma-cli/lumer-cli/shortcuts/media"
	runtimeshortcuts "github.com/luma-cli/lumer-cli/shortcuts/runtime"
	"github.com/luma-cli/lumer-cli/shortcuts/script"
)

// List returns every registered atomic shortcut.
func List() []common.Shortcut {
	var all []common.Shortcut
	all = append(all, media.Shortcuts()...)
	all = append(all, asset.Shortcuts()...)
	all = append(all, runtimeshortcuts.Shortcuts()...)
	all = append(all, script.Shortcuts()...)
	for i := range all {
		all[i].CommandLine = all[i].FullCommand()
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].ID < all[j].ID
	})
	return all
}

// Find returns a shortcut by id.
func Find(id string) (common.Shortcut, bool) {
	for _, sc := range List() {
		if sc.ID == id {
			return sc, true
		}
	}
	return common.Shortcut{}, false
}
