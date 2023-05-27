package target

import (
	"fmt"
	"strings"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"
	"github.com/mevansam/goutils/utils"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/target"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var configureFlags = struct {
	commonFlags

	all bool
}{}

var configureCommand = &cobra.Command{
	Use: "configure [deployment name]",

	Short: "Configure an existing target.",
	Long: `
Use this command to configure an existing target. You will need to
re-apply this changes to the deployment if the target has already
been launched. For application targets you will need to provide
the its space target via the --space flag.
`,

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		ConfigureTarget(getTargetKeyFromArgs(args[0], &(configureFlags.commonFlags)))
	},
	Args: cobra.ExactArgs(1),
}

func ConfigureTarget(targetKey string) {

	var (
		err error

		tgt *target.Target
	)

	if tgt, err = cbcli_config.Config.TargetContext().GetTarget(targetKey); err == nil && tgt != nil {

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

	cbcli_utils.ShowErrorAndExit(
		fmt.Sprintf(
			"Target \"%s\" does not exist. Run 'cb target list' to list the currently configured targets",
			targetKey,
		),
	)
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
	fmt.Println()
	fmt.Println(
		color.OpBold.Render(
			utils.FormatMessage(
				0, 80, true, false,
				"Configure Target %s",
				tgt.Name(),
			),
		),
	)
	fmt.Println()
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
	if tgt.Backend != nil {		
		if backendInputForm, err = tgt.Backend.InputForm(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		if len(backendInputForm.EnabledInputs(true, "target-undeployed")) > 0 {

			storagePrefix := strings.Join(
				append([]string{ tgt.DeploymentName() }, tgt.DependentTargets...), "-")

			if !tgt.Backend.IsValid() {
				if err = tgt.Backend.Configure(
					tgt.Provider,
					storagePrefix,
					tgt.RecipeName,
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
	}

	// save target
	if cbcli_config.Config.TargetContext().HasTarget(tgt.Key()) {

		fmt.Print(utils.FormatMessage(7, 80, false, true, tgt.Name()))
		fmt.Println(" exists.")
		response = cbcli_utils.GetUserInputFromList(
			"Do you wish to overwrite it (yes/no)? ",
			"yes",
			[]string{"no", "yes"},
			true,
		)

		if len(response) > 0 && response != "yes" {
			fmt.Print(color.Red.Render("\nConfiguration for target was not saved.\n\n"))
			return
		}
	}
	cbcli_config.Config.TargetContext().SaveTarget(targetKey, tgt)
	fmt.Print(color.Green.Render("\nConfiguration for target saved.\n\n"))
}

func init() {
	flags := configureCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(configureFlags.commonFlags))

	flags.BoolVarP(&configureFlags.all, "all", "a", false, 
		"configure all possible configuration data values")
}
