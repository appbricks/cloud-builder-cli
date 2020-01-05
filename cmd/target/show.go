package target

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"

	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/cloud-builder/target"

	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var showFlags = struct {
	all bool
}{}

var showCommand = &cobra.Command{
	Use: "show [recipe] [cloud] [region] [deployment name]",

	Short: "Show configuration data for a target.",
	Long: `
Show the deployment configuration values for the target. If the
target has not been created and configured then this sub-command will
return an error. Run 'cb target list' to view the list of configured
targets.
`,

	Run: func(cmd *cobra.Command, args []string) {
		ShowRecipe(args[0], args[1], args[2], args[3])
	},
	Args: cobra.ExactArgs(4),
}

func ShowRecipe(recipe, cloud, region, deploymentName string) {

	var (
		err error

		target    *target.Target
		inputForm forms.InputForm
	)

	targetName := fmt.Sprintf("%s/%s/%s/%s", recipe, cloud, region, deploymentName)
	if target, err = config.Config.Context().GetTarget(targetName); err == nil && target != nil {

		if inputForm, err = target.Provider.InputForm(); err != nil {
			// if this happens there is an internal
			// error and it is most likely a bug
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		showInputFormData(
			fmt.Sprintf("Provider Configuration for Target \"%s\"", targetName),
			inputForm,
		)
		if inputForm, err = target.Recipe.InputForm(); err != nil {
			// if this happens there is an internal
			// error and it is most likely a bug
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		showInputFormData(
			fmt.Sprintf("Recipe Configuration for Target \"%s\"", targetName),
			inputForm,
		)
		if inputForm, err = target.Backend.InputForm(); err != nil {
			// if this happens there is an internal
			// error and it is most likely a bug
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		showInputFormData(
			fmt.Sprintf("Backend Configuration for Target \"%s\"", targetName),
			inputForm,
		)
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

func showInputFormData(title string, inputForm forms.InputForm) {

	var (
		err error

		textForm *ux.TextForm
	)

	fmt.Printf("\n")
	if textForm, err = ux.NewTextForm(
		title,
		"CONFIGURATION DATA",
		inputForm); err != nil {
		// if this happens there is an internal
		// error and it is most likely a bug
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	if showFlags.all {
		textForm.ShowInputReference(
			ux.DescAndValues,
			0, 2, 80)
	} else {
		textForm.ShowInputReference(
			ux.DescAndValues,
			0, 2, 80,
			"target-undeployed", "target-deployed",
		)
	}
	fmt.Println("\n")
}

func init() {
	flags := showCommand.Flags()
	flags.SortFlags = false
	flags.BoolVarP(&showFlags.all, "all", "a", false, "show all configuration data values")
}
