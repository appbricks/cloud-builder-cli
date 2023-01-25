package auth

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/config"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/appbricks/mycloudspace-client/api"
	"github.com/appbricks/mycloudspace-client/mycscloud"
	"github.com/briandowns/spinner"
	"github.com/gookit/color"
	"github.com/mevansam/goutils/crypto"
	"github.com/mevansam/goutils/logger"
	"github.com/peterh/liner"
	"github.com/spf13/cobra"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var callbackPorts = []int{9080, 19080, 29080, 39080, 49080, 59080}

func Authenticate(config config.Config, loginMessages ...string) error {

	var (
		err error

		isAuthenticated bool
		authUrl string
	)

	authn := auth.NewAuthenticator(
		config.AuthContext(),
		&oauth2.Config{
			ClientID:     cbcli_config.CLIENT_ID,
			ClientSecret: cbcli_config.CLIENT_SECRET,
			Scopes:       []string{"openid", "profile"},
			
			Endpoint: oauth2.Endpoint{
				AuthURL:  cbcli_config.AUTH_URL,
				TokenURL: cbcli_config.TOKEN_URL,
			},
		}, 
		callBackHandler,
	)

	if isAuthenticated, err = authn.IsAuthenticated(); err != nil && err.Error() != "not authenticated" {
		return err
	}
	if !isAuthenticated {
		if len(loginMessages) > 0 {
			fmt.Println()
			cbcli_utils.ShowNoticeMessage(loginMessages[0])
		}
		if authUrl, err = authn.StartOAuthFlow(callbackPorts, logoRequestHandler); err != nil {
			logger.DebugMessage("ERROR! Authentication failed: %s", err.Error())	
			return err
		}
		if err = openBrowser(authUrl); err != nil {
			logger.DebugMessage("ERROR! Unable to open browser for authentication: %s", err.Error())

			fmt.Println()
			cbcli_utils.ShowNoteMessage(
				"You need to open a browser window and navigate to the following URL in order to " +
				"login to your My Cloud Space account. Once authenticated the CLI will be ready " +
				"for use.",
			)
			fmt.Printf("\n=> %s\n\n", authUrl)

		} else {
			fmt.Println()
			cbcli_utils.ShowNoteMessage(
				"You have been directed to a browser window from which you need to login to your " +
				"My Cloud Space account. Once authenticated the CLI will be ready for use.",
			)
			fmt.Println()
		}
		
		s := spinner.New(
			spinner.CharSets[cbcli_config.SpinnerNetworkType], 
			100*time.Millisecond,
			spinner.WithSuffix(" Waiting for authentication to complete."),
			spinner.WithFinalMSG(color.Green.Render("Authentication is complete. You are now signed in.\n")),
			spinner.WithHiddenCursor(true),
		)
		s.Start()
		for wait := true; wait; {
			wait, err = authn.WaitForOAuthFlowCompletion(time.Second)
		}
		if err != nil {
			return err
		}
		// update app config with cloud properties
		cloudAPI := mycscloud.NewCloudAPI(api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", config))
		if err = cloudAPI.UpdateProperties(config); err != nil {
			return err
		}
		s.Stop()
	}

	return nil
}

func callBackHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte(authSuccessHTML)); err != nil {
		logger.DebugMessage("ERROR! Unable to return auth success open page.")
	}
}

func logoRequestHandler() (string, func(http.ResponseWriter, *http.Request)) {
	return "/logo.png",
		func(w http.ResponseWriter, r *http.Request) {

			var (
				err  error
				data []byte
			)

			if data, err = base64.StdEncoding.DecodeString(appBricksLogoImg); err != nil {
				logger.DebugMessage("ERROR! Decoding logo image data.")
				return
			}
			if _, err = w.Write([]byte(data)); err != nil {
				logger.DebugMessage("ERROR! Unable to return logo image.")
			}
		}
}

