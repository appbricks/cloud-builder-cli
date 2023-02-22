package target

import (
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/mycloudspace-client/api"
	"github.com/appbricks/mycloudspace-client/mycscloud"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var deleteFlags = struct {
	commonFlags

	keep  bool
	force bool
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

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		DeleteTarget(getTargetKeyFromArgs(args[0], args[1], args[2], &(deleteFlags.commonFlags)))
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
	config := cbcli_config.Config
	context := config.TargetContext()

	if tgt, err = context.GetTarget(targetKey); err == nil && tgt != nil {

		if tgt.HasDependents() {
			cbcli_utils.ShowErrorAndExit(
				fmt.Sprintf(
					"Target '%s' has dependent targets. Please delete all dependent targets before deleting this target.",
					tgt.DeploymentName(),
				),
			)
		}

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
			if deleteFlags.force || tgt.Status() != target.Undeployed {
				if bldr, err = tgt.NewBuilder(config.ContextVars(), os.Stdout, os.Stderr); err != nil {
					cbcli_utils.ShowErrorAndExit(err.Error())
				}
				if tgt.CookbookVersion != tgt.Recipe.CookbookVersion() {
					// force re-initializing
					if err = bldr.Initialize(); err != nil {
						cbcli_utils.ShowErrorAndExit(err.Error())
					}
				} else {
					// initialize if required
					if err = bldr.AutoInitialize(); err != nil {
						cbcli_utils.ShowErrorAndExit(err.Error())
					}
				}
				if err = bldr.Delete(); err != nil {
					cbcli_utils.ShowErrorAndExit(err.Error())
				}
				tgt.Output = nil
				context.SaveTarget(tgt.Key(), tgt)
			}
			if !deleteFlags.keep {
				context.DeleteTarget(tgt.Key())

				// delete target from MyCS account
				if tgt.Recipe.IsBastion() {
					// only recipes with a bastion instance is considered
					// a space. TBD: this criteria should be revisited
					spaceAPI := mycscloud.NewSpaceAPI(api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", config))
					if _, err = spaceAPI.DeleteSpace(tgt); err != nil {
						logger.ErrorMessage("DeleteTarget(): Error attempting to delete space registration: %s", err.Error())
						cbcli_utils.ShowNoteMessage("\nDeleting space registration failed. You may need to manually delete the space from the MyCS cloud dashboard.")
					}

				} else {
					appAPI := mycscloud.NewAppAPI(api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", config))
					if _, err = appAPI.DeleteApp(tgt); err != nil {
						logger.ErrorMessage("DeleteTarget(): Error attempting to delete app registration: %s", err.Error())
						cbcli_utils.ShowNoteMessage("\nDeleting app registration failed. You may need to manually delete the app from the MyCS cloud dashboard.")
					}
				}
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
	bindCommonFlags(flags, &(deleteFlags.commonFlags))

	flags.BoolVarP(&deleteFlags.keep, "keep", "k", false, "destroy deployed resources if any but do not delete the configuration")
	flags.BoolVarP(&deleteFlags.force, "force", "f", false, "run delete on target even if status is undeployed")
}
