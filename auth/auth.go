package auth

import (
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
	"github.com/mevansam/goutils/logger"
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
			spinner.CharSets[39], 
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

		userID, 
		userName string
		user     *userspace.User

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

		if err.Error() == "unauthorized" {
			fmt.Println()
			
			if user, _ = deviceContext.GetGuestUser(userName); user == nil || user.Active /* device was deactivated in mycs account but not in device context */ {
				cbcli_utils.ShowNoticeMessage("User \"%s\" is not authorized to use this device.", user.Name)

				fmt.Println()				
				if requestAccess, err = cbcli_utils.GetYesNoUserInput("Do you wish to request access to this device : ", false); err != nil {
					return err
				}
				if (requestAccess) {
					if user, err = deviceContext.NewGuestUser(userID, userName); err != nil {
						return err
					}						
					if _, _, err = deviceAPI.AddDeviceUser(deviceContext.GetDevice().DeviceID); err != nil {
						return err
					}
					fmt.Println()
					cbcli_utils.ShowNoticeMessage("A request to grant user \"%s\" access to this device has been submitted.", user.Name)	
				} else {
					return fmt.Errorf("access request declined")
				}

			} else if (!user.Active) {
				cbcli_utils.ShowNoticeMessage("User \"%s\" is not authorized to use this device. A request to grant access to this device is still pending.", userName)
			}
			
			return nil
		} else {
			return err
		}
	}

	// ensure that the device has an owner
	_, isOwnerSet := deviceContext.GetOwnerUserName()
	if !isOwnerSet {
		fmt.Println()
		cbcli_utils.ShowCommentMessage(
			"This Cloud Builder CLI device has not been initialized. You can do this by running " +
			"the \"cb init\" command and claiming the device by logging in as the device owner.",
		)
		fmt.Println()
		os.Exit(1)
	}

	return nil
}

func AssertAuthorized(roleMask auth.RoleMask, spaceNode userspace.SpaceNode) func(cmd *cobra.Command, args []string) {

	return func(cmd *cobra.Command, args []string) {
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
						"\nOnly %s can invoke command 'cb %s %s ...'\n", 
						accessType.String(), cmd.Parent().Name(), cmd.Name(),
					),
				)		
			} else {
				cbcli_utils.ShowNoteMessage(
					fmt.Sprintf(
						"\nOnly %s can to invoke command 'cb %s'.\n", 
						accessType.String(), cmd.Name(),
					),
				)
			}
			// reset command
			cmd.Run = func(cmd *cobra.Command, args []string) {}			
		}	
	}
}
