package common

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	NACOS_HOST     = "http://127.0.0.1:8848"
	ZK_HOST        = "127.0.0.1:2181"
	APISIX_CP_HOST = "http://127.0.0.1:9180"
	APISIX_DP_HOST = "http://127.0.0.1:9080"
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

func RequestDP(uri string) (int, string, error) {
	url := APISIX_DP_HOST + uri
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

func RequestCP(uri, method, data string) (*http.Response, error) {
	url := APISIX_CP_HOST + uri
	var body io.Reader
	if data != "" {
		body = strings.NewReader(data)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-API-KEY", APISIX_TOKEN)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, errors.New(fmt.Sprintf("%s %s failed: %s", method, uri, string(body)))
	}
	fmt.Println(method + " route successful: " + uri)
	return resp, nil
}
