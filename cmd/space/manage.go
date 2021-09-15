package space

import (
	"fmt"
	"strconv"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/appbricks/mycloudspace-client/mycsnode"
	"github.com/mevansam/termtables"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var manageFlags = struct {
	commonFlags

	action string
	user   string
	device string
}{}

type manageActionArgs struct {
	apiClient *mycsnode.ApiClient
	user      *userspace.SpaceUser	

	enabledDeviceNames  *[]string
	disabledDeviceNames *[]string
}

type userSelectorArgs struct {
	user *userspace.SpaceUser

	enabledDeviceNames  []string
	disabledDeviceNames []string

	filterKey string
}

var userSelector = cbcli_utils.OptionSelector{
	Options: []cbcli_utils.Option{
		{
			Text: " - Enable Admin Access",
			Command: func(data interface{}) error {
				args := data.(*manageActionArgs)
				return enableAdminAccess(args.apiClient, args.user, true)
			},
		},
		{
			Text: " - Disable Admin Access",
			Command: func(data interface{}) error {
				args := data.(*manageActionArgs)
				return enableAdminAccess(args.apiClient, args.user, false)
			},
		},
		{
			Text: " - Enable Device Access to Space",
			Command: func(data interface{}) error {
				args := data.(*manageActionArgs)
				deviceName := selectDeviceFromList("enable", args.disabledDeviceNames)
				return enableDeviceAccess(args.apiClient, args.user, deviceName, true)
			},
		},
		{
			Text: " - Disable Device Access to Space",
			Command: func(data interface{}) error {
				args := data.(*manageActionArgs)
				deviceName := selectDeviceFromList("enable", args.enabledDeviceNames)
				return enableDeviceAccess(args.apiClient, args.user, deviceName, false)
			},
		},
	},
	OptionListFilter: map[string][]int{
		"userAdmin_devices":         {1, 2, 3},
		"userAdmin_enabledDevices":  {1, 3},
		"userAdmin_disabledDevices": {1, 2},
		"userGuest_devices":         {0, 2, 3},
		"userGuest_enabledDevices":  {0, 3},
		"userGuest_disabledDevices": {0, 2},
	},
	OptionRoleFilter:  map[auth.Role]map[int]bool{
		auth.Manager: {
			0: true, 1: true, 2: true, 3: true,
		},
	},
}

var manageCommand = &cobra.Command{
	Use: "manage [recipe] [cloud] [deployment name]",

	Short: "Manages a quick launch target deployment.",
	Long: `
This sub-command can be used to manage users' access to a space
target space. Once a user has accepted an invite to a space that user
will need to be enabled as either an admin or guest for the space
before they can connect to it. Any user that is an admin of the space
can enable users for that space. All space owners are default admins
of their space by default. Users can be permanently removed from the
authorized list via the MyCS Account Manager console.

The following manage actions (flag -a/--action) are supported:
- enableAdmin: grant user admin access to the space
- disableAdmin: disable admin access
- enableDevice: allow a user's device to connect to the space
- disableDevice: disable a user's device
`,

	PreRun: authorizeSpaceNode(auth.NewRoleMask(auth.Admin, auth.Manager), &(manageFlags.commonFlags)),

	Run: func(cmd *cobra.Command, args []string) {
		ManageSpace(spaceNode)
	},
	Args: cobra.ExactArgs(3),
}

func ManageSpace(space userspace.SpaceNode) {
	
	var (
		err error

		apiClient       *mycsnode.ApiClient
		isAuthenticated bool

		users []*userspace.SpaceUser

		response     string
		userSelected int
	)

	if space.GetStatus() != "running" {
		cbcli_utils.ShowErrorAndExit("Space target node needs to be in a running state in order manage it.")
	}
	if apiClient, err = mycsnode.NewApiClient(cbcli_config.Config.DeviceContext(), space); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	if isAuthenticated, err = apiClient.Authenticate(); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	if !isAuthenticated {
		cbcli_utils.ShowErrorAndExit("Authenticate with space target failed.")
	}
	if users, err = apiClient.GetSpaceUsers(); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}

	if len(manageFlags.action) == 0 {
		// if a manage action is not provided list details of
		// all space users that can connect to the target space
		userIndex := 0
		argsList := []userSelectorArgs{}
		userDevicesTable := buildUserDevicesTable(users, &userIndex, &argsList)
		if userIndex > 0 {
			fmt.Println("\nYou can manage the following users and devices associated with the\nspace target.")
			fmt.Println(color.OpBold.Render("\nSpace Users\n===========\n"))
			fmt.Println(userDevicesTable.Render())
		} else {
			cbcli_utils.ShowInfoMessage("No space users found...")
		}

		if userIndex > 0 {

			optionList := make([]string, userIndex)
			for i := 0; i < userIndex; i++ {
				optionList[i] = strconv.Itoa(i + 1)
			}
			if response = cbcli_utils.GetUserInputFromList(
				"Enter # of the user to manage or (q)uit: ",
				"", optionList, false); response == "q" {
				fmt.Println()
				return
			}
			if userSelected, err = strconv.Atoi(response); err != nil ||
				userSelected < 1 || userSelected > userIndex {
				cbcli_utils.ShowErrorAndExit("invalid entry")
			}
			userSelected--
			args := argsList[userSelected]

			fmt.Print("\nSelect manage action on space user.\n\n")
			if err = userSelector.SelectOption(
				&manageActionArgs{
					apiClient:           apiClient,
					user:                args.user,					
					enabledDeviceNames:  &args.enabledDeviceNames,
					disabledDeviceNames: &args.disabledDeviceNames,
				},
				args.filterKey,
				auth.Manager,
			); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		}

	} else {
		var lookupUser = func() *userspace.SpaceUser {
			if len(manageFlags.user) == 0 {
				cbcli_utils.ShowErrorAndExit("You need to provide the name of the space user to manage.")
			}
			for _, u := range users {
				if manageFlags.user == u.Name {
					return u
				}
			}
			cbcli_utils.ShowErrorAndExit(
				fmt.Sprintf("A space user with the name '%s' not found.", manageFlags.user),
			)
			return nil
		}
		var deviceNameRequired = func() {
			if len(manageFlags.device) == 0 {
				cbcli_utils.ShowErrorAndExit("You need to provide the name of the user's device to manage.")
			}
		}

		user := lookupUser()
		switch manageFlags.action {
			case "enableAdmin": {
				err = enableAdminAccess(apiClient, user, true)
			}
			case "disableAdmin": {
				err = enableAdminAccess(apiClient, user, false)
			}
			case "enableDevice": {
				deviceNameRequired()
				err = enableDeviceAccess(apiClient, user, manageFlags.device, true)
			}
			case "disableDevice": {
				deviceNameRequired()
				err = enableDeviceAccess(apiClient, user, manageFlags.device, false)
			}
			default: {
				cbcli_utils.ShowErrorAndExit("Invalid manage action.")
			}
		}
		if err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}

		fmt.Println()
	}
}

