package tools

import (
	"e2e/tools/common"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Upstream struct {
	ID               string
	LBType           string `json:"type"`
	ServiceName      string `json:"service_name,omitempty"`
	DupServiceName   string `json:"_service_name,omitempty"`
	DiscoveryType    string `json:"discovery_type,omitempty"`
	DupDiscoveryType string `json:"_discovery_type,omitempty"`
	DiscoveryArgs    map[string]interface{}
	Nodes            map[string]int `json:"nodes,omitempty"`
}

func (up *Upstream) Marshal() string {
	str, _ := json.Marshal(up)
	return string(str)
}

func (up *Upstream) Do(method string) error {
	resp, err := common.RequestCP("/apisix/admin/upstreams/"+up.ID, method, up.Marshal())
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("resp data: ", string(body))
	return err
}

func NewUpstream(id, serviceName, regType string) *Upstream {
	return &Upstream{
		ID:            id,
		LBType:        "roundrobin",
		ServiceName:   serviceName,
		DiscoveryType: regType,
		DiscoveryArgs: make(map[string]interface{}),
	}
}

func NewUpstreamWithNodes(id string, host, port string) *Upstream {
	return &Upstream{
		ID: id,
		Nodes: map[string]int{
			common.DOCKER_GATEWAY + ":" + port: 1,
		},
	}
}

type Route struct {
	URI        string `json:"uri"`
	ID         string
	UpstreamID string    `json:"upstream_id,omitempty"`
	Upstream   *Upstream `json:"upstream,omitempty"`
}

func (r *Route) Marshal() string {
	str, _ := json.Marshal(r)
	return string(str)
}

func (r *Route) Do() error {
	_, err := common.RequestCP("/apisix/admin/routes/"+r.ID, "PUT", r.Marshal())
	return err
}

func NewRoute(id, uri, serviceName, regType string) *Route {
	return &Route{
		URI:      uri,
		ID:       id,
		Upstream: NewUpstream("", serviceName, regType),
	}
}

func NewRouteWithUpstreamID(id, uri, uid string) *Route {
	return &Route{
		URI:        uri,
		ID:         id,
		UpstreamID: uid,
	}
}

type routesResp struct {
	List []struct {
		Value struct {
			ID string `json:"ID"`
		} `json:"value"`
	} `json:"list"`
}

func getResourcesID(resource string) ([]string, error) {
	resp, err := common.RequestCP("/apisix/admin/"+resource, "GET", "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	rouResp := &routesResp{}
	err = json.Unmarshal(body, rouResp)
	if err != nil {
		if _, ok := err.(*json.UnmarshalTypeError); ok {
			return []string{}, nil
		}
		return nil, err
	}
	keys := make([]string, len(rouResp.List))
	for i, v := range rouResp.List {
		keys[i] = v.Value.ID
	}
	return keys, nil
}

func deleteResource(resource, id string) error {
	_, err := common.RequestCP("/apisix/admin/"+resource+"/"+id, "DELETE", "")
	return err
}

func CleanResources(resource string) error {
	fmt.Println("clean all routes from etcd...")
	ids, err := getResourcesID(resource)
	if err != nil {
		return err
	}
	for _, id := range ids {
		err = deleteResource(resource, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func CreateRoutes(routes []*Route) error {
	for _, r := range routes {
		err := r.Do()
		if err != nil {
			return err
		}
	}
	return nil
}

func CreateUpstreams(upstreams []*Upstream) error {
	for _, up := range upstreams {
		err := up.Do("PUT")
		if err != nil {
			return err
		}
	}
	return nil
}

func PatchUpstreams(upstreams []*Upstream) error {
	for _, up := range upstreams {
		dupUp := &Upstream{
			ID:               up.ID,
			LBType:           up.LBType,
			DupDiscoveryType: up.DiscoveryType,
			DupServiceName:   up.ServiceName,
			DiscoveryArgs:    up.DiscoveryArgs,
			Nodes:            up.Nodes,
		}

		err := dupUp.Do("PATCH")
		if err != nil {
			return err
		}
	}
	return nil
}
