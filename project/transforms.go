package project

import (
    	"strconv"
	"strings"
    "encoding/xml"
    "fmt"
    "github.com/heroku/force/util"
    "github.com/heroku/force/salesforce"
)

// TransformDeployToIncludeNewFlowVersionsOnly allows you to deploy only those flows that have changed,
// and also that are active.  This is useful because stock Salesforce "helpfully"
// tries to enforce a development process by ensuring version control on flows: that is, once a flow is deployed
// and activated, it can not be replaced, only superceded.  Unfortunately, this ends up self-defeating because it
// stymies attempts to track Salesforce metadata using external change management tools.  This transform works
// around this by determining which versions have already been deployed and removes them from the package.
func TransformDeployToIncludeNewFlowVersionsOnly(sourceMetadata map[string][]byte, targetCurrentMetadata map[string][]byte) (transformedSourceMetadata map[string][]byte) {
    // make a copy of the sourceMetadata so that we can return it without modifying the source at all.
    transformedSourceMetadata = make(map[string][]byte)
    for k, v := range sourceMetadata {
        transformedSourceMetadata[k] = v
    }

    // MetadataFlowState describes the state of a given flow in an environment.
    type MetadataFlowState struct {
		ActiveVersion uint64
		Name          string

		ActiveContent salesforce.ForceMetadataItem
		AllVersions   map[uint64]salesforce.ForceMetadataItem
	}

    // EnvironmentFlowState semantically describes what flows and versions are present in an environment,
    // which are active.
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

		// now, enumerate the flows themselves and index them in:
		flowVersions := salesforce.EnumerateMetadataByType(metadataFiles, "Flow", "flows", "flow", "")
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

	targetState := determineEnvironmentState(targetCurrentMetadata, "target")
	// spew.Dump("TARGET:", targetState)

	sourceState := determineEnvironmentState(sourceMetadata, "source")
	// spew.Dump("SOURCE:", sourceState)

    // now, index the state of the flows we just determined, by using their full path names.
    // this allows us to use them to filter the transformedSourceMetadata itself.

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

    // now, we can finally transform the metadata only include flows that are active in the source (and only
	// that version) if they aren't already active in the target.
	for fileName := range transformedSourceMetadata {
		if _, presentAsInactive := inactiveFlowVersionsInSourceByCompletePath[fileName]; presentAsInactive {
			// this file is not an active flow in the source.  no point at all in deploying it.
			// so, remove it entirely from the package.
			delete(transformedSourceMetadata, fileName)
			fmt.Printf("Not bothering to deploy '%s' because it's not an active flow in our source directory.\n", fileName)
		} else {
			// it either an active flow or some other piece of metadata. awesome, we probably want to deploy it.  However, if it's already deployed
			// on the target and active, then all that no-op would do is just cause sadness (SF does not allow
			// for replacing flows)
			if _, alreadyDeployed := activeFlowsInTargetByCompletePath[fileName]; alreadyDeployed {
				// already deployed, don't need it.
				fmt.Printf("Not going to deploy '%s' because it's already deployed and active on our target!\n", fileName)
				delete(transformedSourceMetadata, fileName)
			}
		}
		// TODO alas, this negative filtering logic is a bit difficult to follow.
	}

    return
}