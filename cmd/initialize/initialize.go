package initialize

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gookit/color"
	"github.com/peterh/liner"
	"github.com/spf13/cobra"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var InitCommand = &cobra.Command{
	Use: "init",

	Short: "Initializes and registers the cloud builder client.",
	Long: `
This will register or associate a cloud builder user with all CLI 
sessions. You need to register if you would like to share access to
targets or would like to synchronize access to configurations across
all your devices. It will also create client specific keys for
encryption of cloud configurations. All credentials including
configuration information are encrypted using public-private key
encryption. When you initialize the CLI for first time the keys will
be created and your private key will be saved to you system's key
store. You will need to add this key to each of your devices from
which you want to interact with or control your launch targets.
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
		
		passphrase,
		verifyPassphrase,
		unlockTimeout string

		timeout int
	)

	config := cbcli_config.Config

	line := liner.NewLiner()
	line.SetCtrlCAborts(true)
	defer func() {
		line.Close()
		if err := recover(); err != nil {
			if err == liner.ErrPromptAborted {
				fmt.Println("\nInitialization aborted.")
				os.Exit(1)
			} else {
				fmt.Println(utils.FormatMessage(7, 80, false, true, "\nError: %s.\n", err.(error).Error()))
				os.Exit(1)
			}
		}
	}()

	fmt.Println("\nInitializing Configuration Context\n==================================")

	if config.Initialized() {
		fmt.Println("\nConfiguration has already been intialized.")
	}

	resetConfig = true
	if curPrimaryUser, isSet := config.DeviceContext().GetPrimaryUser(); isSet {
		fmt.Println()
		if resetConfig, err = cbcli_utils.GetYesNoUserInput("Do you wish to reset the primary user : ", false); err != nil {
			panic(err)
		}
		if resetConfig {
			fmt.Println()
			fmt.Println(
				color.Red.Render(
					color.OpBold.Render(
						utils.FormatMultilineString(
							"DANGER! Resetting the primary user will also reset any saved configurations. " + 
							"If the current primary user has deployed cloud spaces and applications their " + 
							"configurations will be lost and may not be able to be recovered. Before proceding " + 
							"please ensure that you have exported the current configuration, in case you need " + 
							"to recover deployments associated with the current configuration.",
							8, 80, false, false),
					),
				),
			)

			fmt.Println()
			curPrimaryUserCheck, err := line.Prompt("Enter the name of current primary user whose configuration will be overwritten : ")
			if err != nil {
				panic(err)
			}
			if curPrimaryUserCheck != curPrimaryUser {
				cbcli_utils.ShowErrorAndExit("In order to reset the current configuration you need to enter the primary user of the current configuration.")
			}
			if err = config.Reset(); err != nil {
				cbcli_utils.ShowErrorAndExit("Failed to reset current configuration.")
			}
		}
	}
	if resetConfig {
		fmt.Println(
`
Please login as the primary user that will be configured as the owner of this
configuration context. Once configured this user will own all spaces and 
applications launched via the CB CLI with this configuration context. You need
to re-initialize the CLI to change the primary user which also reset any saved
configuration.`,
		)
		if err = cbcli_auth.Authenticate(cbcli_config.Config); err != nil {					
			cbcli_utils.ShowErrorAndExit("My Cloud Space user authentication failed.")
		}
		if awsAuth, err = cbcli_auth.NewAWSCognitoJWT(config); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		if err = awsAuth.ParseJWT(config.AuthContext().GetToken()); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		config.DeviceContext().SetPrimaryUser(awsAuth.Username())
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

	config.SetInitialized()
	fmt.Println()
}
