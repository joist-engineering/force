package main

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"github.com/heroku/force/project"
	"github.com/heroku/force/util"
	"github.com/heroku/force/salesforce"
)

var cmdImport = &Command{
	Usage: "import [deployment options]",
	Short: "Import metadata from a local directory",
	Long: `
Import metadata from a local directory

Deployment Options
  -rollbackonerror, -r    Indicates whether any failure causes a complete rollback
  -runalltests, -t        If set all Apex tests defined in the organization are run (equivalent to -l RunAllTestsInOrg)
  -checkonly, -c          Indicates whether classes and triggers are saved during deployment
  -purgeondelete, -p      If set the deleted components are not stored in recycle bin
  -allowmissingfiles, -m  Specifies whether a deploy succeeds even if files missing
  -autoupdatepackage, -u  Auto add files to the package if missing
  -test                   Run tests in class (implies -l RunSpecifiedTests)
  -testlevel, -l          Set test level (NoTestRun, RunSpecifiedTests, RunLocalTests, RunAllTestsInOrg)
  -ignorewarnings, -i     Indicates if warnings should fail deployment or not
  -directory, -d 		  Path to the package.xml file to import
  -verbose, -v 			  Provide detailed feedback on operation

Examples:

  force import

  force import -directory=my_metadata -c -r -v

  force import -checkonly -runalltests
`,
}

var (
	testsToRun            metaName
	rollBackOnErrorFlag   = cmdImport.Flag.Bool("rollbackonerror", false, "set roll back on error")
	runAllTestsFlag       = cmdImport.Flag.Bool("runalltests", false, "set run all tests")
	testLevelFlag         = cmdImport.Flag.String("testLevel", "NoTestRun", "set test level")
	checkOnlyFlag         = cmdImport.Flag.Bool("checkonly", false, "set check only")
	purgeOnDeleteFlag     = cmdImport.Flag.Bool("purgeondelete", false, "set purge on delete")
	allowMissingFilesFlag = cmdImport.Flag.Bool("allowmissingfiles", false, "set allow missing files")
	autoUpdatePackageFlag = cmdImport.Flag.Bool("autoupdatepackage", false, "set auto update package")
	ignoreWarningsFlag    = cmdImport.Flag.Bool("ignorewarnings", false, "set ignore warnings")
	directory             = cmdImport.Flag.String("directory", "metadata", "relative path to package.xml")
	verbose               = cmdImport.Flag.Bool("verbose", false, "give more verbose output")
)

func init() {
	cmdImport.Run = runImport
	cmdImport.Flag.BoolVar(verbose, "v", false, "give more verbose output")
	cmdImport.Flag.BoolVar(rollBackOnErrorFlag, "r", false, "set roll back on error")
	cmdImport.Flag.BoolVar(runAllTestsFlag, "t", false, "set run all tests")
	cmdImport.Flag.StringVar(testLevelFlag, "l", "NoTestRun", "set test level")
	cmdImport.Flag.BoolVar(checkOnlyFlag, "c", false, "set check only")
	cmdImport.Flag.BoolVar(purgeOnDeleteFlag, "p", false, "set purge on delete")
	cmdImport.Flag.BoolVar(allowMissingFilesFlag, "m", false, "set allow missing files")
	cmdImport.Flag.BoolVar(autoUpdatePackageFlag, "u", false, "set auto update package")
	cmdImport.Flag.BoolVar(ignoreWarningsFlag, "i", false, "set ignore warnings")
	cmdImport.Flag.StringVar(directory, "d", "metadata", "relative path to package.xml")
	cmdImport.Flag.Var(&testsToRun, "test", "Test(s) to run")
}

