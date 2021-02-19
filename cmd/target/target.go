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
quick lauch targets. The sub-commands below allow you to launch and
view the status targets.
`,
}

func init() {
	TargetCommands.AddCommand(listCommand)
	TargetCommands.AddCommand(showCommand)
	TargetCommands.AddCommand(createCommand)
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
	
	if len(showFlags.region) > 0 && len(showFlags.space) > 0 {
		cbcli_utils.ShowErrorAndExit("Please provide only one of region or space options for target lookup.")
	}
	if len(showFlags.region) > 0 {
		targetKey = target.CreateKey(recipe, iaas, showFlags.region, deploymentName)
	} else if len(showFlags.space) > 0 {
		targetKey = target.CreateKey(recipe, iaas, deploymentName, "<"+showFlags.space)
	}
	return targetKey
}