func openBrowser(url string) (err error) {
	switch runtime.GOOS {
		case "linux":
			err = exec.Command("xdg-open", url).Run()
		case "windows":
			err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run()
		case "darwin":
			err = exec.Command("open", url).Run()
		default:
			err = fmt.Errorf("unsupported platform")
	}
	return
}

func GetAuthenticatedToken(config config.Config, forceLogin bool, loginMessages ...string) (*AWSCognitoJWT, error) {

	var (
		err error

		awsAuth *AWSCognitoJWT
	)

	if forceLogin {
		if err = config.AuthContext().Reset(); err != nil {
			return nil, err
		}
	}
	if err = Authenticate(config, loginMessages...); err != nil {				
		logger.DebugMessage("ERROR! Authentication failed: %s", err.Error())	
		return nil, err
	}
	if awsAuth, err = NewAWSCognitoJWT(config); err != nil {
		logger.DebugMessage("ERROR! Failed to extract auth token: %s", err.Error())	
		return nil, err
	}
	if err = config.SetLoggedInUser(awsAuth.UserID(), awsAuth.Username()); err != nil {
		return nil, err
	}
	return awsAuth, nil
}

func AuthorizeDeviceAndUser(config config.Config) error {

	var (
		err error

		awsAuth *AWSCognitoJWT

		user *userspace.User

		userID, 
		userName,
		ownerUserID string
		ownerConfig []byte

		ownerKey *crypto.RSAKey

		requestAccess bool
	)

	deviceAPI := mycscloud.NewDeviceAPI(api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", config))
	deviceContext := config.DeviceContext()

	// validate and parse JWT token
	if awsAuth, err = NewAWSCognitoJWT(config); err != nil {
		return err
	}
	userID = awsAuth.UserID()
	userName = awsAuth.Username()

	// authenticate device and user
	if err = deviceAPI.UpdateDeviceContext(deviceContext); err != nil {

		errStr := err.Error()
		if errStr == "unauthorized(pending)" {
			fmt.Println()
			cbcli_utils.ShowNoticeMessage("User \"%s\" is not authorized to use this device. A request to grant access to this device is still pending.", userName)

		} else if errStr == "unauthorized" {
			fmt.Println()
			cbcli_utils.ShowNoticeMessage("User \"%s\" is not authorized to use this device.", userName)			

			fmt.Println()				
			if requestAccess, err = cbcli_utils.GetYesNoUserInput("Do you wish to request access to this device : ", false); err != nil {
				return err
			}
			if (requestAccess) {
				if user, _ = deviceContext.GetGuestUser(userName); user == nil {
					if user, err = deviceContext.NewGuestUser(userID, userName); err != nil {
						return err
					}
				} else {
					user.Active = false
				}
				if _, _, err = deviceAPI.AddDeviceUser(deviceContext.GetDevice().DeviceID, ""); err != nil {
					return err
				}
				fmt.Println()
				cbcli_utils.ShowNoticeMessage("A request to grant user \"%s\" access to this device has been submitted.", user.Name)

			} else {
				return fmt.Errorf("access request declined")
			}
			
			return nil
		} else {
			return err
		}
	}

	// ensure that the device has an owner
	ownerUserID, isOwnerSet := deviceContext.GetOwnerUserID()
	if !isOwnerSet {
		fmt.Println()
		cbcli_utils.ShowCommentMessage(
			"This Cloud Builder CLI device has not been initialized. You can do this by running " +
			"the \"cb init\" command and claiming the device by logging in as the device owner.",
		)
		fmt.Println()
		os.Exit(1)
	}

	// if logged in user is the owner ensure 
	// owner is intialized and config is latest
	if userID == ownerUserID {
		owner := deviceContext.GetOwner()

		if len(owner.RSAPrivateKey) == 0 {
			fmt.Println()

			line := liner.NewLiner()
			line.SetCtrlCAborts(true)
			if ownerKey, err = ImportPrivateKey(line); err != nil {
				cbcli_utils.ShowErrorAndExit(
					fmt.Sprintf("User's private key import failed with error: %s", err.Error()),
				)
			}
			if err = owner.SetKey(ownerKey); err != nil {
				cbcli_utils.ShowErrorAndExit("Failed to validate provided private key with user's known public key.")
			}		
		}
		if config.GetConfigAsOf() < awsAuth.ConfigTimestamp() {
			userAPI := mycscloud.NewUserAPI(api.NewGraphQLClient(cbcli_config.AWS_USERSPACE_API_URL, "", config))
	
			if ownerConfig, err = userAPI.GetUserConfig(owner); err != nil {
				return err
			}
			if err = config.TargetContext().Reset(); err != nil {
				cbcli_utils.ShowErrorAndExit(
					fmt.Sprintf(
						"Failed to reset current config as a change was detected: %s", 
						err.Error(),
					),
				)
			}
			if err = config.TargetContext().Load(bytes.NewReader(ownerConfig)); err != nil {
				cbcli_utils.ShowErrorAndExit(
					fmt.Sprintf(
						"Failed to reset load updated config: %s", 
						err.Error(),
					),
				)
			}
			config.SetConfigAsOf(awsAuth.ConfigTimestamp())
		}
	} 

	return nil
}

func ImportPrivateKey(line *liner.State) (*crypto.RSAKey, error) {

	var (
		err error

		keyFile,
		keyFilePassphrase string

		key *crypto.RSAKey
	)

	if keyFile, err = line.Prompt("Path to key file (you can drag/drop from a finder/explorer window to the terminal) : "); err != nil {
		return nil, err
	}
	keyFile = strings.Trim(keyFile, " '\"")
	
	if keyFilePassphrase, err = line.PasswordPrompt("Enter the key file passphrase : "); err != nil {
		return nil, err
	}
	if key, err = crypto.NewRSAKeyFromFile(keyFile, []byte(keyFilePassphrase)); err != nil {
		return nil, err
	}
	return key, nil
}

func AssertLoggedIn() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if !cbcli_config.Config.Initialized() {
			cbcli_utils.ShowErrorAndExit(
				fmt.Sprintf(
					"You need to initialize the CLI and log in before you can invoke the command 'cb %s %s ...'", 
					cmd.Parent().Name(), cmd.Name(),
				),
			)
		}	
		if !cbcli_config.Config.AuthContext().IsLoggedIn() {
			cbcli_utils.ShowErrorAndExit(
				fmt.Sprintf(
					"You need to log in before you can invoke the command 'cb %s %s ...'", 
					cmd.Parent().Name(), cmd.Name(),
				),
			)
		}
	}
}

