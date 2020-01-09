package target

import (
	"fmt"
	"os"

	"github.com/mevansam/goutils/term"
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

	Short: "Delete a launch target.",
	Long: `
Deletes a quick launch target configuration. This sub-command will
destroy cloud resources if the target has been deployed and remove
the launch configuration.
`,

	Run: func(cmd *cobra.Command, args []string) {
		DeleteTarget(args[0], args[1], args[2], args[3])
	},
	Args: cobra.ExactArgs(4),
}

func DeleteTarget(recipe, cloud, region, deploymentName string) {

	var (
		err error

		tgt  *target.Target
		bldr *target.Builder

		response string
	)

	targets := config.Config.Context().TargetSet()
	targetName := fmt.Sprintf("%s/%s/%s/%s", recipe, cloud, region, deploymentName)
	if tgt = targets.GetTarget(targetName); tgt != nil {

		fmt.Println(term.BOLD)
		fmt.Print(utils.FormatMessage(
			0, 80, false, true,
			"Found %s.",
			tgt.Description(),
		))
		fmt.Println(term.NC)
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
			fmt.Printf(term.GREEN + "\nTarget has been deleted.\n\n" + term.NC)
		} else {
			fmt.Printf(term.RED + "\nTarget has not been deleted.\n\n" + term.NC)
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
