package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/config"
	"github.com/briandowns/spinner"
	"github.com/gookit/color"
	"github.com/mevansam/goutils/logger"
)

const CLIENT_ID = `4edm63vj2lg5tdoogiq4c9fa5t`
const CLIENT_SECRET = `qu7g80m9ietf8feachkktq2jlu6uhm1e1elq43tilg4i0rjq7i2`
const AUTH_URL = `https://mycsdev.auth.us-east-1.amazoncognito.com/login`
const TOKEN_URL = `https://mycsdev.auth.us-east-1.amazoncognito.com/oauth2/token`

var callbackPorts = []int{9080, 19080, 29080, 39080, 49080, 59080}

func Authenticate(config config.Config) error {

	var (
		err error

		isAuthenticated bool
		authUrl string
	)

	authn := auth.NewAuthenticator(
		config.AuthContext(),
		&oauth2.Config{
			ClientID:     CLIENT_ID,
			ClientSecret: CLIENT_SECRET,
			Scopes:       []string{"openid", "profile"},
			// This points to the test Authorization Server
			// if our Client ID and Client Secret are valid
			// it will attempt to authorize our user
			Endpoint: oauth2.Endpoint{
				AuthURL:  AUTH_URL,
				TokenURL: TOKEN_URL,
			},
		}, 
		callBackHandler,
	)

	if isAuthenticated, err = authn.IsAuthenticated(); err != nil && err.Error() != "not authenticated" {
		return err
	}
	if !isAuthenticated {
		if authUrl, err = authn.StartOAuthFlow(callbackPorts, logoRequestHandler); err != nil {
			logger.DebugMessage("ERROR! Authentication failed: %s", err.Error())	
			return err
		}
		if err = openBrowser(authUrl); err != nil {
			logger.DebugMessage("ERROR! Unable to open browser for authentication: %s", err.Error())

			fmt.Println(
				color.Yellow.Render(
`
You need to open a browser window and navigate to the following URL
in order to login to your My Cloud Space account. Once authenticated
the CLI will be ready for use.
`,
				),
			)
			fmt.Printf("=> %s\n", authUrl)

		} else {
			fmt.Println(
				color.Blue.Render(
`
You have been directed to a browser window from which you need to
login to your My Cloud Space account. Once authenticated the CLI
will be ready for use.
`,
				),
			)
		}
		
		s := spinner.New(
			spinner.CharSets[39], 
			100*time.Millisecond,
			spinner.WithSuffix(" Waiting for authentication to complete."),
			spinner.WithFinalMSG("Authentication is complete. You are now signed in.\n"),
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

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Run()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run()
	case "darwin":
		return exec.Command("open", url).Run()
	default:
		return fmt.Errorf("unsupported platform")
	}
}