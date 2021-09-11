package space

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/cloud-builder/userspace"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var spaceNode userspace.SpaceNode

var SpaceCommands = &cobra.Command{
	Use: "space",

	Short: "List, view, manage and connect to shared spaces.",
	Long: `
Use the space sub-commands to list, view, manage amd connect to share
spaces. Shared spaces can be targets you have launched or targets
launched by another user and shared with you.`,
}

func init() {
	SpaceCommands.AddCommand(listCommand)
	SpaceCommands.AddCommand(connectCommand)
	SpaceCommands.AddCommand(manageCommand)
}

type commonFlags struct {
	region string
}

func bindCommonFlags(
	flags *pflag.FlagSet, 
	commonFlags *commonFlags,
) {
	flags.StringVarP(&commonFlags.region, "region", "r", "", 
		"space target's region")
}

// authorizes the user to perform a command on the selected space target
func authorizeSpaceNode(roleMask auth.RoleMask, commonFlags *commonFlags) func(cmd *cobra.Command, args []string) {

	return func(cmd *cobra.Command, args []string) {

		var (
			targetKey string
		)
		
		if len(commonFlags.region) == 0 {
			cbcli_utils.ShowErrorAndExit("Please provide the region option for space lookup.")
		}
		targetKey = target.CreateKey(args[0], args[1], commonFlags.region, args[2])

		spaceNode = cbcli_config.SpaceNodes.LookupSpaceNode(targetKey, func(nodes []userspace.SpaceNode) userspace.SpaceNode {
			fmt.Printf("Space Nodes to select: %# v\n\n", nodes)
			return nodes[0]
		})
		if spaceNode == nil {
			cbcli_utils.ShowErrorAndExit(
				fmt.Sprintf(
					"Space target \"%s\" does not exist. Run 'cb target list' to list the shared or owned space targets",
					targetKey,
				),
			)
		}

		cbcli_auth.AssertAuthorized(roleMask, spaceNode)(cmd, args)
	}
}