func runImport(cmd *Command, args []string) {
	if len(args) > 0 {
		util.ErrorAndExit("Unrecognized argument: " + args[0])
	}

	loadedProject := project.LoadProject(*directory)

	force, err := ActiveForce()
	if err != nil {
		util.ErrorAndExit(err.Error())
	}
	files := loadedProject.EnumerateContents()

	if projectEnvironmentConfig := loadedProject.GetEnvironmentConfigForActiveEnvironment(force.Credentials.InstanceUrl); projectEnvironmentConfig != nil {
		fmt.Printf("About to deploy to: %s at %s\n", projectEnvironmentConfig.Name, projectEnvironmentConfig.InstanceHost)

		files = *loadedProject.ContentsWithAllTransformsApplied(projectEnvironmentConfig)
	}

	// Now to handle the metadata types that Salesforce has implemented their own versioning regimes for,
	// do a retrieval of the current content of the environment.

	// strategy: determine what flows are active in the target DONE
	// determine what flows are active in the source
	// include only the flows that are *not* active in the target (or not present) and are active in the source.
	// be sure to include *all* of the flowdefinitions themselves.
	// if there are any active flows in the target that were not covered, list 'em out.
	// Also don't bother deploying older versions of any flows. just deploy the current one.

	// NB use `func (fm *ForceMetadata) ListConnectedApps()` code as example for doing
	// simple query of XML.

	query := salesforce.ForceMetadataQuery{
		{Name: "FlowDefinition", Members: []string{"*"}},
		{Name: "Flow", Members: []string{"*"}},
	}

	targetFlowsAndDefinitions, err := force.Metadata.Retrieve(query, salesforce.ForceRetrieveOptions{})
	if err != nil {
		fmt.Printf("Encountered an error with retrieve...\n")
		util.ErrorAndExit(err.Error())
	}

	type MetadataFlowState struct {
		ActiveVersion uint64
		Name          string

		ActiveContent salesforce.ForceMetadataItem
		AllVersions   map[uint64]salesforce.ForceMetadataItem
	}

	type EnvironmentFlowState struct {
		EnvironmentName string
		ActiveFlows     map[string]MetadataFlowState
		InactiveFlows   map[string]MetadataFlowState
	}

	determineEnvironmentState := func(metadataFiles salesforce.ForceMetadataFiles, environmentName string) EnvironmentFlowState {
		flowDefinitions := salesforce.EnumerateMetadataByType(metadataFiles, "FlowDefinition", "flowDefinitions", "flowDefinition", "")

		state := EnvironmentFlowState{
			EnvironmentName: environmentName,
			ActiveFlows:     make(map[string]MetadataFlowState),
			InactiveFlows:   make(map[string]MetadataFlowState),
		}
		// First, determine what flows are active.
		for _, item := range flowDefinitions.Members {
			var res salesforce.FlowDefinition

			if err := xml.Unmarshal(item.Content, &res); err != nil {
				util.ErrorAndExit(err.Error())
			}

			if res.ActiveVersionNumber != 0 {
				state.ActiveFlows[item.Name] = MetadataFlowState{
					ActiveVersion: res.ActiveVersionNumber,
					Name:          item.Name,
					AllVersions:   make(map[uint64]salesforce.ForceMetadataItem),
				}
			} else {
				state.InactiveFlows[item.Name] = MetadataFlowState{
					ActiveVersion: res.ActiveVersionNumber,
					Name:          item.Name,
					AllVersions:   make(map[uint64]salesforce.ForceMetadataItem),
				}
			}
		}

		// now, enumerate the flow versions themselves and index them in:
		flowVersions := salesforce.EnumerateMetadataByType(targetFlowsAndDefinitions, "Flow", "flows", "flow", "")
		for _, version := range flowVersions.Members {

			// the version number is indicated by a normalized naming convention in the entries rendered by the
			// Metadata API: -$version appended to the name
			//   MyFlow-4

			nameFragments := strings.Split(version.Name, "-")
			name := nameFragments[0]
			versionNumber, err := strconv.ParseUint(nameFragments[len(nameFragments)-1], 10, 64)
			if err != nil {
				util.ErrorAndExit(err.Error())
			}

			if flowDefinition, present := state.InactiveFlows[name]; present {
				flowDefinition.AllVersions[versionNumber] = version
			} else if flowDefinition, present := state.ActiveFlows[name]; present {
				flowDefinition.AllVersions[versionNumber] = version
				// set the FlowContent value for the version we have here if it's indeed the active one:
				if state.ActiveFlows[name].ActiveVersion == versionNumber {
					flowDefinition.ActiveContent = version
				}
				// alas because golang is silly and prevents us from mutating stuff in maps
				// while being an imperative language, we have to copy the value, mutate it, and re-insert it.
				state.ActiveFlows[name] = flowDefinition
			} else {
				fmt.Printf("Warning: found a flow version instance on %s for which we have no flow definition at all, consider cleaning it up: %s\n", environmentName, name)
			}
		}

		return state
	}

	targetState := determineEnvironmentState(targetFlowsAndDefinitions, "target")
	// spew.Dump("TARGET:", targetState)

	sourceState := determineEnvironmentState(files, "source")
	// spew.Dump("SOURCE:", sourceState)

	// now, we want to modify the changeset we're going to deploy to only include flows that are active in the source (and only
	// that version) if they aren't already active in the target.

	activeFlowsInSourceByCompletePath := make(map[string]MetadataFlowState)
	for _, flowState := range sourceState.ActiveFlows {
		activeFlowsInSourceByCompletePath[flowState.ActiveContent.CompletePath] = flowState
	}

	activeFlowsInTargetByCompletePath := make(map[string]MetadataFlowState)
	for _, flowState := range targetState.ActiveFlows {
		activeFlowsInTargetByCompletePath[flowState.ActiveContent.CompletePath] = flowState
	}

	inactiveFlowVersionsInSourceByCompletePath := make(map[string]salesforce.ForceMetadataItem)
	for _, flowState := range sourceState.InactiveFlows {
		for _, flowStateVersion := range flowState.AllVersions {
			inactiveFlowVersionsInSourceByCompletePath[flowStateVersion.CompletePath] = flowStateVersion
		}
	}

	for fileName := range files {
		if _, presentAsInactive := inactiveFlowVersionsInSourceByCompletePath[fileName]; presentAsInactive {
			// this file is not an active flow in the source.  no point at all in deploying it.
			// so, remove it entirely from the package.
			delete(files, fileName)
			fmt.Printf("Not bothering to deploy '%s' because it's not an active flow in our source directory.\n", fileName)
		} else {
			// it either an active flow or some other piece of metadata. awesome, we probably want to deploy it.  However, if it's already deployed
			// on the target and active, then all that no-op would do is just cause sadness (SF does not allow
			// for replacing flows)
			if _, alreadyDeployed := activeFlowsInTargetByCompletePath[fileName]; alreadyDeployed {
				// already deployed, don't need it.
				fmt.Printf("Not going to deploy '%s' because it's already deployed and active on our target!\n", fileName)
				delete(files, fileName)
			}
		}
		// all that will be left is all other metadata than flows, plus any active ones that are not already deployed!
		// TODO alas, this negative filtering logic is a bit difficult to follow.
	}

	var DeploymentOptions salesforce.ForceDeployOptions
	DeploymentOptions.AllowMissingFiles = *allowMissingFilesFlag
	DeploymentOptions.AutoUpdatePackage = *autoUpdatePackageFlag
	DeploymentOptions.CheckOnly = *checkOnlyFlag
	DeploymentOptions.IgnoreWarnings = *ignoreWarningsFlag
	DeploymentOptions.PurgeOnDelete = *purgeOnDeleteFlag
	DeploymentOptions.RollbackOnError = *rollBackOnErrorFlag
	DeploymentOptions.TestLevel = *testLevelFlag
	if *runAllTestsFlag {
		DeploymentOptions.TestLevel = "RunAllTestsInOrg"
	}
	DeploymentOptions.RunTests = testsToRun

	result, err := force.Metadata.Deploy(files, DeploymentOptions)
	problems := result.Details.ComponentFailures
	successes := result.Details.ComponentSuccesses
	if err != nil {
		util.ErrorAndExit(err.Error())
	}

	fmt.Printf("\nFailures - %d\n", len(problems))
	if *verbose {
		for _, problem := range problems {
			if problem.FullName == "" {
				fmt.Println(problem.Problem)
			} else {
				fmt.Printf("%s: %s\n", problem.FullName, problem.Problem)
			}
		}
	}

	fmt.Printf("\nSuccesses - %d\n", len(successes))
	if *verbose {
		for _, success := range successes {
			if success.FullName != "package.xml" {
				verb := "unchanged"
				if success.Changed {
					verb = "changed"
				} else if success.Deleted {
					verb = "deleted"
				} else if success.Created {
					verb = "created"
				}
				fmt.Printf("%s\n\tstatus: %s\n\tid=%s\n", success.FullName, verb, success.Id)
			}
		}
	}
	fmt.Printf("Imported from %s\n", loadedProject.LoadedFromPath())
}