func AssertAuthorized(roleMask auth.RoleMask, spaceNode userspace.SpaceNode) func(cmd *cobra.Command, args []string) {

	return func(cmd *cobra.Command, args []string) {
		if cbcli_config.Config.Initialized() {
			return
		}		
		if spaceNode == nil {
			// ensure a user is logged in
			AssertLoggedIn()(cmd, args)
		}
		if !roleMask.LoggedInUserHasRole(cbcli_config.Config.DeviceContext(), spaceNode) {
			
			var accessType strings.Builder
			if roleMask.HasRole(auth.Admin) {
				accessType.WriteString("device ")
			}
			if roleMask.HasRole(auth.Manager) {
				accessType.WriteString("and space ")
			}
			accessType.WriteString("admins")

			if cmd.Parent() != nil {
				cbcli_utils.ShowNoteMessage(
					fmt.Sprintf(
						"\nOnly %s can invoke the command 'cb %s %s ...'\n", 
						accessType.String(), cmd.Parent().Name(), cmd.Name(),
					),
				)		
			} else {
				cbcli_utils.ShowNoteMessage(
					fmt.Sprintf(
						"\nOnly %s can to invoke the command 'cb %s'.\n", 
						accessType.String(), cmd.Name(),
					),
				)
			}
			// reset command
			cmd.Run = func(cmd *cobra.Command, args []string) {}
		}	
	}
}
