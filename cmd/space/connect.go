package space

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/eiannone/keyboard"
	"github.com/gookit/color"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/appbricks/mycloudspace-client/api"
	"github.com/appbricks/mycloudspace-client/mycscloud"
	"github.com/appbricks/mycloudspace-client/mycsnode"
	"github.com/appbricks/mycloudspace-client/network"
	"github.com/appbricks/mycloudspace-common/vpn"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
	"github.com/mevansam/goutils/utils"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var connectFlags = struct {
	managedDevice     string
	managedDeviceUser string
	expirationTimeout int
	inactivityTimeout int	

	useSpaceDNS    bool
	egressViaSpace bool
}{}

var connectCommand = &cobra.Command{
	Use: "connect [deployment name]",

	Short: "Connect to the space mesh network.",
	Long: `
Use this command to securely connect to your cloud space's mesh 
netowrk. This enables any device connected to this network to
communicate point-to-point securely and share services and 
applications with each other as well as access space applications
that have been shared with the mesh. Shared applications and
permissions can be managed via the MyCloudSpace account and space
management dashboard.
`,

	PreRun: authorizeSpaceNode(auth.NewRoleMask(auth.Admin, auth.Manager, auth.Guest)),

	Run: func(cmd *cobra.Command, args []string) {
		ConnectSpace(spaceNode)
	},
	Args: cobra.ExactArgs(1),
}

func ConnectSpace(space userspace.SpaceNode) {

	if space.GetStatus() != "running" {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf(
				"Space \"%s\" in \"%s\" region \"%s\" is not online.",
				space.GetSpaceName(), space.GetIaaS(), space.GetRegion(),
			),
		)
		// TODO: if the current logged in user has admin access he should be able to resume the space
	}

	if len(connectFlags.managedDevice) == 0 {
		connectToSpaceNetwork(space)
	} else {
		// if managed device option is provided we 
		// simply create download a space connection 
		// configuration for configuring a native VPN 
		// client to connect to the space network
		downloadConnectConfig(space)
	}
}