func buildUserDevicesTable(
	users []*userspace.SpaceUser,
	userIndex *int,
	argsList *[]userSelectorArgs,
) *termtables.Table  {

	var (
		index, 
		userName, 
		spaceAdmin, 
		deviceName,
		enabled string
	)

	table := termtables.CreateTable()
	table.AddHeaders(
		color.OpBold.Render("#"),
		color.OpBold.Render("User Name"),
		color.OpBold.Render("Space Admin"),
		color.OpBold.Render("User's Device Name"),
		color.OpBold.Render("Enabled"),
	)

	first := true
	for _, user := range users {
		if user.IsOwner {
			continue
		}
		if !first {
			table.AddSeparator()
		}

		*userIndex++
		index = strconv.Itoa(*userIndex)
		userName = user.Name
		if user.IsAdmin {
			spaceAdmin = "X"
		} else {
			spaceAdmin = ""
		}

		args := userSelectorArgs{
			user: user,

			disabledDeviceNames: []string{},
			enabledDeviceNames:  []string{},
		}

		for _, device := range user.Devices {

			if device.Enabled {
				deviceName = device.Name
				enabled = "X"
				args.enabledDeviceNames = append(args.enabledDeviceNames, device.Name)
			} else {
				deviceName = color.OpFuzzy.Render(device.Name)
				enabled = ""
				args.disabledDeviceNames = append(args.disabledDeviceNames, device.Name)
			}
			table.AddRow(
				index,
				userName,
				spaceAdmin,
				deviceName,
				enabled,
			)

			index = ""
			userName = ""
			spaceAdmin = ""
		}

		// determine manage options filter
		args.filterKey = "userGuest"
		if user.IsAdmin {
			args.filterKey = "userAdmin"
		}
		if len(args.disabledDeviceNames) == 0 {
			args.filterKey = args.filterKey + "_enabledDevices"
		} else if len(args.enabledDeviceNames) == 0 {
			args.filterKey = args.filterKey + "_disabledDevices"
		} else {
			args.filterKey = args.filterKey + "_devices"
		}		
		*argsList = append(*argsList, args)
		
		first = false
	}

	return table
}

func enableAdminAccess(
	apiClient *mycsnode.ApiClient,
	user *userspace.SpaceUser,
	enable bool,
) (err error) {

	if _, err = apiClient.UpdateSpaceUser(user.UserID, enable, !enable); err != nil {
		return err
	}
	fmt.Println("\nSpace user's configuration updated.")
	return nil
}

func enableDeviceAccess(
	apiClient *mycsnode.ApiClient,
	user *userspace.SpaceUser,
	deviceName string,
	enable bool,
) (err error) {

	for _, device := range user.Devices {
		if deviceName == device.Name {
			if _, err = apiClient.EnableUserDevice(user.UserID, device.DeviceID, enable); err != nil {
				return err
			}
			fmt.Println("\nSpace user's device configuration updated.")
			return nil
		}
	}
	return fmt.Errorf(
		"device with name '%s' not one of user %s's devices", 
		deviceName, user.Name)
}

func selectDeviceFromList(prompt string, deviceNames *[]string) string {
	
	fmt.Println()
	return cbcli_utils.GetUserInputFromList(
		fmt.Sprintf("User <TAB> to scroll through and select from list of devices to %s: ", prompt),
		"", *deviceNames, true)
}

func init() {
	flags := manageCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(manageFlags.commonFlags))

	// manage recipe iaas name -r region --action enableAdmin --user john'
	// manage recipe iaas name -r region --action disableAdmin --user john'
	// manage recipe iaas name -r region --action enableDevice --user john --device 'john's iphone'
	// manage recipe iaas name -r region --action disableDevice --user john --device 'john's iphone'

	flags.StringVarP(&manageFlags.action, "action", "a", "", "the manage action to execute")
	flags.StringVarP(&manageFlags.user, "user", "u", "", "the name of the space user to manage")
	flags.StringVarP(&manageFlags.device, "device", "d", "", "the name of the user's device to perform a device specific action")
}
