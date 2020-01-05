package recipe_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mevansam/goutils/logger"

	"github.com/appbricks/cloud-builder-cli/cmd/recipe"
	"github.com/appbricks/cloud-builder/config"
	"github.com/appbricks/cloud-builder/test/helpers"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_cookbook "github.com/appbricks/cloud-builder-cli/cookbook"

	config_mocks "github.com/appbricks/cloud-builder/config/mocks"
)

var _ = Describe("Recipe", func() {

	BeforeEach(func() {

		cb, err := cbcli_cookbook.NewCookbook()
		Expect(err).NotTo(HaveOccurred())
		Expect(cb).ToNot(BeNil())

		context, err := config.NewConfigContext(cb)
		Expect(err).NotTo(HaveOccurred())

		cbcli_config.Config = config_mocks.NewMockConfig(context)
	})

	Context("list", func() {

		It("returns the list of recipes available in cookbook", func() {

			output := helpers.CaptureOutput(func() {
				recipe.ListRecipes()
			})

			logger.DebugMessage("Output of command 'cb list recipes':\n%s", output)

			Expect(output).To(Equal(`
This Cloud Builder cookbook supports launching the following recipes.

+--------+------------------+
| Name   | Supported Clouds |
+--------+------------------+
| basic  | aws, google      |
| simple | google           |
+--------+------------------+

`))

		})

		It("returns the list of recipes for a particular cloud in the cookbook", func() {

			output := helpers.CaptureOutput(func() {
				recipe.ListRecipesForCloud("aws")
			})

			logger.DebugMessage("Output of command 'cb list recipes':\n%s", output)

			Expect(output).To(Equal(`
This Cloud Builder cookbook supports launching the following recipes in the 'aws' cloud.

+-------------+
| Recipe Name |
+-------------+
| basic       |
+-------------+

`))

		})
	})
})
