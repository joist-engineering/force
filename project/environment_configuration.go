package project

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
    util "github.com/heroku/force/util"
)

// EnvironmentConfigJson is the struct within your environment.json that
// describes a single
type EnvironmentConfigJson struct {
	InstanceHost string            `json:"instance"`
	Variables    map[string]string `json:"vars"`

	Name string
}

// EnvironmentsConfigJson is the root struct for JSON unmarshalling that an `environment.json` file
// in your source tree root.  It can describe your SF environments and
// other settings, particularly parameters that can be templated into your
// Salesforce metadata files.
type EnvironmentsConfigJson struct {
	Environments map[string]EnvironmentConfigJson `json:"environments"`
}

func DetermineProjectPath(directory string) string {
	wd, _ := os.Getwd()
	usr, err := user.Current()
	var dir string

	//Manually handle shell expansion short cut
	if err != nil {
		if strings.HasPrefix(directory, "~") {
			util.ErrorAndExit("Cannot determine tilde expansion, please use relative or absolute path to directory.")
		} else {
			dir = directory
		}
	} else {
		if strings.HasPrefix(directory, "~") {
			dir = strings.Replace(directory, "~", usr.HomeDir, 1)
		} else {
			dir = directory
		}
	}

	root := filepath.Join(wd, dir)

	// Check for absolute path
	if filepath.IsAbs(dir) {
		root = dir
	}

	if _, err := os.Stat(filepath.Join(root, "package.xml")); os.IsNotExist(err) {
		util.ErrorAndExit(" \n" + filepath.Join(root, "package.xml") + "\ndoes not exist")
	}

	return root
}