package utils

import (
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/mevansam/goutils/utils"
	"github.com/spf13/cobra"
)

func ShowErrorAndExit(message string) {

	var (
		format string
	)

	if message[len(message)-1] == '.' {
		format = "\nError! %s\n"
	} else {
		format = "\nError! %s.\n"
	}

	fmt.Println(
		color.Red.Render(
			utils.FormatMessage(7, 80, false, true, format, message),
		),
	)
	os.Exit(1)
}

func ShowDangerMessage(message string, args ...interface{}) {

	fmt.Println(
		color.Danger.Render(
			utils.FormatMessage(
				8, 80, false, false, 
				"DANGER! " + message, 
				args...,
			),
		),
	)
}

func ShowMessage(message string, args ...interface{}) {
	fmt.Println(
		utils.FormatMessage(
			0, 80, false, false, 
			message, 
			args...,
		),
	)
}

func ShowNoteMessage(message string, args ...interface{}) {
	fmt.Println(
		color.Note.Render(
			utils.FormatMessage(
				0, 80, false, false, 
				message, 
				args...,
			),
		),
	)
}

func ShowNoticeMessage(message string, args ...interface{}) {
	fmt.Println(
		color.Notice.Render(
			utils.FormatMessage(
				0, 80, false, false, 
				message, 
				args...,
			),
		),
	)
}

func ShowCommentMessage(message string, args ...interface{}) {
	fmt.Println(
		color.Comment.Render(
			utils.FormatMessage(
				0, 80, false, false, 
				message, 
				args...,
			),
		),
	)
}

func ShowInfoMessage(message string, args ...interface{}) {
	fmt.Println(
		color.Info.Render(
			utils.FormatMessage(
				0, 80, false, false, 
				message, 
				args...,
			),
		),
	)
}

func ShowWarningMessage(message string, args ...interface{}) {
	fmt.Println(
		color.Warn.Render(
			utils.FormatMessage(
				0, 80, false, false, 
				message, 
				args...,
			),
		),
	)
}


func AssertAuthorized(cmd *cobra.Command, authorized bool) {

	if !authorized {
		if cmd.Parent() != nil {
			ShowNoteMessage(fmt.Sprintf("Only device admins can invoke command 'cb %s %s ...'\n", cmd.Parent().Name(), cmd.Name()))		
		} else {
			ShowNoteMessage(fmt.Sprintf("Only device admins can to invoke command 'cb %s'.\n", cmd.Name()))		
		}	
		os.Exit(1)
	}
}
