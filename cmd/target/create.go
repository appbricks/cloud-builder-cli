package target

import (
	"fmt"
	"os"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/mycloudspace-client/api"
	"github.com/appbricks/mycloudspace-client/mycscloud"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"
	"github.com/mevansam/goutils/logger"
	"github.com/spf13/cobra"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var createFlags = struct {
	dependentTarget string
}{}

var createCommand = &cobra.Command{
	Use: "create [recipe] [cloud]",

	Short: "Create a launch target.",
	Long: `
A launch target is a configured recipe instance for a particular
cloud. Use this sub-command to create a named target by associating a
configured recipe template with a configured cloud template.
`,

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		CreateTarget(args[0], args[1])
	},
	Args: cobra.ExactArgs(2),
}

func CreateTarget(recipeKey, iaasName string) {

	var (
		err error

		tgt, spaceTgt *target.Target
		spaceTgtKey string

		copySpaceTgtProvider bool
		spaceTgtProvider     config.Configurable

		recipeInputForm,
		providerInputForm forms.InputForm

		region      *string
		regionField *forms.InputField
	)
	config := cbcli_config.Config
	context := config.TargetContext()

	if tgt, err = context.NewTarget(
		recipeKey, iaasName,
	); err == nil && tgt != nil {

		if _, err = tgt.UpdateKeys(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}

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

		configureProvider := providerInputForm != nil

		// non-bastion nodes need to be associated with a space
		// node that provides the space networking services
		if !tgt.Recipe.IsBastion() {
			targets := context.TargetSet()

			if len(createFlags.dependentTarget) == 0 {
				fmt.Println()

				spaceTargets := []string{}
				for _, t := range targets.GetTargets() {
					if t.Recipe.IsBastion() {
						spaceTargets = append(spaceTargets, t.Key())
					}
				}
				if len(spaceTargets) == 0 {
					cbcli_utils.ShowInfoMessage( 
						"No space targets have been configured where application '%s' can be deployed to.\n", 
						recipeKey,
					)
					os.Exit(0)
				}
				spaceTgtKey = cbcli_utils.GetUserInputFromList(
					"User <TAB> to scroll through and select from the list of target spaces to deploy application to: ",
					spaceTargets[0],
					spaceTargets,
					true,
				)
			} else {
				spaceTgtKey = createFlags.dependentTarget
			}
			if spaceTgt = targets.GetTarget(spaceTgtKey); spaceTgt == nil || !spaceTgt.Recipe.IsBastion() {
				cbcli_utils.ShowErrorAndExit(fmt.Sprintf("Invalid space target '%s'.", spaceTgtKey))
			}
			tgt.DependentTargets = []string{spaceTgtKey}
			
			if iaasName == spaceTgt.Provider.Name() {
				// if iaas' match then the application target can reuse the
				// same provider configuration as that of the space to which 
				// it will be deployed to
				copySpaceTgtProvider = cbcli_utils.GetYesNoUserInput(
					"Do you wish deploy to the same cloud environment as the space node : ", 
					true,
				)
				if copySpaceTgtProvider {
					if spaceTgtProvider, err = spaceTgt.Provider.Copy(); err != nil {
						cbcli_utils.ShowErrorAndExit(err.Error())
					}
					tgt.Provider = spaceTgtProvider.(provider.CloudProvider)
				}
				configureProvider = !copySpaceTgtProvider
			}
		}

		// configure the target recipe's provider if required
		if configureProvider {
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
		}
		// set the target's recipe region variable
		// to be same value as that of the provider
		if region = tgt.Provider.Region(); region != nil {
			if regionField, err = recipeInputForm.GetInputField("region"); err == nil {
				logger.TraceMessage(
					"Setting the recipe '%s' region value to: %s",
					tgt.RecipeName, *region,
				)
				if err = regionField.SetValue(region);err != nil {
					cbcli_utils.ShowErrorAndExit(err.Error())
				}
			} else {
				logger.TraceMessage(
					"Recipe '%s' does not have a region input.",
					tgt.RecipeName,
				)
			}
		} else {
			logger.TraceMessage(
				"Provider '%s' does not have a region value.",
				tgt.RecipeIaas,
			)
		}

		configureTarget(tgt, "target-undeployed")

		// add target to MyCS account
		if tgt.Recipe.IsBastion() {
			// only recipes with a bastion instance is considered
			// a space. TBD: this criteria should be revisited
			spaceAPI := mycscloud.NewSpaceAPI(api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", config))
			if err = spaceAPI.AddSpace(tgt, true); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}	

		} else {			
			appAPI := mycscloud.NewAppAPI(api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", config))
			if err = appAPI.AddApp(tgt, spaceTgt.GetSpaceID()); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}	
		}
		return
	}

	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	} else {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf(
				"Unknown recipe \"%s\" for cloud \"%s\" given to the configure "+
					"command. Run 'cb recipe list' to get list of available recipes.",
				recipeKey, iaasName,
			),
		)
	}
}

func init() {
	flags := createCommand.Flags()
	flags.SortFlags = false
	flags.StringVarP(&createFlags.dependentTarget, "space", "s", "", 
		"space target key to deploy application to (format <recipe>/<cloud>/<region>/<name>)")
}
