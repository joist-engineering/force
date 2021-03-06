package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/joist-engineering/force/project"
	"github.com/joist-engineering/force/salesforce"
	"github.com/joist-engineering/force/util"
)

var cmdExport = &Command{
	Run:   runExport,
	Usage: "export [dir]",
	Short: "Export metadata to a local directory",
	Long: `
Export metadata to a local directory

Examples:

  force export

  force export org/schema
`,
}

func runExport(cmd *Command, args []string) {
	// Get path from args if available
	var err error
	var root string
	if len(args) == 1 {
		root, err = filepath.Abs(args[0])
	} else {
		root, err = project.GetSourceDir()
		if err != nil {
			fmt.Printf("Error obtaining root directory\n")
			util.ErrorAndExit(err.Error())
		}
	}
	if err != nil {
		fmt.Printf("Error obtaining file path\n")
		util.ErrorAndExit(err.Error())
	}
	force, _ := ActiveForce()
	sobjects, err := force.ListSobjects()
	if err != nil {
		util.ErrorAndExit(err.Error())
	}
	stdObjects := make([]string, 2, len(sobjects)+2)
	stdObjects[0] = "*"
	stdObjects[1] = "Activity"
	for _, sobject := range sobjects {
		name := sobject["name"].(string)
		if !sobject["custom"].(bool) && !strings.HasSuffix(name, "__Tag") && !strings.HasSuffix(name, "__History") && !strings.HasSuffix(name, "__Share") {
			stdObjects = append(stdObjects, name)
		}
	}
	query := salesforce.ForceMetadataQuery{
		{Name: "AccountSettings", Members: []string{"*"}},
		{Name: "ActivitiesSettings", Members: []string{"*"}},
		{Name: "AddressSettings", Members: []string{"*"}},
		{Name: "AnalyticSnapshot", Members: []string{"*"}},
		{Name: "ApexClass", Members: []string{"*"}},
		{Name: "ApexComponent", Members: []string{"*"}},
		{Name: "ApexPage", Members: []string{"*"}},
		{Name: "ApexTrigger", Members: []string{"*"}},
		{Name: "ApprovalProcess", Members: []string{"*"}},
		{Name: "AssignmentRules", Members: []string{"*"}},
		{Name: "AuthProvider", Members: []string{"*"}},
		{Name: "AutoResponseRules", Members: []string{"*"}},
		{Name: "BusinessHoursSettings", Members: []string{"*"}},
		{Name: "BusinessProcess", Members: []string{"*"}},
		{Name: "CallCenter", Members: []string{"*"}},
		{Name: "CaseSettings", Members: []string{"*"}},
		{Name: "ChatterAnswersSettings", Members: []string{"*"}},
		{Name: "CompanySettings", Members: []string{"*"}},
		{Name: "Community", Members: []string{"*"}},
		{Name: "CompactLayout", Members: []string{"*"}},
		{Name: "ConnectedApp", Members: []string{"*"}},
		{Name: "ContractSettings", Members: []string{"*"}},
		{Name: "CustomApplication", Members: []string{"*"}},
		{Name: "CustomApplicationComponent", Members: []string{"*"}},
		{Name: "CustomApplication", Members: []string{"*"}},
		{Name: "CustomField", Members: []string{"*"}},
		{Name: "CustomLabels", Members: []string{"*"}},
		{Name: "CustomObject", Members: stdObjects},
		{Name: "CustomObjectTranslation", Members: []string{"*"}},
		{Name: "CustomPageWebLink", Members: []string{"*"}},
		{Name: "CustomPermission", Members: []string{"*"}},
		{Name: "CustomSite", Members: []string{"*"}},
		{Name: "CustomTab", Members: []string{"*"}},
		{Name: "Dashboard", Members: []string{"*"}},
		{Name: "DataCategoryGroup", Members: []string{"*"}},
		{Name: "Document", Members: []string{"*"}},
		{Name: "EmailTemplate", Members: []string{"*"}},
		{Name: "EntitlementProcess", Members: []string{"*"}},
		{Name: "EntitlementSettings", Members: []string{"*"}},
		{Name: "EntitlementTemplate", Members: []string{"*"}},
		{Name: "ExternalDataSource", Members: []string{"*"}},
		{Name: "FieldSet", Members: []string{"*"}},
		{Name: "Flow", Members: []string{"*"}},
		{Name: "FlowDefinition", Members: []string{"*"}},
		{Name: "Folder", Members: []string{"*"}},
		{Name: "ForecastingSettings", Members: []string{"*"}},
		{Name: "GlobalPicklist", Members: []string{"*"}},
		{Name: "Group", Members: []string{"*"}},
		{Name: "HomePageComponent", Members: []string{"*"}},
		{Name: "HomePageLayout", Members: []string{"*"}},
		{Name: "IdeasSettings", Members: []string{"*"}},
		{Name: "KnowledgeSettings", Members: []string{"*"}},
		{Name: "Layout", Members: []string{"*"}},
		{Name: "Letterhead", Members: []string{"*"}},
		{Name: "ListView", Members: []string{"*"}},
		{Name: "LiveAgentSettings", Members: []string{"*"}},
		{Name: "LiveChatAgentConfig", Members: []string{"*"}},
		{Name: "LiveChatButton", Members: []string{"*"}},
		{Name: "LiveChatDeployment", Members: []string{"*"}},
		{Name: "MilestoneType", Members: []string{"*"}},
		{Name: "MobileSettings", Members: []string{"*"}},
		{Name: "NamedFilter", Members: []string{"*"}},
		{Name: "Network", Members: []string{"*"}},
		{Name: "OpportunitySettings", Members: []string{"*"}},
		{Name: "PermissionSet", Members: []string{"*"}},
		{Name: "Portal", Members: []string{"*"}},
		{Name: "PostTemplate", Members: []string{"*"}},
		{Name: "ProductSettings", Members: []string{"*"}},
		{Name: "Profile", Members: []string{"*"}},
		{Name: "Queue", Members: []string{"*"}},
		{Name: "QuickAction", Members: []string{"*"}},
		{Name: "QuoteSettings", Members: []string{"*"}},
		{Name: "RecordType", Members: []string{"*"}},
		{Name: "RemoteSiteSetting", Members: []string{"*"}},
		{Name: "Report", Members: []string{"*"}},
		{Name: "ReportType", Members: []string{"*"}},
		{Name: "Role", Members: []string{"*"}},
		{Name: "SamlSsoConfig", Members: []string{"*"}},
		{Name: "Scontrol", Members: []string{"*"}},
		{Name: "SecuritySettings", Members: []string{"*"}},
		{Name: "SharingReason", Members: []string{"*"}},
		{Name: "SharingRules", Members: []string{"*"}},
		{Name: "Skill", Members: []string{"*"}},
		{Name: "StaticResource", Members: []string{"*"}},
		{Name: "Territory", Members: []string{"*"}},
		{Name: "Translations", Members: []string{"*"}},
		{Name: "ValidationRule", Members: []string{"*"}},
		{Name: "Workflow", Members: []string{"*"}},
	}
	files, err := force.Metadata.Retrieve(query, salesforce.ForceRetrieveOptions{
		PreserveZip: preserveZip,
	})
	if err != nil {
		fmt.Printf("Encountered and error with retrieve...\n")
		util.ErrorAndExit(err.Error())
	}

	for name, data := range files {
		file := filepath.Join(root, name)
		dir := filepath.Dir(file)
		if err := os.MkdirAll(dir, 0755); err != nil {
			util.ErrorAndExit(err.Error())
		}
		if err := ioutil.WriteFile(filepath.Join(root, name), data, 0644); err != nil {
			util.ErrorAndExit(err.Error())
		}
	}
	fmt.Printf("Exported to %s\n", root)
}
