package target

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
)

var manageFlags = struct {
	commonFlags

	listUsers          bool	
	enableUserAsAdmin  string
	enableUserAsGuest  string
	disableUser        string
}{}

var manageCommand = &cobra.Command{
	Use: "manage [recipe] [cloud] [deployment name]",

	Short: "Manages a quick launch target deployment.",
	Long: `
This sub-command can be used to manage users' access to a space
target space. Once a user has accepted an invite to a space that user
will need to be enabled as either an admin or guest for the space
before they can connect to it. Any user that is an admin of the space
can enable users for that space. All space owners are default admins
of their space by default. Users can be permanently removed from the
authorized list via the MyCS Account Manager console.
`,

	PreRun: authorizeSpaceTarget(auth.NewRoleMask(auth.Admin, auth.Manager), &(manageFlags.commonFlags)),

	Run: func(cmd *cobra.Command, args []string) {
		Manage(getTargetKeyFromArgs(args[0], args[1], args[2], &(manageFlags.commonFlags)))
	},
	Args: cobra.ExactArgs(3),
}

func Manage(targetKey string) {
	fmt.Printf("Selected space Node: %# v\n", spaceNode)
}

func init() {
	flags := manageCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(manageFlags.commonFlags))

	flags.BoolVarP(&manageFlags.listUsers, "users", "l", false, "list all users authorized to use the space target")
	flags.StringVarP(&manageFlags.enableUserAsAdmin, "enable-admin", "u", "", "user to enable for space target with admin privileges")
	flags.StringVarP(&manageFlags.enableUserAsAdmin, "enable-guest", "g", "", "user to enable for space target with guest privileges")
	flags.StringVarP(&manageFlags.disableUser, "disable", "x", "", "user to disable (disabled users remain authorized but cannot access space functions)")
}
