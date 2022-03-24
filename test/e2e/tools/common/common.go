package common

import (
	"io/ioutil"
	"net/http"
)

const (
	NACOS_HOST     = "http://127.0.0.1:8848"
	APISIX_HOST    = "http://127.0.0.1:9080"
	APISIX_TOKEN   = "edd1c9f034335f136f87ad84b625c8f1"
	DOCKER_GATEWAY = "172.50.238.1"
)

type Node struct {
	Host        string
	Port        string
	Weight      int
	ServiceName string
	Args        map[string]interface{}
	Metadata    map[string]interface{}
}

func (n *Node) IPPort() string {
	return n.Host + ":" + n.Port
}

func (n *Node) String() string {
	return "serviceName=" + n.ServiceName +
		" ip=" + n.Host + " port=" + n.Port
}

func Request(uri string) (int, string, error) {
	url := APISIX_HOST + uri
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", err
	}
	return resp.StatusCode, string(body), nil
}
