package device

import (
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/appbricks/mycloudspace-client/api"
	"github.com/appbricks/mycloudspace-client/mycscloud"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var deleteUserCommand = &cobra.Command{
	Use: "delete-user [device name] [user name]",

	Short: "Delete a user from a managed device.",
	Long: `
Deletes a user from the given managed device. Once deleted native
space connection configurations will become invalid.`,

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		DeleteUser(args[0], args[1])
	},
	Args: cobra.ExactArgs(2),
}

func DeleteUser(deviceName string, userName string) {

	var (
		err error

		device *userspace.Device
		user *userspace.User

		users []*userspace.User
	)

	deviceContext := cbcli_config.Config.DeviceContext()
	if device = deviceContext.GetManagedDevice(deviceName); device == nil {
		cbcli_utils.ShowErrorAndExit("Not a valid managed device name.")
	}
	userExists := false
	for _, u := range device.DeviceUsers {
		if u.Name == userName {
			userExists = true
			break
		}
	}
	if !userExists {
		cbcli_utils.ShowErrorAndExit("Not a valid device user.")
	}

	apiClient := api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", cbcli_config.Config)
	userAPI := mycscloud.NewUserAPI(apiClient)

	if users, err = userAPI.UserSearch(userName); err != nil {
		cbcli_utils.ShowErrorAndExit("Failed to lookup user name")
	}
	numUsers := len(users)
	if numUsers == 0 || (numUsers > 0 && users[0].Name != userName) {
		cbcli_utils.ShowErrorAndExit("Not a valid user name.")
	}
	user = users[0]

	deviceAPI := mycscloud.NewDeviceAPI(apiClient)
	if _, _, err = deviceAPI.RemoveDeviceUser(device.DeviceID, user.UserID); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	
	cbcli_utils.ShowInfoMessage("\nSuccessfully deleted user from managed device.\n")
}

func init() {
	flags := deleteUserCommand.Flags()
	flags.SortFlags = false
}
