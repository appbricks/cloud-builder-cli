package target

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/appbricks/cloud-builder/target"

	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var TargetCommands = &cobra.Command{
	Use: "target",

	Short: "List, show detail of running targets and configure quick launch recipes.",
	Long: `
A target is an instance of a recipe that can be launched with a 
single click to a cloud region. When a recipe is configured for a
particular cloud it will enumerate all the regions of that cloud as
quick lauch targets. The sub-commands below allow you to launch, view
amd manage these target spaces.
`,
}

func init() {
	TargetCommands.AddCommand(createCommand)
	TargetCommands.AddCommand(listCommand)
	TargetCommands.AddCommand(showCommand)
	TargetCommands.AddCommand(configureCommand)
	TargetCommands.AddCommand(launchCommand)
	TargetCommands.AddCommand(deleteCommand)
	TargetCommands.AddCommand(suspendCommand)
	TargetCommands.AddCommand(resumeCommand)
	TargetCommands.AddCommand(connectCommand)
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
