package utils

import (
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/mevansam/goutils/utils"
)

func ShowErrorAndExit(message string) {

	var (
		format string
	)

	if message[len(message)-1] == '.' {
		format = color.Red.Render("\nError! %s\n")
	} else {
		format = color.Red.Render("\nError! %s.\n")
	}

	fmt.Println(utils.FormatMessage(7, 80, false, true, format, message))
	os.Exit(1)
}
