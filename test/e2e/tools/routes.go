package tools

import (
	"e2e/tools/common"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type Upstream struct {
	LBType        string `json:"type"`
	ServiceName   string `json:"service_name"`
	DiscoveryType string `json:"discovery_type"`
	DiscoveryArgs map[string]interface{}
}
type Route struct {
	URI      string `json:"uri"`
	ID       string
	Upstream *Upstream `json:"upstream"`
}

func (r *Route) Marshal() string {
	str, _ := json.Marshal(r)
	return string(str)
}

func (r *Route) Do() error {
	url := common.APISIX_HOST + "/apisix/admin/routes/" + r.ID
	req, err := http.NewRequest("PUT", url, strings.NewReader(r.Marshal()))
	if err != nil {
		return err
	}
	req.Header.Add("X-API-KEY", common.APISIX_TOKEN)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("create route failed: /apisix/admin/routes/%s", r.ID))
	}
	fmt.Println("create route successful: /apisix/admin/routes/" + r.ID)
	return nil
}

func NewRoute(id, uri, serviceName, regType string) *Route {
	return &Route{
		URI: uri,
		ID:  id,
		Upstream: &Upstream{
			LBType:        "roundrobin",
			ServiceName:   serviceName,
			DiscoveryType: regType,
			DiscoveryArgs: make(map[string]interface{}),
		},
	}
}

type routesResp struct {
	Node struct {
		Nodes []struct {
			Value struct {
				ID string `json:"ID"`
			} `json:"value"`
		} `json:"nodes"`
	} `json:"node"`
}

func getRoutes() ([]string, error) {
	url := common.APISIX_HOST + "/apisix/admin/routes"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-API-KEY", common.APISIX_TOKEN)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("get routes failed, stats: %s", resp.Status))
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
	keys := make([]string, len(rouResp.Node.Nodes))
	for i, v := range rouResp.Node.Nodes {
		keys[i] = v.Value.ID
	}
	return keys, nil
}

func deleteRoute(id string) error {
	url := common.APISIX_HOST + "/apisix/admin/routes/" + id
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("X-API-KEY", common.APISIX_TOKEN)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("delete route failed: /apisxi/routes/" + id)
	}
	fmt.Println("delete route: ", "/apisxi/routes/"+id)
	return nil
}
func CleanRoutes() error {
	fmt.Println("clean all routes from etcd...")
	ids, err := getRoutes()
	if err != nil {
		return err
	}
	for _, id := range ids {
		err = deleteRoute(id)
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
