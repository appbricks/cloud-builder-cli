package initialize

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hasura/go-graphql-client"
	"github.com/peterh/liner"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/userspace"
	"github.com/appbricks/mycloudspace-client/api"
	"github.com/appbricks/mycloudspace-client/mycscloud"
	"github.com/appbricks/mycloudspace-client/system"
	"github.com/mevansam/goutils/logger"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var InitCommand = &cobra.Command{
	Use: "init",

	Short: "Initialize and register the cloud builder client.",
	Long: `
This will register or associate a cloud builder user with all CLI 
sessions. You need to register if you would like to share access to
targets or synchronize configurations across all your devices. It
will also create client specific keys for encryption of cloud
configurations. All credentials including configuration information
are encrypted using public-private key encryption. When you
initialize the CLI for first time the keys will be created and your
private key will be saved to you system's key store or local file
system. You will need to add this key to each of your devices from
which you want to interact with and control your launch targets.
`,

	Run: func(cmd *cobra.Command, args []string) {
		initialize()
	},
}

func initialize() {

	var (
		err error

		awsAuth *cbcli_auth.AWSCognitoJWT

		resetConfig,
		resetPassphrase bool

		device *userspace.Device
		// owner *userspace.User
		
		hostName,
		userID,
		userName,
		deviceIDKey,
		deviceID,
		deviceName,
		oldDeviceID,
		passphrase,
		verifyPassphrase,
		unlockTimeout string

		timeout int

		prevOwnerAPIClient, 
		newOwnerAPIClient *graphql.Client

		deviceAPI *mycscloud.DeviceAPI
	)
	config := cbcli_config.Config
	deviceContext := config.DeviceContext()

	line := liner.NewLiner()
	line.SetCtrlCAborts(true)
	defer func() {
		line.Close()
		if err := recover(); err != nil {
			if err == liner.ErrPromptAborted {
				fmt.Println("\nInitialization aborted.")
				os.Exit(1)
			} else {
				cbcli_utils.ShowErrorAndExit(err.(error).Error())
			}
		}
	}()

	fmt.Println("\nInitializing Configuration Context\n==================================")

	if config.Initialized() {
		fmt.Println()
		cbcli_utils.ShowNoteMessage("Configuration has already been intialized.")
	}

	resetConfig = true
	if deviceOwner, isSet := deviceContext.GetOwnerUserName(); isSet {
		fmt.Println()
		if resetConfig, err = cbcli_utils.GetYesNoUserInput("Do you wish to reset the primary user : ", false); err != nil {
			panic(err)
		}
		if resetConfig {
			// confirm device owner by forcing user to re-login
			fmt.Println()
			cbcli_utils.ShowDangerMessage(
				"Resetting the primary user will also reset any saved configurations. If the current " +
				"primary user has deployed cloud spaces and applications their configurations will be " +
				"lost and may not be able to be recovered. Before proceding please ensure that you have " +
				"exported the current configuration, in case you need to recover deployments associated " +
				"with the current configuration.",
			)

			if awsAuth, err = cbcli_auth.GetAuthenticatedToken(
				config, true,
				"To continue please authenticate as the current device owner whose configuration will be overwritten.",
			); err != nil {
				cbcli_utils.ShowErrorAndExit("Failed to authenticate primary user.")
			}

			// retrieve old device id which will be unregistered
			oldDeviceID = deviceContext.GetDevice().DeviceID
			// api client for current owner which will be
			// used to unregister this device with that user
			prevOwnerAPIClient = api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", cbcli_config.Config)

			// reset config context clearing all of current owner's 
			// context data in preparation for a new owner
			if err = config.Reset(); err != nil {
				cbcli_utils.ShowErrorAndExit("Failed to reset current configuration.")
			}
		} else {
			// retrieve auth token of logged in user
			if awsAuth, err = cbcli_auth.GetAuthenticatedToken(
				config, false, 
				"You need to be logged in as the device owner to continue updating the CLI init settings.",
			); err != nil {
				cbcli_utils.ShowErrorAndExit("Failed to authenticate primary user.")
			}
		}
		// ensure init updates are done only by the device owner
		if awsAuth.Username() != deviceOwner {
			cbcli_utils.ShowErrorAndExit("In order to re-initialize a configuration you need to be signed in as the current device owner.")
		}
		fmt.Println()
	}
	if resetConfig {
		cbcli_utils.ShowNoteMessage(
			"Please login as the primary user that will be configured as the owner of this device and " +
			"configuration context.",
		)
		fmt.Println()
		cbcli_utils.ShowNoteMessage(
			"Once configured this user will own all spaces and applications launched via this CB CLI " +
			"device client. Guest users may be authorized by the primary user to use this device to connect " +
			"to space resources they have access to, but they will not be able to create and administer " +
			"spaces. If you want to change the primary user of the device you need to re-initialize the CLI.",
		)

		if awsAuth, err = cbcli_auth.GetAuthenticatedToken(config, false); err != nil {
			cbcli_utils.ShowErrorAndExit("Failed to authenticate primary user.")
		}
		userID = awsAuth.UserID()
		userName = awsAuth.Username()

		fmt.Println()
		cbcli_utils.ShowNoteMessage(
			"This client CLI will be given a unique device identity with the machine it is running on. " +
			"This helps identify which clients or devices are connecting to your space along with the " +
			"users that are connecting. If you use the Cloud Space client or CLI from another machine or " +
			"device it will receive its own identity but your configuration context will remain the same. " +
			"It is recommended that you give this device a name that will help you identify the it within " +
			"your network.",
		)
		fmt.Println()

		if hostName, err = os.Hostname(); err != nil {
			panic(err)
		}
		line.SetCompleter(func(line string) []string {
			return []string{"", hostName}
		})
		if deviceName, err = line.PromptWithSuggestion(
			"What is the name of this device : ", 
			hostName, -1,
		); err != nil {
			panic(err)
		}
		line.SetCompleter(nil)

		// create device owner user
		if /*owner*/ _, err = deviceContext.NewOwnerUser(userID, userName); err != nil {
			panic(err)
		}
		// create new device
		if device, err = deviceContext.NewDevice(); err != nil {
			panic(err)
		}

		// api client for new owner which will be
		// used to register this device with that user
		newOwnerAPIClient = api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", cbcli_config.Config)
	}

	resetPassphrase = true
	if config.Initialized() {		
		if resetPassphrase, err = cbcli_utils.GetYesNoUserInput("Do you wish to reset the passphrase : ", false); err != nil {
			panic(err)
		}
	}
	if resetPassphrase {
		fmt.Println()
		if passphrase, err = line.PasswordPrompt("Enter the encryption passphrase : "); err != nil {
			panic(err)
		}
		if verifyPassphrase, err = line.PasswordPrompt("Verify the encryption passphrase : "); err != nil {
			panic(err)
		}
		if passphrase != verifyPassphrase {
			fmt.Println("\nPassphrases do not match.")
			os.Exit(1)
		}
		config.SetPassphrase(passphrase)
	}

	fmt.Println()
	line.SetCompleter(func(line string) []string {
		return []string{"12h", "1h", "30m", "10m", ""}
	})
	if unlockTimeout, err = line.PromptWithSuggestion("Enter unlock timeout (i.e. ##h(ours)/m(inutes)/(s)econds) : ", "24h", -1); err != nil {
		panic(err)
	}
	line.SetCompleter(nil)

	l := len(unlockTimeout)
	if l > 0 {
		if timeout, err = strconv.Atoi(unlockTimeout[:l-1]); err != nil {
			panic(err)
		}
		switch unlockTimeout[l-1] {
		case 'h':
			logger.DebugMessage("Setting unlock timeout to %d hours.", timeout)
			config.SetKeyTimeout(time.Duration(timeout) * time.Hour)
		case 'm':
			logger.DebugMessage("Setting unlock timeout to %d minutes.", timeout)
			config.SetKeyTimeout(time.Duration(timeout) * time.Minute)
		case 's':
			logger.DebugMessage("Setting unlock timeout to %d seconds.", timeout)
			config.SetKeyTimeout(time.Duration(timeout) * time.Second)
		default:
			fmt.Println("\nUnable to determine time format.")
			os.Exit(1)
		}
	} else {
		config.SetKeyTimeout(0)
	}

	// register device with MyCS account service with current 
	// logged in user as owner. do this last to ensure all 
	// other settings are accepted and valid before writing
	// to the backend
	if resetConfig {
		if prevOwnerAPIClient != nil {
			// unregister this device using the prev owner's API client
			deviceAPI = mycscloud.NewDeviceAPI(prevOwnerAPIClient)
			if _, err = deviceAPI.UnRegisterDevice(oldDeviceID); err != nil {
				panic(err)
			}
		}		
		// register this device using the new owner's API client
		deviceAPI = mycscloud.NewDeviceAPI(newOwnerAPIClient)
		if deviceIDKey, deviceID, err = deviceAPI.RegisterDevice(
			deviceName, 
			system.GetDeviceType(),
			system.GetDeviceVersion(cbcli_config.ClientType, cbcli_config.Version),
			"", 
			device.RSAPublicKey, 
		); err != nil {
			panic(err)
		}
		deviceContext.SetDeviceID(deviceIDKey, deviceID, deviceName)		
	}

	config.SetInitialized()
	fmt.Println()
}
