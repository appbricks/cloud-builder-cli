package cloud_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mevansam/goutils/logger"

	"github.com/appbricks/cloud-builder-cli/cmd/cloud"
	"github.com/appbricks/cloud-builder/config"
	"github.com/appbricks/cloud-builder/test/helpers"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_cookbook "github.com/appbricks/cloud-builder-cli/cookbook"

	config_mocks "github.com/appbricks/cloud-builder/config/mocks"
)

var _ = Describe("Cloud", func() {

	BeforeEach(func() {

		cb, err := cbcli_cookbook.NewCookbook()
		Expect(err).NotTo(HaveOccurred())
		Expect(cb).ToNot(BeNil())

		context, err := config.NewConfigContext(cb)
		Expect(err).NotTo(HaveOccurred())

		cbcli_config.Config = config_mocks.NewMockConfig(context)
	})

	Context("list", func() {

		It("returns a list of clouds supported by the cookbook", func() {
			output := helpers.CaptureOutput(func() {
				cloud.ListClouds()
			})

			logger.DebugMessage("Output of command 'cb cloud list':\n%s", output)

			Expect(output).To(Equal(`
This Cloud Builder cookbook supports launching recipes in the public clouds listed below.

+--------+------------------------------------------+
| Name   | Description                              |
+--------+------------------------------------------+
| aws    | Amazon Web Services Cloud Platform       |
| azure  | Microsoft Azure Cloud Computing Platform |
| google | Google Cloud Platform                    |
+--------+------------------------------------------+

`))

		})

		It("returns a list of clouds with region detail", func() {

			output := helpers.CaptureOutput(func() {
				cloud.ListCloudsByRegion()
			})

			logger.DebugMessage("Output of command 'cb cloud list --regions':\n%s", output)

			Expect(output).To(Equal(`
This Cloud Builder cookbook supports launching recipes in the public cloud regions listed below.

Amazon Web Services Cloud Platform
==================================

+----------------+---------------------------+
| Region Name    | Description               |
+----------------+---------------------------+
| ap-east-1      | Asia Pacific (Hong Kong)  |
| ap-northeast-1 | Asia Pacific (Tokyo)      |
| ap-northeast-2 | Asia Pacific (Seoul)      |
| ap-south-1     | Asia Pacific (Mumbai)     |
| ap-southeast-1 | Asia Pacific (Singapore)  |
| ap-southeast-2 | Asia Pacific (Sydney)     |
| ca-central-1   | Canada (Central)          |
| eu-central-1   | EU (Frankfurt)            |
| eu-north-1     | EU (Stockholm)            |
| eu-west-1      | EU (Ireland)              |
| eu-west-2      | EU (London)               |
| eu-west-3      | EU (Paris)                |
| me-south-1     | Middle East (Bahrain)     |
| sa-east-1      | South America (Sao Paulo) |
| us-east-1      | US East (N. Virginia)     |
| us-east-2      | US East (Ohio)            |
| us-west-1      | US West (N. California)   |
| us-west-2      | US West (Oregon)          |
+----------------+---------------------------+

Microsoft Azure Cloud Computing Platform
========================================

+--------------------+----------------------+
| Region Name        | Description          |
+--------------------+----------------------+
| australiacentral   | Australia Central    |
| australiacentral2  | Australia Central 2  |
| australiaeast      | Australia East       |
| australiasoutheast | Australia Southeast  |
| brazilsouth        | Brazil South         |
| canadacentral      | Canada Central       |
| canadaeast         | Canada East          |
| centralindia       | Central India        |
| centralus          | Central US           |
| eastasia           | East Asia            |
| eastus             | East US              |
| eastus2            | East US 2            |
| francecentral      | France Central       |
| francesouth        | France South         |
| germanynorth       | Germany North        |
| germanywestcentral | Germany West Central |
| japaneast          | Japan East           |
| japanwest          | Japan West           |
| koreacentral       | Korea Central        |
| koreasouth         | Korea South          |
| northcentralus     | North Central US     |
| northeurope        | North Europe         |
| southafricanorth   | South Africa North   |
| southafricawest    | South Africa West    |
| southcentralus     | South Central US     |
| southeastasia      | Southeast Asia       |
| southindia         | South India          |
| uaecentral         | UAE Central          |
| uaenorth           | UAE North            |
| uksouth            | UK South             |
| ukwest             | UK West              |
| westcentralus      | West Central US      |
| westeurope         | West Europe          |
| westindia          | West India           |
| westus             | West US              |
| westus2            | West US 2            |
+--------------------+----------------------+

Google Cloud Platform
=====================

+-------------------------+------------------------------------+
| Region Name             | Description                        |
+-------------------------+------------------------------------+
| asia-east1              | Changhua County, Taiwan            |
| asia-east2              | Hong Kong                          |
| asia-northeast1         | Tokyo, Japan                       |
| asia-northeast2         | Osaka, Japan                       |
| asia-south1             | Mumbai, India                      |
| asia-southeast1         | Jurong West, Singapore             |
| australia-southeast1    | Sydney, Australia                  |
| europe-north1           | Hamina, Finland                    |
| europe-west1            | St. Ghislain, Belgium              |
| europe-west2            | London, England, UK                |
| europe-west3            | Frankfurt, Germany                 |
| europe-west4            | Eemshaven, Netherlands             |
| europe-west6            | Zürich, Switzerland                |
| northamerica-northeast1 | Montréal, Québec, Canada           |
| southamerica-east1      | São Paulo, Brazil                  |
| us-central1             | Council Bluffs, Iowa, USA          |
| us-east1                | Moncks Corner, South Carolina, USA |
| us-east4                | Ashburn, Northern Virginia, USA    |
| us-west1                | The Dalles, Oregon, USA            |
| us-west2                | Los Angeles, California, USA       |
+-------------------------+------------------------------------+

`))

		})
	})
})
