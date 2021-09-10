package target

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

var TargetCommands = &cobra.Command{
	Use: "target",

	Short: "List, show detail of running targets and configure quick launch recipes.",
	Long: `
A target is an instance of a recipe that can be launched with a 
single click to a cloud region. When a recipe is configured for a
particular cloud it will enumerate all the regions of that cloud as
quick lauch targets. The sub-commands below allow you to launch and
view the status targets.
`,
}

func init() {
	TargetCommands.AddCommand(createCommand)
	TargetCommands.AddCommand(listCommand)
	TargetCommands.AddCommand(showCommand)
	TargetCommands.AddCommand(connectCommand)
	TargetCommands.AddCommand(manageCommand)
	TargetCommands.AddCommand(configureCommand)
	TargetCommands.AddCommand(launchCommand)
	TargetCommands.AddCommand(deleteCommand)
	TargetCommands.AddCommand(suspendCommand)
	TargetCommands.AddCommand(resumeCommand)
	TargetCommands.AddCommand(sshCommand)
}

type commonFlags struct {
	region string
	space string
}

func bindCommonFlags(
	flags *pflag.FlagSet, 
	commonFlags *commonFlags,
) {
	flags.StringVarP(&commonFlags.region, "region", "r", "", 
		"space target's region")
	flags.StringVarP(&commonFlags.space, "space", "s", "", 
		"application's attached space target\n(format <recipe>/<cloud>/<region>/<name>)")	
}

func getTargetKeyFromArgs(
	recipe, iaas, deploymentName string, 
	commonFlags *commonFlags,
) string {

	var (
		targetKey string
	)
	
	if len(commonFlags.region) > 0 && len(commonFlags.space) > 0 {
		cbcli_utils.ShowErrorAndExit("Please provide only one of region or space options for target lookup.")
	}
	if len(commonFlags.region) > 0 {
		targetKey = target.CreateKey(recipe, iaas, commonFlags.region, deploymentName)
	} else if len(commonFlags.space) > 0 {
		targetKey = target.CreateKey(recipe, iaas, deploymentName, "<"+commonFlags.space)
	}
	return targetKey
}

// authorizes the user to perform a command on the selected space target
func authorizeSpaceTarget(roleMask auth.RoleMask, commonFlags *commonFlags) func(cmd *cobra.Command, args []string) {

	return func(cmd *cobra.Command, args []string) {

		var (
			targetKey string
		)
		
		if len(commonFlags.region) > 0 && len(commonFlags.space) > 0 {
			cbcli_utils.ShowErrorAndExit("Please provide only one of region or space options for target lookup.")
		}
		if len(commonFlags.region) > 0 {
			targetKey = target.CreateKey(args[0], args[1], commonFlags.region, args[2])
		} else if len(commonFlags.space) > 0 {
			targetKey = target.CreateKey(args[0],  args[1], args[2], "<"+commonFlags.space)
		}

		spaceNode = cbcli_config.SpaceNodes.LookupSpaceNode(targetKey, func(nodes []userspace.SpaceNode) userspace.SpaceNode {
			fmt.Printf("Space Nodes to select: %# v\n\n", nodes)
			return nodes[0]
		})

		cbcli_auth.AssertAuthorized(roleMask, spaceNode)(cmd, args)
	}
}