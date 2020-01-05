package target

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"
	"github.com/mevansam/goutils/term"
	"github.com/mevansam/goutils/utils"

	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/cloud-builder/target"

	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var configureFlags = struct {
	all bool
}{}

var configureCommand = &cobra.Command{
	Use: "configure [recipe] [cloud] [region] [deployment name]",

	Short: "Configure an existing target.",
	Long: `
Use this command to configure an existing target. You will need to
re-apply this changes to the deployment if the target has already
been launched.
`,

	Run: func(cmd *cobra.Command, args []string) {
		ConfigureTarget(args[0], args[1], args[2], args[3])
	},
	Args: cobra.ExactArgs(4),
}

func ConfigureTarget(recipe, cloud, region, deploymentName string) {

	var (
		err error

		tgt *target.Target
	)

	targetName := fmt.Sprintf("%s/%s/%s/%s", recipe, cloud, region, deploymentName)
	if tgt, err = config.Config.Context().GetTarget(targetName); err == nil && tgt != nil {

		if tgt.Status() == target.Undeployed {
			if configureFlags.all {
				configureTarget(tgt, "recipe", "target-undeployed")
			} else {
				configureTarget(tgt, "target-undeployed")
			}
		} else {
			if configureFlags.all {
				cbcli_utils.ShowErrorAndExit("You can re-configure all inputs for undeployed targets only.")
			}

			configureTarget(tgt, "target-deployed")
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

func configureTarget(tgt *target.Target, tags ...string) {

	var (
		err error

		recipeInputForm,
		backendInputForm forms.InputForm

		response string
	)

	targetKey := tgt.Key()

	// target form title header
	divider := strings.Repeat("+", 80)
	fmt.Println()
	fmt.Println(divider)
	fmt.Print(term.BOLD)
	fmt.Println(utils.FormatMessage(0, 80, true, false, "Configure Target %s", tgt.Description()))
	fmt.Print(term.NC)
	fmt.Println(divider)
	fmt.Println()

	// reconfigure existing target's recipe variables
	if recipeInputForm, err = tgt.Recipe.InputForm(); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	if err = ux.GetFormInput(recipeInputForm,
		fmt.Sprintf(
			"Target's Recipe \"%s\" Configuration",
			tgt.Recipe.Name(),
		),
		"CONFIGURATION DATA INPUT",
		2, 80, tags...,
	); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}

	// ensure target's recipe configuration is complete before proceeding
	if !tgt.Recipe.IsValid() {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf(
				"Configuration values for the recipe '%s' are not complete. "+
					"Run 'cb recipe configure %s %s' to ensure common recipe "+
					"configurations have been saved",
				tgt.RecipeName, tgt.RecipeName, tgt.RecipeIaas,
			),
		)
	}

	// configure the recipe backend
	if backendInputForm, err = tgt.Backend.InputForm(); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	if len(backendInputForm.EnabledInputs("target-undeployed")) > 0 {

		backend := tgt.Backend.(backend.CloudBackend)
		if !backend.IsValid() {
			if err = backend.Configure(
				tgt.Provider.(provider.CloudProvider),
				tgt.DeploymentName(), tgt.RecipeName,
			); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		}

		if err = ux.GetFormInput(backendInputForm,
			fmt.Sprintf(
				"Target's Backend \"%s\" Configuration",
				tgt.Backend.Name(),
			),
			"CONFIGURATION DATA INPUT",
			2, 80, "target-undeployed",
		); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
	}

	// save target
	if config.Config.Context().HasTarget(tgt.Key()) {

		fmt.Print(utils.FormatMessage(7, 80, false, true, tgt.Description()))
		fmt.Println(" exists.")
		response = cbcli_utils.GetUserInputFromList(
			"Do you wish to overwrite it (yes/no)? ",
			"yes",
			[]string{"no", "yes"},
		)

		if len(response) > 0 && response != "yes" {
			fmt.Printf(term.RED + "\nConfiguration for target was not saved.\n\n" + term.NC)
			return
		}
	}
	config.Config.Context().SaveTarget(targetKey, tgt)
	fmt.Printf(term.GREEN + "\nConfiguration for target saved.\n\n" + term.NC)
}

func init() {
	flags := configureCommand.Flags()
	flags.SortFlags = false
	flags.BoolVarP(&configureFlags.all, "all", "a", false, "configure all possible configuration data values")
}