func connectToSpaceNetwork(space userspace.SpaceNode) {
	
	var (
		err error

		home    string
		isAdmin bool

		tsd *network.TailscaleDaemon
		tsc *network.TailscaleClient

		cachedIPs []string

		key keyboard.Key

		sent, recd int64
	)

	deviceContext := cbcli_config.Config.DeviceContext()

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
		cbcli_utils.ShowInfoMessage(
			"\nConnecting to space \"%s\" in \"%s\" region \"%s\".",
			space.GetSpaceName(), space.GetIaaS(), space.GetRegion(),
		)
	}
	
	// find home directory.
	home, err = homedir.Dir()
	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}

	// trap keyboard exit/termination event
	disconnect := make(chan bool, 2)

	if runtime.GOOS == "windows" {
		// ctrl-c is not trapped correctly in windows
		// as we also wait on keyboard. this handler
		// traps the event at the win32 API and handles
		// the interrupt to the connection.
		if err = run.HandleInterruptEvent(
			func() bool {
				// ensure all listeners 
				// receive the event
				disconnect <- true
				disconnect <- true
				return true
			},
		); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())	
		}	
	}

	if err := keyboard.Open(); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	go func() {
		if runtime.GOOS == "windows" {
			// don't handle ctrl-c for windows as that
			// is handled via the interrupt event handler
			for key != keyboard.KeyCtrlX {
				if _, key, err = keyboard.GetKey(); err != nil {
					if err.Error() != "operation canceled" {
						logger.ErrorMessage(
							"ConnectSpace: Unable to pause for key intput. Received error: %s", 
							err.Error(),
						)	
					}
					break
				}
			}	
		} else {
			for key != keyboard.KeyCtrlX && key != keyboard.KeyCtrlC {
				if _, key, err = keyboard.GetKey(); err != nil {
					logger.ErrorMessage(
						"ConnectSpace: Unable to pause for key intput. Received error: %s", 
						err.Error(),
					)
					break
				}
			}	
		}
		// ensure all listeners 
		// receive the event
		disconnect <- true
		disconnect <- true
	}()

	// tailscale daemon starts background network mesh connection services
	tsd = network.NewTailscaleDaemon(
		filepath.Join(home, ".cb", strings.ToLower(deviceContext.GetDevice().Name)), 
		cbcli_config.SpaceNodes, 
		cbcli_config.MonitorService, 
	)
	if cachedIPs, err = tsd.CacheDNSNames(cbcli_config.GetApiEndpointNames()); err != nil {
		logger.ErrorMessage("Error while caching API endpoints in tailscale DNS: %s", err.Error())
	}

	if err = tsd.Start(); err != nil {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf("Error starting space network mesh connection daemon: %s", err.Error()))
	}
	// tailscale client to issue commands to the background service
	if tsc, err = network.NewTailscaleClient(
		tsd.TunnelDeviceName(),
		deviceContext.GetDevice().Name,
		cbcli_config.SpaceNodes,
	); err != nil {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf("Error starting space network mesh connection daemon: %s", err.Error()))
	}
	tsc.AddSplitDestinations(cachedIPs)
	
	// cleanup on exit
	defer func() {
		_ = keyboard.Close()

		cbcli_config.ShutdownSpinner = spinner.New(
			spinner.CharSets[cbcli_config.SpinnerShutdownType], 
			100*time.Millisecond,
			spinner.WithSuffix(" Shutting down background services."),
			spinner.WithFinalMSG(""),
			spinner.WithHiddenCursor(true),
		)
		cbcli_config.ShutdownSpinner.Start()

		// terminate and cleanup tailscale connection
		if err = tsc.Disconnect(); err != nil {
			logger.DebugMessage("Error disconnecting tailscale client: %s", err.Error())
		}
		tsd.Stop()
	}()
	
	// intitiate the connecting to the space network. 
	// timeout after 10s if connection cannot be established
	errStatus := " "
	retryTimer := time.NewTicker(1000 * time.Millisecond)	
	go func() {
		defer retryTimer.Stop()

		for timeoutCounter := 0;
			timeoutCounter < 100; 
			timeoutCounter++ {

			select {
			case <-disconnect:
				return
			case <-retryTimer.C:
				if err = tsc.Connect(
					space, 
					connectFlags.useSpaceDNS, 
					connectFlags.egressViaSpace,
				); err != nil {
					logger.ErrorMessage(
						"ConnectSpace(): Failed to initiate login to the space network mesh via the client: %s", 
						err.Error(),
					)
					errStatus = color.Red.Render(" (Mesh login failed) ")
				} else {
					errStatus = " "
					return
				}	
			}
		}
		fmt.Println()
		cbcli_utils.ShowErrorMessage("Timed out while attempting to connect to space network mesh. You may not be authorized to connect.")

		disconnect <- true
	}()

	fmt.Println()
	s := spinner.New(
		spinner.CharSets[cbcli_config.SpinnerNetworkType], 
		100*time.Millisecond,
		spinner.WithSuffix(" Press CTRL-x or CTRL-c to disconnect."),
		spinner.WithFinalMSG("Connection to space network mesh has been terminated.\n"),
		spinner.WithHiddenCursor(true),
	)

	setStatus := func() {
		status := tsc.GetStatus()
		if status == "Connected" {
			if sent, recd, err = tsd.BytesTransmitted(); err != nil {
				logger.DebugMessage("Error retrieving tailscale connection status: %s", err.Error())
				s.Prefix = "\nUnable to retrieve connection status.\n"
			}
			if logrus.GetLevel()  == logrus.TraceLevel {
				if s.Prefix, err = tsd.WireguardStatusText(); err != nil {
					s.Prefix = fmt.Sprintf("Error retrieving connection status: %s", err.Error())
				}
			} else {
				s.Prefix = fmt.Sprintf(
					"Connected: recd %s, sent %s ", 
					utils.ByteCountIEC(sent), 
					utils.ByteCountIEC(recd),
				)
			}
		} else {
			s.Prefix = status + errStatus
		}
	}
	setStatus()
	s.Start()

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

