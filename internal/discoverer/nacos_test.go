package discoverer

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/core/comm"
	"github.com/api7/apisix-seed/internal/utils"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
)

var registerClient naming_client.INamingClient
var TestService string
var TestGroup string

func init() {
	rand.Seed(time.Now().UnixNano())
	TestService = fmt.Sprintf("APISIX-SEED-TEST-%d", rand.Int())
	TestGroup = fmt.Sprintf("Group-%d", rand.Int())
}

func TestServerConfig(t *testing.T) {
	bc := ReadFile("host.yaml")
	nacosConf, _ := conf.DisBuilders["nacos"](bc)
	discoverer, err := NewNacosDiscoverer(nacosConf)
	assert.Nil(t, err)
	nacosDiscoverer := discoverer.(*NacosDiscoverer)

	for auth, serverConfigs := range nacosDiscoverer.ServerConfigs {
		assert.True(t, auth == "username:password", "Test auth")
		assert.Len(t, serverConfigs, 1)

		config := serverConfigs[0]
		assert.True(t, config.Scheme == "https", "Test scheme")
		assert.True(t, config.Port == 8858, "Test port")
	}

	err = nacosDiscoverer.newClient("APISIX")
	assert.Nil(t, err)
}

func TestNacosDiscoverer(t *testing.T) {
	bc := ReadFile("discoverer.yaml")
	nacosConf, _ := conf.DisBuilders["nacos"](bc)

	discoverer, err := NewNacosDiscoverer(nacosConf)
	assert.Nil(t, err)
	nacosDiscoverer := discoverer.(*NacosDiscoverer)

	// For register some services to test
	registerClient, _ = clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &constant.ClientConfig{},
			ServerConfigs: nacosDiscoverer.ServerConfigs[""],
		},
	)

	testQueryService(t, discoverer)
	testWatchService(t, discoverer)
	testUpdateArgs(t, discoverer)
	testDeleteService(t, discoverer)
}

func testQueryService(t *testing.T, discoverer Discoverer) {
	registerService(t, "10.0.0.11", "")

	query := newQuery(t, utils.EventAdd, nil)
	watch := fmt.Sprintf(`key: event, value: update
key: service, value: %s
key: entity, value: upstream1
key: node, value: 10.0.0.11:8848
key: weight, value: 10`, TestService)
	tests := []struct {
		caseDesc  string
		giveQuery comm.Query
		wantMsg   string
	}{
		{
			caseDesc:  "Test query new service",
			giveQuery: query,
			wantMsg:   watch,
		},
		{
			caseDesc:  "Test query cached service",
			giveQuery: query,
			wantMsg:   watch,
		},
	}

	for _, tc := range tests {
		err := discoverer.Query(&tc.giveQuery)
		assert.Nil(t, err)
		watchMsg := <-discoverer.Watch()
		assert.True(t, tc.wantMsg == watchMsg.String(), tc.caseDesc)
	}
}

func testWatchService(t *testing.T, discoverer Discoverer) {
	// register a new instance of the service
	registerService(t, "10.0.0.12", "")

	caseDesc := "Test watch updated service"
	wantMsgs := []string{
		fmt.Sprintf(`key: event, value: update
key: service, value: %s
key: entity, value: upstream1
key: node, value: 10.0.0.12:8848
key: weight, value: 10
key: node, value: 10.0.0.11:8848
key: weight, value: 10`, TestService),
		fmt.Sprintf(`key: event, value: update
key: service, value: %s
key: entity, value: upstream1
key: node, value: 10.0.0.11:8848
key: weight, value: 10
key: node, value: 10.0.0.12:8848
key: weight, value: 10`, TestService)}

	watchMsg := <-discoverer.Watch()
	ret := false
	for _, wantMsg := range wantMsgs {
		if wantMsg == watchMsg.String() {
			ret = true
			break
		}
	}
	assert.True(t, ret, caseDesc)
}

func testUpdateArgs(t *testing.T, discoverer Discoverer) {
	registerService(t, "10.0.0.13", TestGroup)

	caseDesc := "Test update service args"
	wantMsg := fmt.Sprintf(`key: event, value: update
key: service, value: %s
key: entity, value: upstream1
key: node, value: 10.0.0.13:8848
key: weight, value: 10`, TestService)

	// update group argument
	update := newUpdate(t, utils.EventUpdate, nil, map[string]string{"group_name": TestGroup})
	err := discoverer.Update(&update)
	assert.Nil(t, err)
	watchMsg := <-discoverer.Watch()
	assert.True(t, wantMsg == watchMsg.String(), caseDesc)
}

func testDeleteService(t *testing.T, discoverer Discoverer) {
	caseDesc := "Test delete service"
	// First delete the service
	query := newQuery(t, utils.EventDelete, map[string]string{"group_name": TestGroup})
	err := discoverer.Query(&query)
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

func ReadFile(file string) []byte {
	wd, _ := os.Getwd()
	dir := wd[:strings.Index(wd, "internal")]
	path := filepath.Join(dir, "test/testdata/nacos_conf/", file)
	bs, _ := ioutil.ReadFile(path)
	return bs
}

func newQuery(t *testing.T, event string, args map[string]string) comm.Query {
	headerVals := []string{event, "upstream1", TestService}
	query, err := comm.NewQuery(headerVals, args)
	assert.Nil(t, err)

	return query
}

func newUpdate(t *testing.T, event string, oldArgs, newArgs map[string]string) comm.Update {
	headerVals := []string{event, TestService}
	update, err := comm.NewUpdate(headerVals, oldArgs, newArgs)
	assert.Nil(t, err)

	return update
}
