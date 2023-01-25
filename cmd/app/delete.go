package app

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var deleteCommand = &cobra.Command{
	Use: "delete [cookbook name]",

	Short: "Delete recipes",
	Long: `
Deletes all recipes within the given application cookbook. Imported
recipes can only be deleted by removing the cookbook that was
imported. To restore an older version of the cookbook and its recipes
simply import that version. The last imported cookbook will always be 
considered the current version in use by the cloud builder CLI.
`,

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		DeleteRecipe(args[0])
	},
	Args: cobra.ExactArgs(1),
}

func DeleteRecipe(cookbookName string) {

	cookbook := cbcli_config.Config.TargetContext().Cookbook()
	cm := cookbook.GetCookbook(cookbookName)
	if cm == nil {
		cbcli_utils.ShowErrorAndExit(fmt.Sprintf("Cookbook named '%s' does not exist.", cookbookName))
	}
	if !cm.Imported {
		cbcli_utils.ShowErrorAndExit("You cannot delete embedded cookbooks.")
	}

	targets := cbcli_config.Config.TargetContext().TargetSet().GetTargets()
	for _, t := range targets {
		if t.CookbookName == cookbookName {
			cbcli_utils.ShowErrorAndExit("Cookbook cannot be deleted while targets for the cookbook recipes exist.")
		}
	}

	fmt.Println()
	response := cbcli_utils.GetUserInput(
		"Confirm deletion by entering the cookbook name: ",
	)
	if response == cookbookName {
		if err := cookbook.DeleteImportedCookbook(cookbookName); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
	} else {
		cbcli_utils.ShowWarningMessage("\nName entered does not match the cookbook to be deleted.")
	}
	fmt.Println()
}

func init() {
	flags := deleteCommand.Flags()
	flags.SortFlags = false
}
