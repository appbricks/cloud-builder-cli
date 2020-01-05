package cookbook_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cbcli "github.com/appbricks/cloud-builder-cli/cookbook"
	"github.com/appbricks/cloud-builder/cookbook"
)

var _ = Describe("Cookbook", func() {

	var (
		err error

		cookbook *cookbook.Cookbook
	)

	Context("initializes an embedded cookbook", func() {

		It("is a valid cookbook", func() {
			cookbook, err = cbcli.NewCookbook()
			Expect(err).NotTo(HaveOccurred())
			err = cookbook.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
