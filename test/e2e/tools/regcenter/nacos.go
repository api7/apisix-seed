package regcenter

import (
	"e2e/tools/common"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Nacos struct {
}

func NewNacos() *Nacos {
	return &Nacos{}
}

type servicesResp struct {
	Count int      `json:"count"`
	Doms  []string `json:"doms"`
}

func (n *Nacos) Online(node *common.Node) error {
	//curl -X POST 'http://127.0.0.1:8848/nacos/v1/ns/instance?serviceName=APISIX-NACOS&ip=127.0.0.1&port=10000'
	url := common.NACOS_HOST + "/nacos/v1/ns/instance?healthy=true&" +
		"serviceName=" + node.ServiceName + "&" +
		"ip=" + common.DOCKER_GATEWAY + "&" +
		"port=" + node.Port

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("register instance failed: " + node.String())
	}

	fmt.Println("register instance to Nacos: ", node.String())
	return nil
}

func (n *Nacos) Offline(node *common.Node) error {
	url := common.NACOS_HOST + "/nacos/v1/ns/instance?" +
		"serviceName=" + node.ServiceName + "&" +
		"ip=" + common.DOCKER_GATEWAY + "&" +
		"port=" + node.Port

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("delete instance failed: " + node.String())
	}

	fmt.Println("offline instance to Nacos: ", node.String())
	return nil
}

func (n *Nacos) getServices() ([]string, error) {
	// curl -X GET '127.0.0.1:8848/nacos/v1/ns/service/list?pageNo=1&pageSize=2'
	// we just get 10 services, it's enough
	url := common.NACOS_HOST + "/nacos/v1/ns/service/list?pageNo=1&pageSize=10"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	servResp := &servicesResp{}
	err = json.Unmarshal(body, servResp)
	if err != nil {
		return nil, err
	}
	if servResp.Count > len(servResp.Doms) {

	}
	return servResp.Doms, nil
}

func (n *Nacos) deleteService(service string) error {
	// curl -X DELETE '127.0.0.1:8848/nacos/v1/ns/service?serviceName=APISIX-NACOS'
	url := common.NACOS_HOST + "/nacos/v1/ns/service?serviceName=" + service
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("delete service failed, serviceName=:" + service)
	}
	fmt.Println("delete service, serviceName=" + service)
	return nil
}

func (n *Nacos) Clean() error {
	fmt.Println("clean all service form nacos...")
	services, err := n.getServices()
	if err != nil {
		panic(err)
	}
	for _, srv := range services {
		err = n.deleteService(srv)
		if err != nil {
			return err
		}
	}
	return nil
}
