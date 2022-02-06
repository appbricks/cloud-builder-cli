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

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/mycloudspace-client/mycsnode"
	"github.com/appbricks/mycloudspace-client/vpn"
	vpn_common "github.com/appbricks/mycloudspace-common/vpn"
	"github.com/mevansam/goutils/utils"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var connectFlags = struct {
	commonFlags

	download  bool
}{}

var connectCommand = &cobra.Command{
	Use: "connect [recipe] [cloud] [deployment name]",

	Short: "Connect to an existing target.",
	Long: `
Use this command to securely connect to your cloud space. This
command will establish a VPN connection to the bastion instance of
the target cloud space you specify. Once established the connection
will pass all traffic originating at your machine via the bastion
gateway, resolving any resource requests to cloud space resources or
forwarding them to the internet. You can effectively use this
connection as a traditional VPN to access the internet anonymously or
securely access your cloud space resources.
`,

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		ConnectTarget(getTargetKeyFromArgs(args[0], args[1], args[2], &(connectFlags.commonFlags)))
	},
	Args: cobra.ExactArgs(3),
}

func ConnectTarget(targetKey string) {

	var (
		err error

		tgt *target.Target

		isAdmin bool

		vpnConfigData vpn_common.ConfigData
		vpnConfig     vpn_common.Config
		vpnClient     vpn_common.Client

		fileInfo           os.FileInfo
		configInstructions string

		key keyboard.Key

		sent, recd int64
	)

	if tgt, err = cbcli_config.Config.TargetContext().GetTarget(targetKey); err == nil && tgt != nil {

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
			
			// load target and retrieve vpn config
			vpnConfigData, vpnConfig = getVPNConfig(tgt)
			// save retrieved config
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
				cbcli_utils.ShowWarningMessage("\nPlease enter you password for admin privileges required to update the network configuration, if requested.")
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

			// load target and retrieve vpn config
			vpnConfigData, vpnConfig = getVPNConfig(tgt)
			// create vpn client using retrieve config
			if vpnClient, err = vpnConfig.NewClient(cbcli_config.MonitorService); err != nil {
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
					"recd %s, sent %s ", 
					utils.ByteCountIEC(sent), 
					utils.ByteCountIEC(recd),
				)
			}
			setStatus()
			s.Start()

			defer func() {
				// delete vpn configuration
				if err = vpnConfigData.Delete(); err != nil {
					logger.DebugMessage("connect(): Error deleting vpn config data: %s", err.Error())
				}
			}()

			for {
				select {
				case <-disconnect:
					s.Stop()
					fmt.Println()
					return
				case <-time.After(time.Millisecond * 500):					
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

func getVPNConfig(tgt *target.Target) (vpn_common.ConfigData, vpn_common.Config) {

	var (
		err error

		apiClient       *mycsnode.ApiClient		
		isAuthenticated bool

		vpnConfigData vpn_common.ConfigData
		vpnConfig     vpn_common.Config
	)

	if err = tgt.LoadRemoteRefs(); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}

	if apiClient, err = mycsnode.NewApiClient(cbcli_config.Config, tgt); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	if isAuthenticated, err = apiClient.Authenticate(); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	if !isAuthenticated {
		cbcli_utils.ShowErrorAndExit("Authenticate with space target failed.")
	}
	if vpnConfigData, err = vpn.NewVPNConfigData(apiClient); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}	
	if vpnConfig, err = vpn_common.NewConfigFromTarget(vpnConfigData); err != nil {
		logger.DebugMessage("Error loading VPN configuration: %s", err.Error())
		cbcli_utils.ShowErrorAndExit("Unable to retrieve VPN configuration. This could be because your VPN server is still starting up or in the process of shutting down. Please try again.")
	}
	return vpnConfigData, vpnConfig
}

func init() {
	flags := connectCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(connectFlags.commonFlags))

	flags.BoolVarP(&connectFlags.download, "download", "d", false, 
		"download the VPN configuration file instead of\nestablishing a connection")
}
