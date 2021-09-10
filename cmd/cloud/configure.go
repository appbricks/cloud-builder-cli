package cloud

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
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

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

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

	if provider, err = cbcli_config.Config.TargetContext().GetCloudProvider(name); err == nil && provider != nil {

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

		cbcli_config.Config.TargetContext().SaveCloudProvider(provider)
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
