package regcenter_test

import (
	"e2e/tools"
	"e2e/tools/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Nacos", Ordered, func() {
	Context("single route, one server", func() {
		var s1 *tools.SimServer
		var reg tools.IRegCenter
		BeforeAll(func() {
			Expect(tools.CreateRoutes([]*tools.Route{
				tools.NewRoute("1", "/test1", "APISIX-NACOS", "nacos"),
			})).To(BeNil())
			//create sim server
			s1 = tools.NewSimServer("0.0.0.0", "9990", "APISIX-NACOS")
			Expect(tools.CreateSimServer([]*tools.SimServer{
				s1,
			})).To(BeNil())
			reg = tools.NewIRegCenter("nacos")
		})

		It("request successful, returns 200", func() {
			// request
			// register server to discover center
			s1.Register(reg)
			time.Sleep(3 * time.Second)
			status, body, err := common.Request("/test1")
			Expect(err).To(BeNil())
			Expect(status).To(Equal(200))
			Expect(body).To(Equal("response: 0.0.0.0:9990"))
		})

		It("request failed, returns 503", func() {
			s1.LogOut(reg)
			time.Sleep(3 * time.Second)
			status, _, err := common.Request("/test1")
			Expect(err).To(BeNil())
			Expect(status).To(Equal(503))
		})
	})
})
