package storer

import (
	"fmt"
	"reflect"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/core/entity"
)

type HubKey string

const (
	HubKeyRoute    HubKey = "route"
	HubKeyService  HubKey = "service"
	HubKeyUpstream HubKey = "upstream"
)

var storeHub = map[HubKey]*GenericStore{}

func InitStore(key HubKey, opt GenericStoreOption, stg Interface) error {
	s, err := NewGenericStore(string(key), opt, stg)
	if err != nil {
		return err
	}

	storeHub[key] = s
	return nil
}

func InitStores(stg Interface) (err error) {
	err = InitStore(HubKeyRoute, GenericStoreOption{
		BasePath: conf.ETCDConfig.Prefix + "/routes",
		ObjType:  reflect.TypeOf(entity.Route{}),
	}, stg)
	if err != nil {
		return
	}

	err = InitStore(HubKeyService, GenericStoreOption{
		BasePath: conf.ETCDConfig.Prefix + "/services",
		ObjType:  reflect.TypeOf(entity.Service{}),
	}, stg)
	if err != nil {
		return
	}

	err = InitStore(HubKeyUpstream, GenericStoreOption{
		BasePath: conf.ETCDConfig.Prefix + "/upstreams",
		ObjType:  reflect.TypeOf(entity.Upstream{}),
	}, stg)
	if err != nil {
		return
	}

	return
}

func GetStore(key HubKey) *GenericStore {
	if s, ok := storeHub[key]; ok {
		return s
	}
	panic(fmt.Sprintf("no store with key: %s", key))
}

func GetStores() []*GenericStore {
	stores := make([]*GenericStore, 0, len(storeHub))
	for _, store := range storeHub {
		stores = append(stores, store)
	}
	return stores
}
