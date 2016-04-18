package main

import (
	"fmt"
    "strings"
    "regexp"
    // "github.com/davecgh/go-spew/spew"
    "encoding/xml"
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


func runWipe(cmd *Command, args[]string) {
    force, _ := ActiveForce()

    query := ForceMetadataQuery{
        {Name: "ApexClass", Members: []string{"*"}},
		//{Name: "ApexComponent", Members: []string{"*"}},
		// {Name: "ApexPage", Members: []string{"*"}},
		//{Name: "ApexTrigger", Members: []string{"*"}},
    }
    files, err := force.Metadata.Retrieve(query)
	if err != nil {
		fmt.Printf("Encountered and error with retrieve...\n")
		ErrorAndExit(err.Error())
	}

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

    ApexClassTypeEntry := MetaType{
        Name: "ApexClass",
        Members: make([]string, 0),
    }



    // now, the only way to infer the resources it determine it from the regularly-formatted
    // names of metadata items that were returned to us in a the package (the Metadata API actually
    // returned a ZIP file)
    // compile a regex:
    ApexClassNameScraper, err := regexp.Compile("^classes\\/(.*)\\.cls$")
    if err != nil {
		ErrorAndExit(err.Error())
    }

    for name, _ := range files {
        //fmt.Printf("%s\n", name)
        MatchedName := ApexClassNameScraper.FindStringSubmatch(name)

        //spew.Printf("shitpoop: %v\n", MatchedName)

        if MatchedName != nil && len(MatchedName) == 2 {
            ApexClassName := MatchedName[1]
            // fmt.Printf("MATCHED AN APEX CLASS: %s\n", ApexClassName)
            ApexClassTypeEntry.Members = append(ApexClassTypeEntry.Members, ApexClassName)
        }

        // ^classes\/(.)*.cls


        // now split the bastard:
        //baseFile := strings.SplitAfter(name, "/")[1]

        //fmt.Printf("... which is actually %s\n", baseFile)


    }

    DestructiveChanges.Types = append(DestructiveChanges.Types, ApexClassTypeEntry)

    // spew.Dump(DestructiveChanges)

    byteXML, _ := xml.MarshalIndent(DestructiveChanges, "", "    ")
    byteXML = append([]byte(xml.Header), byteXML...)
    //fmt.Printf("Got XML: %s\n", string(byteXML))

    var DeploymentOptions ForceDeployOptions
    DeploymentOptions.AllowMissingFiles = false
	DeploymentOptions.AutoUpdatePackage = false
	DeploymentOptions.CheckOnly = true // lol
	DeploymentOptions.IgnoreWarnings = false
	DeploymentOptions.PurgeOnDelete = false
	DeploymentOptions.RollbackOnError = true
	DeploymentOptions.TestLevel = "RunLocalTests"

    destructiveChangesPackage := make(ForceMetadataFiles)
    destructiveChangesPackage["destructiveChanges.xml"] = byteXML

    fmt.Printf("Now deploying destructiveChanges.xml...")

    _, err = force.Metadata.Deploy(destructiveChangesPackage, DeploymentOptions)
	//problems := result.Details.ComponentFailures
	//successes := result.Details.ComponentSuccesses
	if err != nil {
		ErrorAndExit(err.Error())
	}
    // destr



}
