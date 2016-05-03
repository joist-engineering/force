package main

import (
	"fmt"

	"github.com/heroku/force/util"
)

var cmdApiVersion = &Command{
	Run:   runApiVersion,
	Usage: "apiversion",
	Short: "Display/Set current API version",
	Long: `
Display/Set current API version

Examples:

  force apiversion
  force apiversion 36.0
`,
}

func init() {
}

func runApiVersion(cmd *Command, args []string) {
	force, _ := ActiveForce()
	if len(args) == 1 {
		// Todo validate that the version is in the right format
		apiVersionNumber = args[0]
		apiVersion = "v" + apiVersionNumber
		force.Credentials.ApiVersion = apiVersionNumber
		ForceSaveLogin(force.Credentials)
	} else if len(args) == 0 {
		fmt.Println(apiVersion)
	} else {
		util.ErrorAndExit("The apiversion command only accepts a single argument in the form of nn.0")
	}
}
