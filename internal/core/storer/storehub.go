package storer

import (
	"fmt"
	"github.com/api7/gopkg/pkg/log"
	"strings"

	"github.com/api7/apisix-seed/internal/conf"
)

var storeHub = map[string]*GenericStore{}

func InitStore(key string, opt GenericStoreOption, stg Interface) error {
	s, err := NewGenericStore(key, opt, stg)
	if err != nil {
		log.Errorf("New %s GenericStore err: %s", key, err)
		return err
	}

	storeHub[key] = s
	return nil
}

func InitStores(stg Interface) (err error) {
	err = InitStore("routes", GenericStoreOption{
		BasePath: conf.ETCDConfig.Prefix + "/routes",
		Prefix:   conf.ETCDConfig.Prefix,
	}, stg)
	if err != nil {
		return
	}

	err = InitStore("services", GenericStoreOption{
		BasePath: conf.ETCDConfig.Prefix + "/services",
		Prefix:   conf.ETCDConfig.Prefix,
	}, stg)
	if err != nil {
		return
	}

	err = InitStore("upstreams", GenericStoreOption{
		BasePath: conf.ETCDConfig.Prefix + "/upstreams",
		Prefix:   conf.ETCDConfig.Prefix,
	}, stg)
	if err != nil {
		return
	}

	return
}

func FromatKey(key, prefix string) (string, string, string) {
	s := strings.TrimPrefix(key, prefix)
	if s == "" || s == key {
		return "", "", ""
	}

	entityindecx := strings.IndexByte(s[1:], '/')
	if entityindecx == -1 {
		return prefix, "", ""
	}
	entity := s[1 : entityindecx+1]
	id := s[entityindecx+2:]

	return prefix, entity, id
}

func GetStore(entity string) *GenericStore {
	if s, ok := storeHub[entity]; ok {
		return s
	}
	panic(fmt.Sprintf("no store with key: %s", entity))
}

func GetStores() []*GenericStore {
	stores := make([]*GenericStore, 0, len(storeHub))
	for _, store := range storeHub {
		stores = append(stores, store)
	}
	return stores
}

func ClrearStores() {
	for key := range storeHub {
		delete(storeHub, key)
	}
}
