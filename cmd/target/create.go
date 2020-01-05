package target

import (
	"fmt"

	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"
	"github.com/mevansam/goutils/logger"
	"github.com/spf13/cobra"
	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/cloud-builder/target"

	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var createCommand = &cobra.Command{
	Use: "create [recipe] [cloud]",

	Short: "Create a launch target.",
	Long: `
A launch target is a configured recipe instance for a particular
cloud. Use this sub-command to create a named target by associating a
configured recipe template with a configured cloud template.
`,

	Run: func(cmd *cobra.Command, args []string) {
		CreateTarget(args[0], args[1])
	},
	Args: cobra.ExactArgs(2),
}

func CreateTarget(recipeName, iaasName string) {

	var (
		err error

		tgt *target.Target

		recipeInputForm,
		providerInputForm forms.InputForm

		region      *string
		regionField *forms.InputField
	)

	if tgt, err = config.Config.Context().NewTarget(
		recipeName, iaasName,
	); err == nil && tgt != nil {

		if !tgt.Provider.IsValid() {
			cbcli_utils.ShowErrorAndExit(
				fmt.Sprintf(
					"Credentials for the '%s' cloud provider have not been configured. "+
						"Run 'cb cloud configure %s' to configure the cloud provider.",
					iaasName, iaasName,
				),
			)
		}

		if providerInputForm, err = tgt.Provider.InputForm(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		if recipeInputForm, err = tgt.Recipe.InputForm(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}

		// new target so provider configuration needs to be completed
		if err = ux.GetFormInput(providerInputForm,
			fmt.Sprintf(
				"Configure Cloud Provider \"%s\" for New Target",
				tgt.RecipeIaas,
			),
			"CONFIGURATION DATA INPUT",
			2, 80, "target-undeployed",
		); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}

		// set the target's recipe region variable
		// to be same value as that of the provider
		if region = tgt.Provider.Region(); region != nil {
			if regionField, err = recipeInputForm.GetInputField("region"); err == nil {
				logger.TraceMessage(
					"Setting the recipe '%s' region value to: %s",
					tgt.RecipeName, *region,
				)
				err = regionField.SetValue(region)
			} else {
				logger.TraceMessage(
					"Recipe '%s' does not have a region input.",
					tgt.RecipeName,
				)
			}
			if err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		} else {
			logger.TraceMessage(
				"Provider '%s' does not have a region value.",
				tgt.RecipeIaas,
			)
		}

		configureTarget(tgt, "target-undeployed")
		return
	}

	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	} else {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf(
				"Unknown recipe \"%s\" for cloud \"%s\" given to the configure "+
					"command. Run 'cb recipe list' to get list of available recipes.",
				recipeName, iaasName,
			),
		)
	}
}

func init() {
	flags := showCommand.Flags()
	flags.SortFlags = false
}
