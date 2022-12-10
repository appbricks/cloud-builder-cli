package initialize

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hasura/go-graphql-client"
	"github.com/peterh/liner"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder-cli/auth"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/appbricks/mycloudspace-client/api"
	"github.com/appbricks/mycloudspace-client/mycscloud"
	"github.com/appbricks/mycloudspace-client/system"
	"github.com/mevansam/goutils/crypto"
	"github.com/mevansam/goutils/logger"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var deviceNameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9]$`)

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

		fi os.FileInfo

		importKey,
		resetConfig,
		resetPassphrase bool

		device *userspace.Device
		owner *userspace.User
		
		hostName,
		userID,
		userName,
		ownerKeyFile,
		ownerKeyFilePEM,
		ownerKeyFilePassphrase,		
		deviceIDKey,
		deviceID,
		deviceName,
		oldDeviceID,
		passphrase,
		verifyPassphrase,
		unlockTimeout string

		ownerKey    *crypto.RSAKey
		ownerConfig []byte
		
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

		// create device owner user
		if owner, err = deviceContext.NewOwnerUser(userID, userName); err != nil {
			panic(err)
		}
		// create new device to associate with the owner
		if device, err = deviceContext.NewDevice(); err != nil {
			panic(err)
		}

		needNewKey := awsAuth.KeyTimestamp() == 0
		if needNewKey {
			fmt.Println()
			cbcli_utils.ShowNoteMessage(
				fmt.Sprintf(
					"It appears that user '%s' is not associated with a public key. " + 
					"You will need to either import the user's key-pair or create a new one.", 
					userName,
				),
			)
			fmt.Println()
			if importKey, err = cbcli_utils.GetYesNoUserInput(
				fmt.Sprintf("Do you wish to import a private key for user '%s' : ", userName),
				false,
			); err != nil {
				panic(err)
			}

		} else {
			fmt.Println()
			cbcli_utils.ShowNoteMessage(
				fmt.Sprintf(
					"In order to continue you need to import user %s's private key. The key is " +
					"required to unlock the user's global configuration and initialize this device.", 
					userName,
				),
			)
			importKey = true
		}

		fmt.Println()
		if importKey {
			if ownerKey, err = auth.ImportPrivateKey(line); err != nil {
				panic(err)
			}

		} else {
			if ownerKeyFile, err = line.Prompt("Path to save key file (you can drag/drop from a finder/explorer window to the terminal) : "); err != nil {
				panic(err)
			}
			ownerKeyFile = strings.Trim(ownerKeyFile, " '\"")

			if fi, err = os.Stat(ownerKeyFile); err != nil {
				if os.IsNotExist(err) {
					cbcli_utils.ShowErrorAndExit(fmt.Sprintf("Path '%s' does not exist.", ownerKeyFile))
				}
				panic(err)
			}
			if !fi.IsDir() {
				cbcli_utils.ShowErrorAndExit(fmt.Sprintf("Path '%s' is not a directory.", ownerKeyFile))
			}
			ownerKeyFile = filepath.Join(ownerKeyFile, userName + "-key.pem")

			if ownerKeyFilePassphrase, err = line.PasswordPrompt("Enter the key file passphrase : "); err != nil {
				panic(err)
			}
			if verifyPassphrase, err = line.PasswordPrompt("Verify the key file passphrase : "); err != nil {
				panic(err)
			}
			if ownerKeyFilePassphrase != verifyPassphrase {
				fmt.Println("\nPassphrases do not match.")
				os.Exit(1)
			}
			if ownerKey, err = crypto.NewRSAKey(); err != nil {
				panic(err)
			}
			if ownerKeyFilePEM, err = ownerKey.GetEncryptedPrivateKeyPEM([]byte(ownerKeyFilePassphrase)); err != nil {
				panic(err)
			}
			if err = os.WriteFile(ownerKeyFile, []byte(ownerKeyFilePEM), 0600); err != nil {
				panic(err)
			}

			fmt.Println()
			cbcli_utils.ShowNoteMessage(
				"A new RSA private key has been generated and saved to the following path:", 
			)
			cbcli_utils.ShowNoteMessage(
				fmt.Sprintf(
					"- %s", 
					ownerKeyFile,
				),
			)
			fmt.Println()
			cbcli_utils.ShowNoteMessage(
				fmt.Sprintf(
					"This key will be used to secure all data associated with the user '%s' as well as " +
					"establishing a verifiable identity. It is important that this key is saved in a " +
					"secure location offline such as a USB key which can be locked away in a safe.", 
					userName,
				),
			)
		}

		// api client for new owner used owner's 
		// user public key and user config
		newOwnerAPIClient = api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", cbcli_config.Config)
		userAPI := mycscloud.NewUserAPI(newOwnerAPIClient)

		if needNewKey {
			// save new key
			if err = owner.SetKey(ownerKey); err != nil {
				panic(err)
			}
			if err = userAPI.UpdateUserKey(owner); err != nil {
				panic(err)
			}

		} else {
			// validate known public key with provided private 
			// key by encrypting some data with the known public 
			// key and decrypting with the provided private key
			if _, err = userAPI.GetUser(owner); err != nil {
				panic(err)
			}
			// save imported key
			if err = owner.SetKey(ownerKey); err != nil {
				cbcli_utils.ShowErrorAndExit("Failed to validate provided private key with user's known public key.")
			}
			// load saved configuration
			if ownerConfig, err = userAPI.GetUserConfig(owner); err != nil {
				panic(err)
			}	
		}

		// if no config exists (for example if user did not have a 
		// public key then it is assumed no encrypted config would 
		// exist) then save the current default else load the config
		targetContext := config.TargetContext()
		if ownerConfig == nil {
			var configTimestamp int64
			
			defaultConfig := new(bytes.Buffer)
			if err = targetContext.Save(defaultConfig); err != nil {
				panic(err)
			}
			if configTimestamp, err = userAPI.UpdateUserConfig(owner, defaultConfig.Bytes(), 0); err != nil {
				panic(err)
			}
			config.SetConfigAsOf(configTimestamp)

		} else {
			if err = targetContext.Load(bytes.NewReader(ownerConfig)); err != nil {
				panic(err)
			}
			config.SetConfigAsOf(awsAuth.ConfigTimestamp())
		}

		fmt.Println()
		cbcli_utils.ShowNoteMessage(
			"This client CLI will be given a unique device identity with the machine it is running on. " +
			"This helps identify which clients or devices are connecting to your space along with the " +
			"users that are connecting. If you use the Cloud Space client or CLI from another machine or " +
			"device it will receive its own identity but your configuration context will remain the same. " +
			"It is recommended that you give this device a name that will help you identify it within " +
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
		if !deviceNameRE.MatchString(deviceName) {
			cbcli_utils.ShowErrorAndExit(
				"Invalid device name. Device name should contain only alpha-numeric characters. It must start " +
				"and end with an alpha-numeric character and can optionally include '-'s in between.")
		}
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
		return []string{"24h", "12h", "1h", "30m", "15m", ""}
	})
	defTimeout := "15m"
	if awsAuth.Preferences().RememberFor24h {
		defTimeout = "24h"
	}
	if unlockTimeout, err = line.PromptWithSuggestion("Enter unlock timeout (i.e. ##h(ours)/m(inutes)/(s)econds) : ", defTimeout, -1); err != nil {
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
				logger.DebugMessage(
					"initialize(): Unable to unregister device with ID '%s'. Registration of new device will continue: %s", 
					oldDeviceID, err.Error(),
				)
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
			"",
		); err != nil {
			panic(err)
		}
		deviceContext.SetDeviceID(deviceIDKey, deviceID, deviceName)		
	}

	config.SetInitialized()
	fmt.Println()
}
