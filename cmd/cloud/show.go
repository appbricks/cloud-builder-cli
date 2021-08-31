package cloud

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"

	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/cloud-builder/auth"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var showCommand = &cobra.Command{
	Use: "show [name]",

	Short: "Show information regarding a particular cloud.",
	Long: `
Show detailed information regarding the given cloud. This sub-command 
will also show help for the configuration data required for the given 
cloud.
`,

	Run: func(cmd *cobra.Command, args []string) {
		cbcli_utils.AssertAuthorized(cmd,
			auth.NewRoleMask(auth.Admin).LoggedInUserHasRole(cbcli_config.Config.DeviceContext()))
		
		ShowCloud(args[0])
	},
	Args: cobra.ExactArgs(1),
}

func ShowCloud(name string) {

	var (
		err error

		inputForm forms.InputForm
		textForm  *ux.TextForm
	)

	if provider, err := config.Config.Context().GetCloudProvider(name); err == nil && provider != nil {

		if inputForm, err = provider.InputForm(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}

		fmt.Printf("\n")
		if textForm, err = ux.NewTextForm(
			"Cloud Provider Configuration",
			"CONFIGURATION DATA INPUT REFERENCE",
			inputForm); err != nil {
			// if this happens there is an internal
			// error and it is most likely a bug
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		textForm.ShowInputReference(ux.DescAndDefaults, 0, 2, 80)
		fmt.Print("\n\n")
		return
	}

	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	} else {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf(
				"Unknown cloud IaaS name \"%s\" given to the show command. "+
					"Run 'cb cloud list' to get list of available clouds.",
				name,
			),
		)
	}
}

func init() {
	flags := showCommand.Flags()
	flags.SortFlags = false
}
