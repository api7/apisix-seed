package regcenter_test

import (
	"e2e/tools"
	"e2e/tools/common"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Normal test", Ordered, func() {
	Context("single route, one server", func() {

		type normalCase struct {
			URI        string
			Route      *tools.Route
			Upstream   *tools.Upstream
			Server     *tools.SimServer
			Reg        tools.IRegCenter
			ExpectBody string
		}

		DescribeTable("general logic: route with upstream", Ordered,
			func(tc normalCase) {
				Expect(tools.CreateRoutes([]*tools.Route{tc.Route})).To(BeNil())
				//create sim server
				Expect(tools.CreateSimServer([]*tools.SimServer{tc.Server})).To(BeNil())

				// upstream server online
				Expect(tc.Server.Register(tc.Reg)).To(BeNil())
				time.Sleep(3 * time.Second)
				status, body, err := common.RequestDP(tc.URI)
				Expect(err).To(BeNil())
				Expect(status).To(Equal(200))
				Expect(body).To(Equal(tc.ExpectBody))
				// upstream server offline
				Expect(tc.Server.LogOut(tc.Reg)).To(BeNil())
				time.Sleep(3 * time.Second)
				status, _, err = common.RequestDP(tc.URI)
				Expect(err).To(BeNil())
				Expect(status).To(Equal(503))

				tools.DestroySimServer([]*tools.SimServer{tc.Server})
			},
			Entry("Nacos", normalCase{
				URI:        "/test1",
				Route:      tools.NewRoute("1", "/test1", "APISIX-NACOS", "nacos"),
				Server:     tools.NewSimServer("0.0.0.0", "9990", "APISIX-NACOS"),
				Reg:        tools.NewIRegCenter("nacos"),
				ExpectBody: "response: 0.0.0.0:9990",
			}),
			Entry("Zookeeper", normalCase{
				URI:        "/test2",
				Route:      tools.NewRoute("2", "/test2", "APISIX-ZK", "zookeeper"),
				Server:     tools.NewSimServer("0.0.0.0", "9991", "APISIX-ZK"),
				Reg:        tools.NewIRegCenter("zookeeper"),
				ExpectBody: "response: 0.0.0.0:9991",
			}),
		)

		DescribeTable("general logic: route with upstream_id", Ordered,
			func(tc normalCase) {
				Expect(tools.CreateUpstreams([]*tools.Upstream{tc.Upstream})).To(BeNil())
				Expect(tools.CreateRoutes([]*tools.Route{tc.Route})).To(BeNil())
				//create sim server
				Expect(tools.CreateSimServer([]*tools.SimServer{tc.Server})).To(BeNil())

				// upstream server online
				Expect(tc.Server.Register(tc.Reg)).To(BeNil())
				time.Sleep(3 * time.Second)
				status, body, err := common.RequestDP(tc.URI)
				Expect(err).To(BeNil())
				Expect(status).To(Equal(200))
				Expect(body).To(Equal(tc.ExpectBody))
				// upstream server offline
				Expect(tc.Server.LogOut(tc.Reg)).To(BeNil())
				time.Sleep(3 * time.Second)
				status, _, err = common.RequestDP(tc.URI)
				Expect(err).To(BeNil())
				Expect(status).To(Equal(503))

				tools.DestroySimServer([]*tools.SimServer{tc.Server})
			},
			Entry("Nacos", normalCase{
				URI:        "/test3",
				Upstream:   tools.NewUpstream("1", "APISIX-NACOS", "nacos"),
				Route:      tools.NewRouteWithUpstreamID("1", "/test3", "1"),
				Server:     tools.NewSimServer("0.0.0.0", "9990", "APISIX-NACOS"),
				Reg:        tools.NewIRegCenter("nacos"),
				ExpectBody: "response: 0.0.0.0:9990",
			}),
			Entry("Zookeeper", normalCase{
				URI:        "/test4",
				Upstream:   tools.NewUpstream("2", "APISIX-ZK", "zookeeper"),
				Route:      tools.NewRouteWithUpstreamID("2", "/test4", "2"),
				Server:     tools.NewSimServer("0.0.0.0", "9991", "APISIX-ZK"),
				Reg:        tools.NewIRegCenter("zookeeper"),
				ExpectBody: "response: 0.0.0.0:9991",
			}),
		)
	})

	Context("switch discover mode and nodes mode", func() {
		type normalCase struct {
			URI           string
			Route         *tools.Route
			DisUpstream   *tools.Upstream
			NodesUpstream *tools.Upstream
			DisServer     *tools.SimServer
			NodesServer   *tools.SimServer
			Reg           tools.IRegCenter
		}

		discoverModeFirst := func(tc normalCase) {
			Expect(tools.CreateUpstreams([]*tools.Upstream{tc.DisUpstream})).To(BeNil())
			Expect(tools.CreateRoutes([]*tools.Route{tc.Route})).To(BeNil())
			//create sim server
			Expect(tools.CreateSimServer([]*tools.SimServer{tc.DisServer})).To(BeNil())
			Expect(tools.CreateSimServer([]*tools.SimServer{tc.NodesServer})).To(BeNil())
			// upstream server online
			Expect(tc.DisServer.Register(tc.Reg)).To(BeNil())

			time.Sleep(3 * time.Second)
			status, body, err := common.RequestDP(tc.URI)
			Expect(err).To(BeNil())
			Expect(status).To(Equal(200))
			Expect(body).To(Equal("response: 0.0.0.0:" + tc.DisServer.Node.Port))
		}

		nodesModeFirst := func(tc normalCase) {
			Expect(tools.CreateUpstreams([]*tools.Upstream{tc.NodesUpstream})).To(BeNil())
			Expect(tools.CreateRoutes([]*tools.Route{tc.Route})).To(BeNil())
			//create sim server
			Expect(tools.CreateSimServer([]*tools.SimServer{tc.DisServer})).To(BeNil())
			Expect(tools.CreateSimServer([]*tools.SimServer{tc.NodesServer})).To(BeNil())
			// upstream server online
			Expect(tc.DisServer.Register(tc.Reg)).To(BeNil())

			time.Sleep(3 * time.Second)
			status, body, err := common.RequestDP(tc.URI)
			Expect(err).To(BeNil())
			Expect(status).To(Equal(200))
			expectBody := ""
			for k := range tc.NodesUpstream.Nodes {
				// host is DOCKERGATEWAY, we should replace it to 0.0.0.0
				port := strings.Split(k, ":")[1]
				expectBody = "response: 0.0.0.0:" + port
			}
			Expect(body).To(Equal(expectBody))
		}

		changeNodes2Discover := func(tc normalCase, method string) {
			fmt.Println("change nodes to discover mode")
			if method == "PATCH" {
				// use _service_name instead of service_name
				Expect(tools.PatchUpstreams([]*tools.Upstream{tc.DisUpstream})).To(BeNil())
			} else {
				Expect(tools.CreateUpstreams([]*tools.Upstream{tc.DisUpstream})).To(BeNil())
			}

			time.Sleep(3 * time.Second)
			status, body, err := common.RequestDP(tc.URI)
			Expect(err).To(BeNil())
			Expect(status).To(Equal(200))
			Expect(body).To(Equal("response: 0.0.0.0:" + tc.DisServer.Node.Port))
		}

		changeDiscover2Nodes := func(tc normalCase) {
			fmt.Println("change discover to nodes mode")
			// Just use PUT method, for Patch method need delete "service_name" and "discover_type" attr
			// it's not related to apisix-seed
			Expect(tools.CreateUpstreams([]*tools.Upstream{tc.NodesUpstream})).To(BeNil())
			time.Sleep(3 * time.Second)
			status, body, err := common.RequestDP(tc.URI)
			Expect(err).To(BeNil())
			Expect(status).To(Equal(200))
			expectBody := ""
			for k := range tc.NodesUpstream.Nodes {
				// host is DOCKERGATEWAY, we should replace it to 0.0.0.0
				port := strings.Split(k, ":")[1]
				expectBody = "response: 0.0.0.0:" + port
			}
			Expect(body).To(Equal(expectBody))
		}

		DescribeTable("discover mode to nodes to discover: discover first", Ordered,
			func(tc normalCase) {
				discoverModeFirst(tc)
				changeDiscover2Nodes(tc)
				changeNodes2Discover(tc, "PUT")
				changeDiscover2Nodes(tc)
				changeNodes2Discover(tc, "PATCH")

				tools.DestroySimServer([]*tools.SimServer{tc.DisServer})
				tools.DestroySimServer([]*tools.SimServer{tc.NodesServer})
			},
			Entry("nacos", normalCase{
				URI:           "/test5",
				DisUpstream:   tools.NewUpstream("1", "APISIX-NACOS", "nacos"),
				DisServer:     tools.NewSimServer("0.0.0.0", "9990", "APISIX-NACOS"),
				NodesUpstream: tools.NewUpstreamWithNodes("1", "0.0.0.0", "9991"),
				NodesServer:   tools.NewSimServer("0.0.0.0", "9991", ""),
				Route:         tools.NewRouteWithUpstreamID("1", "/test5", "1"),
				Reg:           tools.NewIRegCenter("nacos"),
			}),
			Entry("zookeeper", normalCase{
				URI:           "/test6",
				DisUpstream:   tools.NewUpstream("1", "APISIX-ZK", "zookeeper"),
				DisServer:     tools.NewSimServer("0.0.0.0", "9990", "APISIX-ZK"),
				NodesUpstream: tools.NewUpstreamWithNodes("1", "0.0.0.0", "9991"),
				NodesServer:   tools.NewSimServer("0.0.0.0", "9991", ""),
				Route:         tools.NewRouteWithUpstreamID("1", "/test6", "1"),
				Reg:           tools.NewIRegCenter("zookeeper"),
			}),
		)

		DescribeTable("discover mode to nodes to discover: nodes first", Ordered,
			func(tc normalCase) {
				nodesModeFirst(tc)
				changeNodes2Discover(tc, "PUT")
				changeDiscover2Nodes(tc)
				changeNodes2Discover(tc, "PATCH")
				changeDiscover2Nodes(tc)

				tools.DestroySimServer([]*tools.SimServer{tc.DisServer})
				tools.DestroySimServer([]*tools.SimServer{tc.NodesServer})
			},
			Entry("nacos", normalCase{
				URI:           "/test7",
				DisUpstream:   tools.NewUpstream("1", "APISIX-NACOS", "nacos"),
				DisServer:     tools.NewSimServer("0.0.0.0", "9990", "APISIX-NACOS"),
				NodesUpstream: tools.NewUpstreamWithNodes("1", "0.0.0.0", "9991"),
				NodesServer:   tools.NewSimServer("0.0.0.0", "9991", ""),
				Route:         tools.NewRouteWithUpstreamID("1", "/test7", "1"),
				Reg:           tools.NewIRegCenter("nacos"),
			}),
			Entry("zookeeper", normalCase{
				URI:           "/test8",
				DisUpstream:   tools.NewUpstream("1", "APISIX-ZK", "zookeeper"),
				DisServer:     tools.NewSimServer("0.0.0.0", "9990", "APISIX-ZK"),
				NodesUpstream: tools.NewUpstreamWithNodes("1", "0.0.0.0", "9991"),
				NodesServer:   tools.NewSimServer("0.0.0.0", "9991", ""),
				Route:         tools.NewRouteWithUpstreamID("1", "/test8", "1"),
				Reg:           tools.NewIRegCenter("zookeeper"),
			}),
		)
	})
})
