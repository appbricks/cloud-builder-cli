package initialize

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/peterh/liner"
	"github.com/spf13/cobra"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"

	"github.com/appbricks/cloud-builder-cli/config"
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

		resetPassphrase,
		passphrase,
		verifyPassphrase,
		unlockTimeout string

		timeout int
	)

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

	fmt.Println("\nInitializing Encryption\n=======================")

	resetPassphrase = "y"
	if config.Config.Initialized() {
		fmt.Println("\nConfiguration has already been intialized.")

		line.SetCompleter(func(line string) []string {
			return []string{"yes", "no"}
		})
		if resetPassphrase, err = line.PromptWithSuggestion("Do you wish to reset the passphrase : ", "no", -1); err != nil {
			panic(err)
		}
		line.SetCompleter(nil)

		resetPassphrase = strings.ToLower(resetPassphrase)
		if match, err := regexp.Match(`^((y(es)?)|(no?))$`, []byte(resetPassphrase)); !match || err != nil {
			fmt.Println("\nUnrecognized response.")
			os.Exit(1)
		}
	}

	if resetPassphrase == "y" || resetPassphrase == "yes" {
		fmt.Println()
		if passphrase, err = line.PasswordPrompt("Please enter the encryption passphrase : "); err != nil {
			panic(err)
		}
		if verifyPassphrase, err = line.PasswordPrompt("Please verify the encryption passphrase : "); err != nil {
			panic(err)
		}
		if passphrase != verifyPassphrase {
			fmt.Println("\nPassphrases do not match.")
			os.Exit(1)
		}
		config.Config.SetPassphrase(passphrase)
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
			config.Config.SetKeyTimeout(time.Duration(timeout) * time.Hour)
		case 'm':
			logger.DebugMessage("Setting unlock timeout to %d minutes.", timeout)
			config.Config.SetKeyTimeout(time.Duration(timeout) * time.Minute)
		case 's':
			logger.DebugMessage("Setting unlock timeout to %d seconds.", timeout)
			config.Config.SetKeyTimeout(time.Duration(timeout) * time.Second)
		default:
			fmt.Println("\nUnable to determine time format.")
			os.Exit(1)
		}
	} else {
		config.Config.SetKeyTimeout(0)
	}

	config.Config.SetInitialized()

	// reset auth token
	config.Config.AuthContext().Reset()

	fmt.Println()
}
