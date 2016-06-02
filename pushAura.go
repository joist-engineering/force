package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/joist-engineering/force/salesforce"
	"github.com/joist-engineering/force/util"
)

var cmdPushAura = &Command{
	Usage: "pushAura",
	Short: "force pushAura -resourcepath=<filepath>",
	Long: `
	force pushAura -resourcepath <fullFilePath>

	force pushAura -f=<fullFilePath>

	`,
}

func init() {
	cmdPushAura.Run = runPushAura
	cmdPushAura.Flag.Var(&resourcepath, "f", "fully qualified file name for entity")
	cmdPushAura.Flag.StringVar(&metadataType, "t", "", "Type of entity or bundle to create")
	cmdPushAura.Flag.StringVar(&metadataType, "type", "", "Type of entity or bundle to create")
}

func runPushAura(cmd *Command, args []string) {
	// For some reason, when called from sublime, the quotes are included
	// in the resourcepath argument.  Quoting is needed if you have blank spaces
	// in the path name. So need to strip them out.
	if strings.Contains(resourcepath[0], "\"") || strings.Contains(resourcepath[0], "'") {
		resourcepath[0] = strings.Replace(resourcepath[0], "\"", "", -1)
		resourcepath[0] = strings.Replace(resourcepath[0], "'", "", -1)
	}
	absPath, _ := filepath.Abs(resourcepath[0])
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		fmt.Println(err.Error())
		util.ErrorAndExit("File does not exist\n" + absPath)
	}

	// Verify that the file is in an aura bundles folder
	if !InAuraBundlesFolder(absPath) {
		util.ErrorAndExit("File is not in an aura bundle folder (aura)")
	}

	// See if this is a directory
	info, _ := os.Stat(absPath)
	if info.IsDir() {
		// If this is a path, then it is expected be a direct child of "metatdata/aura".
		// If so, then we are going to push all the definitions in the bundle one at a time.
		filepath.Walk(absPath, func(path string, inf os.FileInfo, err error) error {
			info, err = os.Stat(filepath.Join(absPath, inf.Name()))
			if err != nil {
				fmt.Println(err.Error())
			} else {
				if info.IsDir() || inf.Name() == ".manifest" {
					fmt.Println("\nSkip")
				} else {
					pushAuraComponent(filepath.Join(absPath, inf.Name()))
				}
			}
			return nil
		})
	} else {
		pushAuraComponent(absPath)
	}
	return
}

func pushAuraComponent(fname string) {
	force, _ := ActiveForce()
	// Check for manifest file
	if _, err := os.Stat(filepath.Join(filepath.Dir(fname), ".manifest")); os.IsNotExist(err) {
		// No manifest, but is in aurabundle folder, assume creating a new bundle with this file
		// as the first artifact.
		createNewAuraBundleAndDefinition(*force, fname)
	} else {
		// Got the manifest, let's update the artifact
		fmt.Println("Updating")
		updateAuraDefinition(*force, fname)
		return
	}
}

func isValidAuraExtension(fname string) bool {
	var ext = strings.Trim(strings.ToLower(filepath.Ext(fname)), " ")
	if ext == ".app" || ext == ".cmp" || ext == ".evt" || ext == ".intf" {
		return true
	} else {
		util.ErrorAndExit("You need to create an application (.app) or component (.cmp) or and event (.evt) as the first item in your bundle.")
	}
	return false
}

func createNewAuraBundleAndDefinition(force salesforce.Force, fname string) {
	// 	Creating a new bundle. We need
	// 		the name of the bundle (parent folder of file)
	//		the type of artifact (based on naming convention)
	// 		the contents of the file
	if isValidAuraExtension(fname) {
		// Need the parent folder name to name the bundle
		var bundleName = filepath.Base(filepath.Dir(fname))
		// Create the manifext
		var manifest salesforce.BundleManifest
		manifest.Name = bundleName

		_, _ = getFormatByresourcepath(fname)
		targetDirectory, mdbase = SetTargetDirectory(fname)
		// Create a bundle defintion
		bundle, err, emessages := force.CreateAuraBundle(bundleName)
		if err != nil {
			if emessages[0].ErrorCode == "DUPLICATE_VALUE" {
				// Should look up the bundle and get it's id then update it.
				FetchManifest(bundleName)
				updateAuraDefinition(force, fname)
				return
			}
			util.ErrorAndExit(err.Error())
		} else {
			manifest.Id = bundle.Id
			component, err, emessages := createBundleEntity(manifest, force, fname)
			if err != nil {
				util.ErrorAndExit(err.Error(), emessages[0].ErrorCode)
			}
			createManifest(manifest, component, fname)
		}
	}
}

func SetTargetDirectory(fname string) (dir string, base string) {
	// Need to get the parent of metadata, do this by walking up the path
	done := false
	for done == false {
		base = filepath.Base(fname)
		fname = filepath.Dir(fname)
		if filepath.Base(fname) == "metadata" {
			dir = filepath.Dir(fname)
			done = true
		}
	}
	//return strings.Split(fname, "/metadata/aura")[0]
	return
}

func createBundleEntity(manifest salesforce.BundleManifest, force salesforce.Force, fname string) (component salesforce.ForceCreateRecordResult, err error, emessages []salesforce.ForceError) {
	// create the bundle entity
	format, deftype := getFormatByresourcepath(fname)
	mbody, _ := readFile(fname)
	component, err, emessages = force.CreateAuraComponent(map[string]string{"AuraDefinitionBundleId": manifest.Id, "DefType": deftype, "Format": format, "Source": mbody})
	return
}

