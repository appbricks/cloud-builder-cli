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
	commonFlags

	keep bool
}{}

var deleteCommand = &cobra.Command{
	Use: "delete [recipe] [cloud] [deployment name]",

	Short: "Deletes a quick launch target deployment.",
	Long: `
This sub-command will destroy cloud resources if the target has been
deployed and removes the launch configuration. If you wish to retain
the configuration in order to re-launch the target at a latter date
then provide the --keep flag.
`,

	Run: func(cmd *cobra.Command, args []string) {
		DeleteTarget(getTargetKeyFromArgs(args[0], args[1], args[2], &(showFlags.commonFlags)))
	},
	Args: cobra.ExactArgs(3),
}

func DeleteTarget(targetKey string) {

	var (
		err error

		tgt  *target.Target
		bldr *target.Builder

		response string
	)

	if tgt, err = config.Config.Context().GetTarget(targetKey); err == nil && tgt != nil {

		fmt.Println()
		fmt.Print(
			color.OpBold.Render(
				utils.FormatMessage(
					0, 80, false, true,
					"Found %s.",
					tgt.Name(),
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
				if tgt.CookbookTimestamp != tgt.Recipe.CookbookTimestamp() {
					// force re-initializing
					if err = bldr.Initialize(); err != nil {
						cbcli_utils.ShowErrorAndExit(err.Error())
					}
				}
				if err = bldr.Delete(); err != nil {
					cbcli_utils.ShowErrorAndExit(err.Error())
				}
				tgt.Output = nil
			}
			if !deleteFlags.keep {
				config.Config.Context().TargetSet().DeleteTarget(tgt.Key())
			}
			fmt.Print(color.Green.Render("\nTarget has been deleted.\n\n"))
		} else {
			fmt.Print(color.Red.Render("\nTarget has not been deleted.\n\n"))
		}
		return
	}

	cbcli_utils.ShowErrorAndExit(
		fmt.Sprintf(
			"Target \"%s\" does not exist. Run 'cb target list' to list the currently configured targets",
			targetKey,
		),
	)
}

func init() {
	flags := deleteCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(showFlags.commonFlags))

	flags.BoolVarP(&deleteFlags.keep, "keep", "k", false, "destroy deployed resources if any but do not delete the configuration")
}
