package discoverer

import (
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/api7/apisix-seed/internal/core/message"
	"gopkg.in/yaml.v3"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
)

var naYamlConfig = `
host:
  - "http://127.0.0.1:8848"
prefix: ~
`

var naYamlConfigWithPasswd = `
host:
  - "https://console.nacos.io:8858"
user:       "username"
password:   "password"
`

func getNaConfig(str string) (*conf.Nacos, error) {
	naConf := &conf.Nacos{}
	err := yaml.Unmarshal([]byte(str), naConf)
	if err != nil {
		return nil, err
	}
	return naConf, nil
}

var TestService string
var TestGroup string

func init() {
	rand.Seed(time.Now().UnixNano())
	TestService = fmt.Sprintf("APISIX-SEED-TEST-%d", rand.Int())
	TestGroup = fmt.Sprintf("Group-%d", rand.Int())
}

func TestServerConfig(t *testing.T) {
	nacosConf, err := getNaConfig(naYamlConfigWithPasswd)
	assert.Nil(t, err)
	discoverer, err := NewNacosDiscoverer(nacosConf)
	assert.Nil(t, err)
	nacosDiscoverer := discoverer.(*NacosDiscoverer)

	for auth, serverConfigs := range nacosDiscoverer.ServerConfigs {
		assert.True(t, auth == "username", "Test auth")
		assert.Len(t, serverConfigs, 1)

		config := serverConfigs[0]
		assert.True(t, config.Scheme == "https", "Test scheme")
		assert.True(t, config.Port == 8858, "Test port")
	}

	err = nacosDiscoverer.newClient("APISIX")
	assert.Nil(t, err)
}

func TestNacosDiscoverer(t *testing.T) {
	nacosConf, err := getNaConfig(naYamlConfig)
	assert.Nil(t, err)

	discoverer, err := NewNacosDiscoverer(nacosConf)
	assert.Nil(t, err)

	testQueryService(t, discoverer)
	testUpdateArgs(t, discoverer)
	testDeleteService(t, discoverer)
}

func testQueryService(t *testing.T, discoverer Discoverer) {
	registerService(t, "10.0.0.11", "")

	a6fmt := `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"%s"}}`
	a6Str := fmt.Sprintf(a6fmt, TestService)
	expectA6StrFmt := `{
	    "uri": "/hh",
	    "upstream": {
	        "nodes": [
	            {"host":"%s","port": %d,"weight":%d}
	        ],
	        "_discovery_type":"nacos","_service_name":"%s"}}`
	msg, err := message.NewMessage("/apisix/routes/1", []byte(a6Str), 1, message.EventAdd, message.A6RoutesConf)
	assert.Nil(t, err)
	tests := []struct {
		caseDesc  string
		givenMsg  *message.Message
		wantA6Str string
	}{
		{
			caseDesc:  "Test query new service",
			givenMsg:  msg,
			wantA6Str: fmt.Sprintf(expectA6StrFmt, "10.0.0.11", 8848, 10, TestService),
		},
		{
			caseDesc:  "Test query new service",
			givenMsg:  msg,
			wantA6Str: fmt.Sprintf(expectA6StrFmt, "10.0.0.11", 8848, 10, TestService),
		},
	}

	for _, tc := range tests {
		err = discoverer.Query(tc.givenMsg)
		assert.Nil(t, err)
		watchMsg := <-discoverer.Watch()
		assert.JSONEq(t, tc.wantA6Str, naMsg2Value(watchMsg), tc.caseDesc)
	}
}

func testUpdateArgs(t *testing.T, discoverer Discoverer) {
	registerService(t, "10.0.0.13", TestGroup)

	caseDesc := "Test update service args"
	oldA6StrFmt := `{
    "uri": "/hh",
    "upstream": {
        "nodes": [
            {"host": "%s","port": %d,"weight":%d}
        ],
        "_discovery_type":"nacos","_service_name":"%s"}}`
	oldA6Str := fmt.Sprintf(oldA6StrFmt, "10.0.0.11", 8848, 10, TestService)

	oldMsg, err := message.NewMessage("/apisix/routes/1", []byte(oldA6Str), 1, message.EventAdd, message.A6RoutesConf)
	assert.Nil(t, err)

	a6fmt := `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"%s","discovery_args":{"group_name":"%s"}}}`
	a6Str := fmt.Sprintf(a6fmt, TestService, TestGroup)
	msg, err := message.NewMessage("/apisix/routes/1", []byte(a6Str), 1, message.EventAdd, message.A6RoutesConf)
	assert.Nil(t, err)
	err = discoverer.Update(oldMsg, msg)
	assert.Nil(t, err, caseDesc)

	expectA6StrFmt := `{
    "uri": "/hh",
    "upstream": {
        "nodes": [
            {"host": "%s","port": %d,"weight":%d}
        ],
        "_discovery_type":"nacos","_service_name":"%s","discovery_args":{"group_name":"%s"}}}`
	expectA6Str := fmt.Sprintf(expectA6StrFmt, "10.0.0.13", 8848, 10, TestService, TestGroup)

	watchMsg := <-discoverer.Watch()
	assert.JSONEq(t, expectA6Str, naMsg2Value(watchMsg), caseDesc)
}

func testDeleteService(t *testing.T, discoverer Discoverer) {
	caseDesc := "Test delete service"
	// First delete the service
	a6fmt := `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"%s","discovery_args":{"group_name":"%s"}}}`
	a6Str := fmt.Sprintf(a6fmt, TestService, TestGroup)
	msg, err := message.NewMessage("/apisix/routes/1", []byte(a6Str), 1, message.EventAdd, message.A6RoutesConf)
	assert.Nil(t, err)
	err = discoverer.Delete(msg)
	assert.Nil(t, err)

	registerService(t, "10.0.0.14", TestGroup)
	select {
	case <-discoverer.Watch():
		// Since the subscription is cancelled, the receiving operation will be blocked
		assert.True(t, false, caseDesc)
	case <-time.After(3 * time.Second):
	}
}

func registerService(t *testing.T, ip string, group string) {
	conf, err := getNaConfig(naYamlConfig)
	assert.Nil(t, err)
	serverConfigs := make([]constant.ServerConfig, 0, len(conf.Host))
	for _, host := range conf.Host {
		u, _ := url.Parse(host)
		port := 8848 // nacos default port
		if portStr := u.Port(); len(portStr) != 0 {
			port, _ = strconv.Atoi(portStr)
		}
		serverConfig := *constant.NewServerConfig(
			u.Hostname(),
			uint64(port),
			constant.WithScheme(u.Scheme),
			constant.WithContextPath(conf.Prefix),
		)
		serverConfigs = append(serverConfigs, serverConfig)
	}
	//Another way of create clientConfig
	clientConfig := constant.NewClientConfig(
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogLevel("info"),
	)

	// For register some services to test
	registerClient, _ := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	success, err := registerClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        8848,
		ServiceName: TestService,
		GroupName:   group,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Metadata:    map[string]string{"idc": "shanghai"},
	})
	assert.NoError(t, err)
	assert.True(t, success)
}

func naMsg2Value(msg *message.Message) string {
	str, _ := msg.Marshal()
	return string(str)
}
