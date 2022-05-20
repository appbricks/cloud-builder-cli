package device

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/appbricks/mycloudspace-client/api"
	"github.com/appbricks/mycloudspace-client/mycscloud"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var deleteCommand = &cobra.Command{
	Use: "delete [name]",

	Short: "Delete a managed device.",
	Long: `
Delete a managed device associated with the current primary device.
`,

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		DeleteDevice(args[0])
	},
	Args: cobra.ExactArgs(1),
}

func DeleteDevice(deviceName string) {

	var (
		err error

		device *userspace.Device
	)

	deviceContext := cbcli_config.Config.DeviceContext()
	if device = deviceContext.GetManagedDevice(deviceName); device == nil {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf("No managed device with name \"%s\" found.", deviceName),
		)
	}

	apiClient := api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", cbcli_config.Config)
	deviceAPI := mycscloud.NewDeviceAPI(apiClient)
	if _, err = deviceAPI.UnRegisterDevice(device.DeviceID); err != nil {
		cbcli_utils.ShowErrorAndExit("Failed to unregister managed device.")
	}
	deviceContext.DeleteManageDevice(device.DeviceID)

	cbcli_utils.ShowInfoMessage("\nSuccessfully deleted managed device.\n")
}

func init() {
	flags := deleteCommand.Flags()
	flags.SortFlags = false
}
