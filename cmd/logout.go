package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var logoutCommand = &cobra.Command{
	Use: "logout",

	Short: "Log out the current user.",
	Long: `
Signs out the current user in context.
`,

	Run: func(cmd *cobra.Command, args []string) {

		var (
			err error

			awsAuth *cbcli_auth.AWSCognitoJWT
		)

		config := cbcli_config.Config
		
		if config.AuthContext().IsLoggedIn() {
			if awsAuth, err = cbcli_auth.NewAWSCognitoJWT(config); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		}
		if err = config.AuthContext().Reset(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		fmt.Println()
		if awsAuth != nil {
			cbcli_utils.ShowNoteMessage("User \"%s\" has been logged out.", awsAuth.Username())
		} else {
			cbcli_utils.ShowNoteMessage("Logout complete.")
		}
		fmt.Println()
	},
}
