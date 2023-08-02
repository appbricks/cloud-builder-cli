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

var addCommand = &cobra.Command{
	Use: "add [device name] [device type]",

	Short: "Add a managed device.",
	Long: `
Add a managed device associated with the current device. A managed
device is used to identify devices that do not run MyCS natively, but
can still connect to the private space network via a native VPN
client such wireguard. Such devices have the same access as the
primary device they are associated with.
`,

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		AddDevice(args[0], args[1])
	},
	Args: cobra.ExactArgs(2),
}

func AddDevice(deviceName string, deviceType string) {

	var (
		err    error
		exists bool
		primaryDeviceID string

		device *userspace.Device
	)

	apiClient := api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", cbcli_config.Config.AuthContext())
	deviceAPI := mycscloud.NewDeviceAPI(apiClient)

	deviceContext := cbcli_config.Config.DeviceContext()
	if primaryDeviceID, exists = deviceContext.GetDeviceID(); !exists {
		cbcli_utils.ShowErrorAndExit("The primary device for this client has not been configured.")
	}

	if device, err = deviceContext.NewManagedDevice(); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	device.Name = deviceName
	device.Type = deviceType

	if _, device.DeviceID, err = deviceAPI.RegisterDevice(
		device.Name, 
		device.Type,
		"wireguard (managed)",
		"", 
		device.RSAPublicKey, 
		primaryDeviceID,
	); err != nil {
		cbcli_utils.ShowErrorAndExit("Failed to register new managed device.")
	}

	cbcli_utils.ShowInfoMessage("\nSuccessfully registered managed device.\n")
}

func init() {
	flags := addCommand.Flags()
	flags.SortFlags = false
}
