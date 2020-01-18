package target

import (
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/mevansam/goutils/utils"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/cloud-builder/target"

	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var deleteFlags = struct {
	keep bool
}{}

var deleteCommand = &cobra.Command{
	Use: "delete [recipe] [cloud] [region] [deployment name]",

	Short: "Deletes a quick launch target deployment.",
	Long: `
This sub-command will destroy cloud resources if the target has been
deployed and removes the launch configuration. If you wish to retain
the configuration in order to re-launch the target at a latter date
then provide the --keep flag.
`,

	Run: func(cmd *cobra.Command, args []string) {
		DeleteTarget(args[0], args[1], args[2], args[3])
	},
	Args: cobra.ExactArgs(4),
}

func DeleteTarget(recipe, iaas, region, deploymentName string) {

	var (
		err error

		tgt  *target.Target
		bldr *target.Builder

		response string
	)

	targets := config.Config.Context().TargetSet()
	targetName := fmt.Sprintf("%s/%s/%s/%s", recipe, iaas, region, deploymentName)
	if tgt = targets.GetTarget(targetName); tgt != nil {

		fmt.Println()
		fmt.Print(
			color.OpBold.Render(
				utils.FormatMessage(
					0, 80, false, true,
					"Found %s.",
					tgt.Description(),
				),
			),
		)
		fmt.Println()
		response = cbcli_utils.GetUserInput(
			"Confirm deletion by entering the deployment name: ",
		)

		if response == tgt.DeploymentName() {
			if tgt.Status() != target.Undeployed {
				if bldr, err = tgt.NewBuilder(os.Stdout, os.Stderr); err != nil {
					cbcli_utils.ShowErrorAndExit(err.Error())
				}
				if err = bldr.Delete(); err != nil {
					cbcli_utils.ShowErrorAndExit(err.Error())
				}
				tgt.Output = nil
			}
			if !deleteFlags.keep {
				targets.DeleteTarget(tgt.Key())
			}
			fmt.Print(color.Green.Render("\nTarget has been deleted.\n\n"))
		} else {
			fmt.Print(color.Red.Render("\nTarget has not been deleted.\n\n"))
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
	flags := deleteCommand.Flags()
	flags.SortFlags = false
	flags.BoolVarP(&deleteFlags.keep, "keep", "k", false, "destroy deployed resources if any but do not delete the configuration")
}
