package main

import (
	"fmt"

	"github.com/heroku/force/salesforce"
)

var apiVersion = "v36.0"
var apiVersionNumber = "36.0"

//Dood, what
var cmdVersion = &Command{
	Run:   runVersion,
	Usage: "version",
	Short: "Display current version",
	Long: `
Display current version

Examples:

  force version
`,
}

func init() {
}

func runVersion(cmd *Command, args []string) {
	fmt.Println(salesforce.Version)
}