func downloadConnectConfig(space userspace.SpaceNode) {

	var (
		err error

		home     string
		fileInfo os.FileInfo

		managedDevice      *userspace.Device
		managedDeviceUser  *userspace.User
		configInstructions string

		nodeAPIClient *mycsnode.ApiClient

		vpnConfigData vpn.ConfigData
		vpnConfig     vpn.Config
	)

	deviceContext := cbcli_config.Config.DeviceContext()
	if !auth.NewRoleMask(auth.Admin).LoggedInUserHasRole(deviceContext, nil /* only checking if device admin */) {
		cbcli_utils.ShowErrorAndExit("Only device admins can download space connect configurations.")
	}

	// if managed device option is provided we 
	// simply create download a space connection 
	// configuration for configuring a native VPN 
	// client to connect to the space network
	if managedDevice = deviceContext.GetManagedDevice(connectFlags.managedDevice); managedDevice == nil {
		cbcli_utils.ShowErrorAndExit("Not a valid managed device name.")
	}
	if len(connectFlags.managedDeviceUser) > 0 {
		if ownerName, _ := deviceContext.GetOwnerUserName(); connectFlags.managedDeviceUser == ownerName {
			managedDeviceUser = deviceContext.GetOwner()

		} else {
			for _, u := range managedDevice.DeviceUsers {
				if u.Name == connectFlags.managedDeviceUser {
					managedDeviceUser = u
					break
				}
			}
			if managedDeviceUser == nil {
				cbcli_utils.ShowErrorAndExit("Not a valid managed device user.")
			}	
		}
		
	} else {
		managedDeviceUser = deviceContext.GetOwner()
	}

	home, err = homedir.Dir()
	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}	
	downloadDir := filepath.Join(home, "Downloads")
	if fileInfo, err = os.Stat(downloadDir); err != nil {
		if os.IsNotExist(err) {
			downloadDir = home
		} else if fileInfo != nil && !fileInfo.IsDir() {
			downloadDir = home
		} else {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
	}
	
	// create api client for target node
	if nodeAPIClient, err = cbcli_config.SpaceNodes.GetApiClientForSpace(space); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	defer cbcli_config.SpaceNodes.ReleaseApiClientForSpace(nodeAPIClient)

	if vpnConfigData, err = vpn.NewVPNConfigData(&nodeConnectService{
		ApiClient:           nodeAPIClient,
		managedDeviceID:     managedDevice.DeviceID,
		managedDeviceUserID: managedDeviceUser.UserID,
	}); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}	
	if vpnConfig, err = vpn.NewConfigFromTarget(vpnConfigData); err != nil {
		logger.ErrorMessage("Error loading VPN configuration: %s", err.Error())
		cbcli_utils.ShowErrorAndExit(
			"Unable to retrieve VPN configuration. This could be because your VPN server " + 
			"is still starting up or in the process of shutting down. Please try again.")
	}

	// save retrieved config to local file system
	if configInstructions, err = vpnConfig.Save(downloadDir, managedDeviceUser.Name + "-"); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	fmt.Println()
	fmt.Println(configInstructions)

	// save retrieved config data to MyCS cloud 
	// so it can be shared with the device user
	mycsAPIClient := api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", cbcli_config.Config.AuthContext())
	deviceAPI := mycscloud.NewDeviceAPI(mycsAPIClient)
	if err = deviceAPI.SetDeviceWireguardConfig(
		managedDeviceUser.UserID,
		managedDevice.DeviceID, 
		space.GetSpaceID(),
		vpnConfigData.Name(),
		string(vpnConfigData.Data()),
		connectFlags.expirationTimeout * 24, // hours
		connectFlags.inactivityTimeout * 24, // hours
	); err != nil {
		logger.ErrorMessage("Error uploading wireguard config to MyCS cloud for sharing: %s", err.Error())
		cbcli_utils.ShowInfoMessage(
			"Failed to upload config so it can be shared with device users via " + 
			"the MyCS cloud dashboard. Please share downloaded config manually.",
		)
	}

	fmt.Println()
}

type nodeConnectService struct {
	*mycsnode.ApiClient
	
	managedDeviceID, 
	managedDeviceUserID string
}

func (s *nodeConnectService) Connect() (*vpn.ServiceConfig, error) {
	return s.CreateConnectConfig(
		connectFlags.useSpaceDNS, 
		connectFlags.egressViaSpace,
		s.managedDeviceID, 
		s.managedDeviceUserID,
	)
}

func init() {
	flags := connectCommand.Flags()
	flags.SortFlags = false

	flags.StringVarP(&connectFlags.managedDevice, "device", "d", "", 
		"managed device to download connection config for (device admins only)")
	flags.StringVarP(&connectFlags.managedDeviceUser, "user", "u", "", 
		"user of managed device to create connection config for (device admins only)")
	flags.IntVarP(&connectFlags.expirationTimeout, "expiration", "x", 30, 
		"managed device config expiration timeout in days")
	flags.IntVarP(&connectFlags.inactivityTimeout, "inactivity", "a", 7, 
		"managed device config inactivity timeout in days")

	flags.BoolVarP(&connectFlags.useSpaceDNS, "user-space-dns", "n", false, 
		"use space DNS services")
	flags.BoolVarP(&connectFlags.egressViaSpace, "egress-via-space", "e", false, 
		"egress all network traffic via space node")
}
