package recipe

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var showCommand = &cobra.Command{
	Use: "show [recipe] [cloud]",

	Short: "Show details of the given cloud recipe",
	Long: `
Show information regarding the given cloud recipe. This sub-command
will show help for the recipe inputs including defaults that can be
provided to customize the deployment of the recipe.
`,

	PreRun: cbcli_auth.AssertAuthorized,

	Run: func(cmd *cobra.Command, args []string) {
		ShowRecipe(args[0], args[1])
	},
	Args: cobra.ExactArgs(2),
}

func ShowRecipe(name, cloud string) {

	var (
		err error

		inputForm forms.InputForm
		textForm  *ux.TextForm
	)

	cookbook := cbcli_config.Config.TargetContext().Cookbook()
	recipes := cookbook.RecipeList()

	for _, r := range recipes {
		if r.Name == name {

			validCloud := false
			for _, c := range r.IaaSList {

				if cloud == c.Name() {
					recipe := cookbook.GetRecipe(r.Name, c.Name())

					if inputForm, err = recipe.InputForm(); err != nil {
						// if this happens there is an internal
						// error and it is most likely a bug
						cbcli_utils.ShowErrorAndExit(err.Error())
					}

					fmt.Printf("\n")
					if textForm, err = ux.NewTextForm(
						fmt.Sprintf("Cloud Recipe Configuration for \"%s\"", name),
						"CONFIGURATION DATA INPUT REFERENCE",
						inputForm); err != nil {
						// if this happens there is an internal
						// error and it is most likely a bug
						cbcli_utils.ShowErrorAndExit(err.Error())
					}
					textForm.ShowInputReference(ux.DescAndDefaults, 0, 2, 80)
					fmt.Print("\n\n")

					validCloud = true
				}
			}

			if validCloud {
				return
			}

			cbcli_utils.ShowErrorAndExit(
				fmt.Sprintf(
					"Unknown cloud name \"%s\" given to \"--cloud\" option. "+
						"Run 'cb recipe list' to get list of clouds the recipe "+
						"\"%s\" can be targeted to.", cloud, name,
				),
			)
		}
	}
	cbcli_utils.ShowErrorAndExit(
		fmt.Sprintf(
			"Unknown cloud recipe name \"%s\" given to the show command. "+
				"Run 'cb recipe list' to get list of available recipes.",
			name,
		),
	)
}

func init() {
	flags := showCommand.Flags()
	flags.SortFlags = false
}
