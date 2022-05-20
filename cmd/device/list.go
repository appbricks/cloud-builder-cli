package device

import (
	"fmt"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/mevansam/goutils/utils"
	"github.com/mevansam/termtables"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var listCommand = &cobra.Command{
	Use: "list",

	Short: "List managed devices.",
	Long: `
Lists all the managed devices created for the current primary device.
`,

	Run: func(cmd *cobra.Command, args []string) {
		ListDevices()
	},
}

func ListDevices() {

	deviceContext := cbcli_config.Config.DeviceContext()

	fmt.Printf(
		"\n%s\n==============\n\n", 
		color.OpBold.Render("Primary Device"),
	)
	fmt.Println(
		utils.FormatMessage(
			6, 80, false, false, 
			"Name: %s", deviceContext.GetDevice().Name,
		),
	)
	fmt.Println(
		utils.FormatMessage(6, 80, false, false, 
			"Type: %s", deviceContext.GetDevice().Type,
		),
	)

	guestUsers := deviceContext.GetGuestUsers()
	if len(guestUsers) > 0 {
		cbcli_utils.ShowInfoMessage(
			"\nTo add guest users to this device client have them " + 
			"login to the client and request access. The following " +
			"highlighted users have access to this device.\n",
		)
		
		for _, u := range guestUsers {
			if u.Active {
				fmt.Printf(
					"- %s (%s)\n", u.Name, 
					utils.FormatFullName(
						u.FirstName, u.MiddleName, u.FamilyName,
					),
				)
			} else {
				fmt.Printf(
					"- %s %s\n", 
					color.OpFuzzy.Render(u.Name),
					color.OpFuzzy.Render(						
						"(" +utils.FormatFullName(
							u.FirstName, u.MiddleName, u.FamilyName,
						) + ")",
					),
				)
			}
		}	
	}

	fmt.Printf(
		"\n%s\n===============\n\n",
		color.OpBold.Render("Managed Devices"),
	)

	managedDevices := deviceContext.GetManagedDevices()
	if len(managedDevices) > 0 {
		
		table := termtables.CreateTable()
		table.AddHeaders(
			color.OpBold.Render("Name"),
			color.OpBold.Render("Type"),
			color.OpBold.Render("Users"),
		)

		for i, device := range managedDevices {
			if i > 0 {
				table.AddSeparator()
			}

			if len(device.DeviceUsers) == 0 {
				table.AddRow(
					device.Name,
					device.Type,
					"",
				)	
			} else {
				deviceName := device.Name
				deviceType := device.Type
				
				for _, u := range device.DeviceUsers {
					fullName := utils.FormatFullName(
						u.FirstName, u.MiddleName, u.FamilyName,
					)
					if len(fullName) > 0 {
						table.AddRow(
							deviceName,
							deviceType,
							fmt.Sprintf(
								"%s (%s)", u.Name, fullName,
							),
						)
					} else {
						table.AddRow(
							deviceName,
							deviceType,
							u.Name,
						)						
					}
					deviceName = ""
					deviceType = ""	
				}
			}
		}
		fmt.Println(table.Render())

	} else {
		cbcli_utils.ShowInfoMessage("No managed devices found.")
	}

	fmt.Println()
}

func init() {
	flags := listCommand.Flags()
	flags.SortFlags = false
}
