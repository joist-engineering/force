package project

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
    util "github.com/heroku/force/util"
	"io/ioutil"
	"fmt"
)

// EnvironmentConfigJson is the struct within your environment.json that
// describes a single
type EnvironmentConfigJson struct {
	InstanceHost string            `json:"instance"`
	Variables    map[string]string `json:"vars"`

	Name string
}

type project struct {
    path string

    // Lazily loaded project contents.
    lazyProjectContents *map[string][]byte
}

func LoadProject(directory string) *project {
    newProject := project{
        path: determineProjectPath(directory),
    }
    return &newProject
}

// EnvironmentsConfigJson is the root struct for JSON unmarshalling that an `environment.json` file
// in your source tree root.  It can describe your SF environments and
// other settings, particularly parameters that can be templated into your
// Salesforce metadata files.
type EnvironmentsConfigJson struct {
	Environments map[string]EnvironmentConfigJson `json:"environments"`
}

func determineProjectPath(directory string) string {
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

func (project *project) EnumerateContents() map[string][]byte {
    root := project.path

    if(project.lazyProjectContents == nil) {
        files := make(map[string][]byte)

        err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
            if f.Mode().IsRegular() {
                if f.Name() != ".DS_Store" {
                    data, err := ioutil.ReadFile(path)
                    if err != nil {
                        util.ErrorAndExit(err.Error())
                    }
                    files[strings.Replace(path, fmt.Sprintf("%s%s", root, string(os.PathSeparator)), "", -1)] = data
                }
            }
            return nil
        })
        if err != nil {
            util.ErrorAndExit(err.Error())
        }

        project.lazyProjectContents = &files
    }

    // we return a copy of the memoized data so consumers can't mutate state in the Project
    // unexpectedly.
    return *project.lazyProjectContents
}
