package routing

import (
	"errors"
	"fmt"
	"math/rand"
)

type RoundRobinStrategy struct {
}

func (r *RoundRobinStrategy) Select(spp SelectablePluginPool, spa []SelectablePlugin) (SelectablePlugin, error) {
	var h int = -1
	var index int = -1
	fmt.Printf("Using round robin selection on pool of %d plugins\n", len(spa))
	for i, sp := range spa {
		// look for the lowest hit count
		if sp.HitCount() < h || h == -1 {
			index = i
			h = sp.HitCount()
		}
		// on a hitcount tie we randomly choose one
		if sp.HitCount() == h {
			if rand.Intn(1) == 1 {
				index = i
				h = sp.HitCount()
			}
		}
	}
	if index > -1 {
		fmt.Printf("Selecting plugin at index (%d) with hitcount of (%d)\n", index, spa[index].HitCount())
		return spa[index], nil
	}
	return nil, errors.New("could not select a plugin (round robin strategy)")
}
