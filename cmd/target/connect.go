package target

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
	"github.com/eiannone/keyboard"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/mycloudspace-client/vpn"
	"github.com/mevansam/goutils/utils"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var connectFlags = struct {
	commonFlags

	superUser bool
	download  bool
}{}

var connectCommand = &cobra.Command{
	Use: "connect [recipe] [cloud] [deployment name]",

	Short: "Connect to an existing target.",
	Long: `
Use this command to securly connect to your cloud space. This command
will establish a VPN connection to the bastion instance of the target
cloud space you specify. Once established the connection will pass
all traffic originating at your machine via the bastion gateway,
resolving any resource requests to cloud space resources or 
forwarding them to the internet. You can effectively use this 
connection as a traditional VPN to access the internet anonymously or
securely access your cloud space resources.
`,

	Run: func(cmd *cobra.Command, args []string) {
		ConnectTarget(getTargetKeyFromArgs(args[0], args[1], args[2], &(connectFlags.commonFlags)))
	},
	Args: cobra.ExactArgs(3),
}

func ConnectTarget(targetKey string) {

	var (
		err error

		tgt *target.Target
		user, passwd string

		isAdmin bool

		vpnConfig vpn.Config
		vpnClient vpn.Client

		fileInfo           os.FileInfo
		configInstructions string

		key keyboard.Key

		sent, recd int64
	)

	if tgt, err = cbcli_config.Config.TargetContext().GetTarget(targetKey); err == nil && tgt != nil {
		if err = tgt.LoadRemoteRefs(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
	
		instance := tgt.ManagedInstance("bastion")
		if instance == nil {
			cbcli_utils.ShowErrorAndExit(
				fmt.Sprintf(
					"Unable to locate a deployed bastion instance within the space target \"%s\".",
					targetKey,
				),
			)
		}

		if tgt.Status() != target.Running {
			ResumeTarget(targetKey)
		}
		
		if connectFlags.superUser {
			user = instance.RootUser()
			passwd = instance.RootPassword()
		} else {
			user = instance.NonRootUser()
			passwd = instance.NonRootPassword()
		}
		if vpnConfig, err = vpn.NewConfigFromTarget(tgt, user, passwd); err != nil {
			logger.DebugMessage("Error loading VPN configuration: %s", err.Error())
			cbcli_utils.ShowErrorAndExit("Unable to retrieve VPN configuration. This could be because your VPN server is still starting up or in the process of shutting down. Please try again.")
		}
		
		if connectFlags.download {
			home, _ := homedir.Dir()
			downloadDir := filepath.Join(home, "Downloads")
			if fileInfo, err = os.Stat(downloadDir); err != nil {
				if os.IsNotExist(err) {
					downloadDir = home
				} else {
					cbcli_utils.ShowErrorAndExit(err.Error())
				}
			}
			if !fileInfo.IsDir() {
				downloadDir = home
			}
			if configInstructions, err = vpnConfig.Save(downloadDir); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			fmt.Println()
			fmt.Println(configInstructions)

		} else {
			// re-spawn the CLI with elevated privileges 
			// if it is not running cli with such access
			if isAdmin, err = run.IsAdmin(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			if !isAdmin {
				cbcli_utils.ShowWarningMessage("\nPlease enter you password for admin priveleges required to update the network configuration, if requested.")
				if err = run.RunAsAdmin(os.Stdout, os.Stderr); err != nil {
					logger.DebugMessage(
						"Execution of CLI command with elevated privileges failed with error: %s", 
						err.Error(),
					)
					os.Exit(1)
				} else {
					os.Exit(0)
				}
			} else {
				cbcli_utils.ShowInfoMessage("\nStarting VPN connection.")
			}

			if vpnClient, err = vpnConfig.NewClient(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			if err = vpnClient.Connect(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			if err := keyboard.Open(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			defer func() {
				_ = keyboard.Close()
				if err = vpnClient.Disconnect(); err != nil {
					logger.DebugMessage("Error disconnecting from VPN: %s", err.Error())
				}
			}()

			disconnect := make(chan bool)
			go func() {
				for key != keyboard.KeyCtrlX && key != keyboard.KeyCtrlC {
					if _, key, err = keyboard.GetKey(); err != nil {
						cbcli_utils.ShowErrorAndExit(err.Error())
					}
				}
				disconnect <- true
			}()
			
			fmt.Println()
			s := spinner.New(
				spinner.CharSets[39], 
				100*time.Millisecond,
				spinner.WithSuffix(" Press CTRL-x or CTRL-c to disconnect."),
				spinner.WithFinalMSG("VPN connection has been disconnected.\n"),
				spinner.WithHiddenCursor(true),
			)
			
			setStatus := func() {
				if sent, recd, err = vpnClient.BytesTransmitted(); err != nil {
					logger.DebugMessage("Error retrieving VPN connection status: %s", err.Error())
					s.Prefix = "\nUnable to retrieve connection status.\n"
				}
				s.Prefix = fmt.Sprintf(
					"recd %s, sent %s: ", 
					utils.ByteCountIEC(sent), 
					utils.ByteCountIEC(recd),
				)
			}
			setStatus()
			s.Start()

			for {
				select {
				case <-disconnect:
					s.Stop()
					fmt.Println()
					return
				case <-time.After(time.Millisecond * 100):					
				}
				setStatus()
			}
		}		
		return
	}

	cbcli_utils.ShowErrorAndExit(
		fmt.Sprintf(
			"Target \"%s\" does not exist. Run 'cb target list' to list the currently configured targets",
			targetKey,
		),
	)
}

func init() {
	flags := connectCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(connectFlags.commonFlags))

	flags.BoolVarP(&connectFlags.superUser, "super", "u", false, 
		"connect as a super user with no restrictions")
	flags.BoolVarP(&connectFlags.download, "download", "d", false, 
		"download the VPN configuration file instead of\nestablishing a connection")
}
