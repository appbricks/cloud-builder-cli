package target

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/cloud-builder/target"
	"github.com/mevansam/gocloud/cloud"

	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var resumeFlags = struct {
	instance string
}{}

var resumeCommand = &cobra.Command{
	Use: "resume [recipe] [cloud] [region] [deployment name]",

	Short: "Resumes a suspended target.",
	Long: `
This sub-command resumes all instances deployed to a target. To
resume a specific instance provide the instance name via the 
'-i|--instance' option.
`,

	Run: func(cmd *cobra.Command, args []string) {
		ResumeTarget(args[0], args[1], args[2], args[3])
	},
	Args: cobra.ExactArgs(4),
}

func ResumeTarget(recipe, iaas, region, deploymentName string) {

	var (
		err error

		tgt *target.Target
	)

	targets := config.Config.Context().TargetSet()
	targetName := fmt.Sprintf("%s/%s/%s/%s", recipe, iaas, region, deploymentName)
	if tgt = targets.GetTarget(targetName); tgt != nil {

		if err = tgt.LoadRemoteRefs(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		if tgt.Status() == target.Shutdown {
			fmt.Println()
			if err = tgt.LoadRemoteRefs(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())

			} else if err = tgt.Resume(
				func(name string, instance cloud.ComputeInstance) {
					state, _ := instance.State()
					if state == cloud.StateStopped {
						fmt.Printf("Starting instance \"%s\"...", name)
					} else {
						fmt.Println("done")
					}
				},
			); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		} else {
			cbcli_utils.ShowErrorAndExit("target needs to be 'shutdown' to be resumed")
		}
		return
	}

	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	} else {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf(
				"Unknown target named \"%s\". Run 'cb target list' "+
					"to list the currently configured targets",
				targetName,
			),
		)
	}
}

func init() {
	flags := resumeCommand.Flags()
	flags.SortFlags = false
	flags.StringVarP(&resumeFlags.instance, "instance", "i", "", "name of the instance to resume")
}