func createManifest(manifest salesforce.BundleManifest, component salesforce.ForceCreateRecordResult, fname string) {
	cfile := salesforce.ComponentFile{}
	cfile.FileName = fname
	cfile.ComponentId = component.Id

	manifest.Files = append(manifest.Files, cfile)
	bmBody, _ := json.Marshal(manifest)

	ioutil.WriteFile(filepath.Join(filepath.Dir(fname), ".manifest"), bmBody, 0644)
	return
}

func updateManifest(manifest salesforce.BundleManifest, component salesforce.ForceCreateRecordResult, fname string) {
	cfile := salesforce.ComponentFile{}
	cfile.FileName = fname
	cfile.ComponentId = component.Id

	manifest.Files = append(manifest.Files, cfile)
	bmBody, _ := json.Marshal(manifest)

	ioutil.WriteFile(filepath.Join(filepath.Dir(fname), ".manifest"), bmBody, 0644)
	return
}

func GetManifest(fname string) (manifest salesforce.BundleManifest, err error) {
	manifestname := filepath.Join(filepath.Dir(fname), ".manifest")

	if _, err = os.Stat(manifestname); os.IsNotExist(err) {
		return
	}

	mbody, _ := readFile(filepath.Join(filepath.Dir(fname), ".manifest"))
	json.Unmarshal([]byte(mbody), &manifest)
	return
}

func updateAuraDefinition(force salesforce.Force, fname string) {

	//Get the manifest
	manifest, err := GetManifest(fname)

	for i := range manifest.Files {
		component := manifest.Files[i]
		if filepath.Base(component.FileName) == filepath.Base(fname) {
			//Here is where we make the call to send the update
			mbody, _ := readFile(fname)
			err := force.UpdateAuraComponent(map[string]string{"source": mbody}, component.ComponentId)
			if err != nil {
				util.ErrorAndExit(err.Error())
			}
			fmt.Printf("Aura definition updated: %s\n", filepath.Base(fname))
			return
		}
	}
	component, err, emessages := createBundleEntity(manifest, force, fname)
	if err != nil {
		util.ErrorAndExit(err.Error(), emessages[0].ErrorCode)
	}
	updateManifest(manifest, component, fname)
	fmt.Println("New component in the bundle")
}

func getFormatByresourcepath(resourcepath string) (format string, defType string) {
	var fname = strings.ToLower(resourcepath)
	if strings.Contains(fname, "application.app") {
		format = "XML"
		defType = "APPLICATION"
	} else if strings.Contains(fname, ".intf") {
		format = "XML"
		defType = "INTERFACE"
	} else if strings.Contains(fname, "component.cmp") {
		format = "XML"
		defType = "COMPONENT"
	} else if strings.Contains(fname, "event.evt") {
		format = "XML"
		defType = "EVENT"
	} else if strings.Contains(fname, "controller.js") {
		format = "JS"
		defType = "CONTROLLER"
	} else if strings.Contains(fname, "model.js") {
		format = "JS"
		defType = "MODEL"
	} else if strings.Contains(fname, "helper.js") {
		format = "JS"
		defType = "HELPER"
	} else if strings.Contains(fname, "renderer.js") {
		format = "JS"
		defType = "RENDERER"
	} else if strings.Contains(fname, "style.css") {
		format = "CSS"
		defType = "STYLE"
	} else {
		if filepath.Ext(fname) == ".app" {
			format = "XML"
			defType = "APPLICATION"
		} else if filepath.Ext(fname) == ".cmp" {
			format = "XML"
			defType = "COMPONENT"
		} else if filepath.Ext(fname) == ".evt" {
			format = "XML"
			defType = "EVENT"
		} else if filepath.Ext(fname) == ".design" {
			format = "XML"
			defType = "DESIGN"
		} else if filepath.Ext(fname) == ".svg" {
			format = "SVG"
			defType = "SVG"
		} else if filepath.Ext(fname) == ".css" {
			format = "CSS"
			defType = "STYLE"
		} else if filepath.Ext(fname) == ".intf" {
			format = "XML"
			defType = "INTERFACE"
		} else if filepath.Ext(fname) == ".auradoc" {
			format = "XML"
			defType = "DOCUMENTATION"
		} else {
			util.ErrorAndExit("Could not determine aura definition type.", fname)
		}
	}
	return
}

func getDefinitionFormat(deftype string) (result string) {
	switch strings.ToUpper(deftype) {
	case "APPLICATION", "COMPONENT", "EVENT", "INTERFACE", "DOCUMENTATION", "DESIGN":
		result = "XML"
	case "CONTROLLER", "MODEL", "HELPER", "RENDERER":
		result = "JS"
	case "STYLE":
		result = "CSS"
	case "SVG":
		result = "SVG"
	}
	return
}

func InAuraBundlesFolder(fname string) bool {
	info, _ := os.Stat(fname)
	if info.IsDir() {
		return strings.HasSuffix(filepath.Dir(fname), filepath.FromSlash("aura"))
	} else {
		return strings.HasSuffix(filepath.Dir(filepath.Dir(fname)), filepath.FromSlash("aura"))
	}
}

func readFile(resourcepath string) (body string, err error) {
	data, err := ioutil.ReadFile(resourcepath)
	if err != nil {
		return
	}
	body = string(data)
	return
}
