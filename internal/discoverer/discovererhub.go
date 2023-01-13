package discoverer

import (
	"fmt"

	"github.com/api7/gopkg/pkg/log"

	"github.com/api7/apisix-seed/internal/conf"
)

var (
	Discoveries = make(map[string]NewDiscoverFunc)
)

var discovererHub = map[string]Discoverer{}

func InitDiscoverer(key string, disConfig interface{}) error {
	discoverer, err := Discoveries[key](disConfig)
	if err != nil {
		log.Errorf("New %s Discoverer err: %s", key, err)
		return err
	}

	discovererHub[key] = discoverer
	return nil
}

func InitDiscoverers() (err error) {
	for key, disConfig := range conf.DisConfigs {
		err = InitDiscoverer(key, disConfig)
		if err != nil {
			return
		}
	}
	return
}

func GetDiscoverer(key string) Discoverer {
	if d, ok := discovererHub[key]; ok {
		return d
	}
	panic(fmt.Sprintf("no discoverer with key: %s", key))
}

func GetDiscoverers() []Discoverer {
	discoverers := make([]Discoverer, 0, len(discovererHub))
	for _, discoverer := range discovererHub {
		discoverers = append(discoverers, discoverer)
	}
	return discoverers
}
