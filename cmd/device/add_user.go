package device

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/appbricks/mycloudspace-client/api"
	"github.com/appbricks/mycloudspace-client/mycscloud"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var addUserCommand = &cobra.Command{
	Use: "add-user [device name] [user name]",

	Short: "Add a user to a managed device.",
	Long: `
Adds a user to the given managed device. Users added to managed
devices are activated for the device by default. Once added a 
native connection config can be download via the 'space connect'
command.
`,

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		AddUser(args[0], args[1])
	},
	Args: cobra.ExactArgs(2),
}

func AddUser(deviceName string, userName string) {

	var (
		err error

		device *userspace.Device
		user *userspace.User

		users []*userspace.User

		response  string
		userIndex int
	)

	deviceContext := cbcli_config.Config.DeviceContext()
	if device = deviceContext.GetManagedDevice(deviceName); device == nil {
		cbcli_utils.ShowErrorAndExit("Not a valid managed device name.")
	}

	deviceUsers := make(map[string]bool)
	for _, u := range device.DeviceUsers {
		deviceUsers[u.Name] = true
	}

	apiClient := api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", cbcli_config.Config.AuthContext())
	userAPI := mycscloud.NewUserAPI(apiClient)

	if users, err = userAPI.UserSearch(userName); err != nil {
		cbcli_utils.ShowErrorAndExit("Failed to lookup user name")
	}
	numUsers := len(users)
	if numUsers == 0 {
		cbcli_utils.ShowErrorAndExit("No user name matches the given name.")
	}
	if numUsers > 1 || users[0].Name != userName {
		msg := "\nMore than one user matches the user name pattern provided."
		if numUsers == 5 {
			msg += " The top 5 results are given below."
		}
		cbcli_utils.ShowInfoMessage(msg)
		fmt.Println()

		optionList := make([]string, numUsers)
		for i, u := range users {
			optionList[i] = strconv.Itoa(i + 1)
			fmt.Printf("%s - %s\n", optionList[i], u.Name)
		}		
		fmt.Println()
		if response = cbcli_utils.GetUserInputFromList(
			"Enter # user to add or (q)uit: ",
			"", optionList, false); response == "q" {
			fmt.Println()
			return
		}
		if userIndex, err = strconv.Atoi(response); err != nil ||
			userIndex < 1 || userIndex > numUsers {
			cbcli_utils.ShowErrorAndExit("invalid entry")
		}
		user = users[userIndex-1]

	} else {
		user = users[0]
	}

	if _, exists := deviceUsers[user.Name]; exists {
		cbcli_utils.ShowInfoMessage("\nUser has already been added to the device.")

	} else {
		deviceAPI := mycscloud.NewDeviceAPI(apiClient)
		if _, _, err = deviceAPI.AddDeviceUser(device.DeviceID, user.UserID); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		cbcli_utils.ShowInfoMessage("\nSuccessfully added user to managed device.\n")	
	}
}

func init() {
	flags := addUserCommand.Flags()
	flags.SortFlags = false
}
