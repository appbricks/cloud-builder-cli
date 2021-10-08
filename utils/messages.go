package utils

import (
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/mevansam/goutils/utils"
)

func ShowErrorAndExit(message string) {
	ShowErrorMessage(message)
	os.Exit(1)
}

func ShowErrorMessage(message string) {

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
