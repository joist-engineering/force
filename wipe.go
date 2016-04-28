package main

import (
	"fmt"
	"strings"
	// "github.com/davecgh/go-spew/spew"
	"encoding/xml"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/heroku/force/util"
	project "github.com/heroku/force/project"
)

var cmdWipe = &Command{
	Run:   runWipe,
	Usage: "wipe",
	Short: "Completely scrub immutable types of metadata",
	Long: `
Scrub out certain types of metadata from Salesforce.account, particularly
Apex classes, triggers, process builder flows and workflows.

Certain types of metadata are not mutable on Salesforce, as an attempt
by the Salesforce team to enforce semi-inspired developer process.  Unfortunately,
for those of us who are trying to persue those same goals outside of the limited Salesforce
tools, these limitations become deeply problematic, particularly as they can block
deployment of dependencies.

Examples:

  force wipe --apex

  force wipe --triggers

  force wipe --flows
`,
}

// destructiveChanges
// https://developer.salesforce.com/docs/atlas.en-us.daas.meta/daas/daas_destructive_changes.htm

func runWipe(cmd *Command, args []string) {
	force, _ := ActiveForce()

	// a first attempt to discover files via metadata api, but, frankly,
	// we may want to not touch any Apex classes not in our local project. TODO make it switchable!
	query := ForceMetadataQuery{
		{Name: "ApexClass", Members: []string{"*"}},
		//{Name: "ApexComponent", Members: []string{"*"}},
		// {Name: "ApexPage", Members: []string{"*"}},
		{Name: "ApexTrigger", Members: []string{"*"}},
		{Name: "FlowDefinition", Members: []string{"*"}},
		{Name: "Flow", Members: []string{"*"}},
	}
	salesforceSideFiles, err := force.Metadata.Retrieve(query)
	if err != nil {
		fmt.Printf("Encountered an error with retrieve...\n")
		util.ErrorAndExit(err.Error())
	}

	root := project.DetermineProjectPath("joist/src") // lol

	files := make(ForceMetadataFiles)

	// TODO this was copypasta'd from elsewhere.  Should be refactored.
	err = filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
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

	// now, we want to generate a destructiveChanges.xml.  It's in
	// the same format as package.xml, so we'll borrow the types
	// from package builder.  (And then, we'll actually use
	// we'll actually use packagebuilder to build a deployable
	// package that contains just that destructiveChanges.xml)

	DestructiveChanges := Package{
		Version: strings.TrimPrefix(apiVersion, "v"),
		Xmlns:   "http://soap.sforce.com/2006/04/metadata",
	}

	DestructiveChanges.Types = make([]MetaType, 0)

	// shit. looks like deletes have to be ordered by dependencies in the XML file, since it's all just processed as dumb commands.

	// DEACTIVATE ALL FLOWS
	// DELETE ALL VISUALFORCE PAGES (and put up empty ones to work around limitation of at least 1 layout must be present)
    filesForType := enumerateMetadataByType(salesforceSideFiles, "ApexTrigger", "triggers", "trigger", "^DS")

	DestructiveChanges.Types = append(DestructiveChanges.Types, filesForType.MetaType())
    filesForType = enumerateMetadataByType(salesforceSideFiles, "ApexClass", "classes", "cls", "^DS|^test_DS")
	DestructiveChanges.Types = append(DestructiveChanges.Types, filesForType.MetaType())
    filesForType = enumerateMetadataByType(salesforceSideFiles, "Flow", "flows", "flow", "bogusbogusbogusbogusbogusbogus")
	DestructiveChanges.Types = append(DestructiveChanges.Types, filesForType.MetaType())
	// DestructiveChanges.Types = append(DestructiveChanges.Types, metadataEnumerator(salesforceSideFiles, "FlowDefinition", "flowDefinitions", "flowDefinition"))

	// OTHER DEPLOY STEPS:

	// DELETE ALL FLOWS
	//

	// TODO prompt the user with a list of all files that will be deleted!

	// spew.Dump(DestructiveChanges)

	byteXML, _ := xml.MarshalIndent(DestructiveChanges, "", "    ")
	byteXML = append([]byte(xml.Header), byteXML...)
	fmt.Printf("Generated destructiveChanges.xml: %s\n", string(byteXML))

	var DeploymentOptions ForceDeployOptions
	DeploymentOptions.AllowMissingFiles = true
	DeploymentOptions.AutoUpdatePackage = false
	DeploymentOptions.CheckOnly = true // lol
	DeploymentOptions.IgnoreWarnings = false
	DeploymentOptions.PurgeOnDelete = false
	DeploymentOptions.RollbackOnError = true
	DeploymentOptions.TestLevel = "RunLocalTests"

	packageBuilder := NewPushBuilder()
	packageBuilder.AddDestructiveChangesData(byteXML)

	fmt.Printf("Now deploying destructiveChanges.xml...")

	_, err = force.Metadata.Deploy(packageBuilder.ForceMetadataFiles(), DeploymentOptions)
	//problems := result.Details.ComponentFailures
	//successes := result.Details.ComponentSuccesses
	if err != nil {
		util.ErrorAndExit(err.Error())
	}
}
