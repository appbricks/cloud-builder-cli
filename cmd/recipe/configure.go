package recipe

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"

	"github.com/appbricks/cloud-builder/cookbook"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var configureCommand = &cobra.Command{
	Use: "configure [recipe] [cloud]",

	Short: "Configure recipe parameters.",
	Long: `
Recipes are parameterized to accomodate different configurations in
the cloud. This sub-command can be used to configure a common recipe
template which can be further customized when configuring a target.
`,

	PreRun: cbcli_auth.AssertAuthorized,

	Run: func(cmd *cobra.Command, args []string) {
		ConfigureRecipe(args[0], args[1])
	},
	Args: cobra.ExactArgs(2),
}

func ConfigureRecipe(name, cloud string) {

	var (
		err error

		recipe    cookbook.Recipe
		inputForm forms.InputForm
	)

	if recipe, err = cbcli_config.Config.Context().GetCookbookRecipe(name, cloud); err == nil && recipe != nil {

		if inputForm, err = recipe.InputForm(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		if err = ux.GetFormInput(inputForm,
			fmt.Sprintf("Cloud Recipe Configuration for \"%s\"", name),
			"CONFIGURATION DATA INPUT",
			2, 80, "recipe",
		); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}

		cbcli_config.Config.Context().SaveCookbookRecipe(recipe)
		fmt.Print("\nConfiguration input saved\n\n")
		return
	}

	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	} else {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf(
				"Unknown recipe \"%s\" for cloud \"%s\" given to the configure "+
					"command.\nRun 'cb recipe list' to get list of available recipes.",
				name, cloud,
			),
		)
	}
}

func init() {
	flags := showCommand.Flags()
	flags.SortFlags = false
}
