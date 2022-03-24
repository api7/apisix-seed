package regcenter_test

import (
	"e2e/tools"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
	Expect(tools.CleanRoutes()).To(BeNil())
	Expect(tools.NewIRegCenter("nacos").Clean()).To(BeNil())

})

func TestRegcenter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Regcenter Suite")
}
