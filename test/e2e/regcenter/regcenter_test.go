package regcenter_test

import (
	"e2e/tools"
	"e2e/tools/common"
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
})
