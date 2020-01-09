package cloud

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"

	"github.com/appbricks/cloud-builder-cli/config"

	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var configureCommand = &cobra.Command{
	Use: "configure [cloud]",

	Short: "Configure cloud credentials.",
	Long: `
Recipe resources are created in the public cloud using your cloud
credentials. This requires that you have a valid cloud account in one
or more of the clouds the recipe can be launched in. This sub-command
can be used to configure your cloud credentials for the cloud
environments you wish to target.
`,

	Run: func(cmd *cobra.Command, args []string) {
		ConfigureCloud(args[0])
	},
	Args: cobra.ExactArgs(1),
}

func ConfigureCloud(name string) {

	var (
		err error

		provider  provider.CloudProvider
		inputForm forms.InputForm
	)

	if provider, err = config.Config.Context().GetCloudProvider(name); err == nil && provider != nil {

		if inputForm, err = provider.InputForm(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		if err = ux.GetFormInput(inputForm,
			"Cloud Provider Configuration",
			"CONFIGURATION DATA INPUT",
			2, 80, "provider",
		); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}

		config.Config.Context().SaveCloudProvider(provider)
		fmt.Print("\nConfiguration input saved\n\n")
		return
	}

	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	} else {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf(
				"Unknown cloud IaaS name \"%s\" given to the configure "+
					"command.Run 'cb cloud list' to get list of available clouds.",
				name,
			),
		)
	}
}

func init() {
	flags := showCommand.Flags()
	flags.SortFlags = false
}
