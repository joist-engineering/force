package project_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/heroku/force/project"
)

var _ = Describe("Project transforms", func() {
	Describe("TransformDeployToIncludeNewFlowVersionsOnly", func() {
		It("should filter out flows already active on the target", func() {
			sourceMetadata := map[string][]byte{
				"flowDefinitions/MyShittyFlow.flowDefinition": []byte(`
					<?xml version="1.0" encoding="UTF-8"?>
					<FlowDefinition xmlns="http://soap.sforce.com/2006/04/metadata">
    				<activeVersionNumber>2</activeVersionNumber>
					</FlowDefinition>`),
				"flows/MyShittyFlow-2.flow": []byte("poop"),
				"flows/MyShittyFlow-1.flow": []byte("bums"),
			}

			targetExistingMetadata := map[string][]byte{
				"flowDefinitions/MyShittyFlow.flowDefinition": []byte(`
					<?xml version="1.0" encoding="UTF-8"?>
					<FlowDefinition xmlns="http://soap.sforce.com/2006/04/metadata">
    				<activeVersionNumber>2</activeVersionNumber>
					</FlowDefinition>`),
				"flows/MyShittyFlow-2.flow": []byte("poop"),
				"flows/MyShittyFlow-1.flow": []byte("bums"),
			}

			transformedMetadata := project.TransformDeployToIncludeNewFlowVersionsOnly(
				sourceMetadata,
				targetExistingMetadata,
			)

			Ω(transformedMetadata).Should(HaveKey("flowDefinitions/MyShittyFlow.flowDefinition"))
			Ω(transformedMetadata).ShouldNot(HaveKey("flows/MyShittyFlow-2.flow"))
			Ω(transformedMetadata).Should(HaveKey("flows/MyShittyFlow-1.flow"))
		})
	})

	It("should deploy flows that aren't already active on the target", func() {
		sourceMetadata := map[string][]byte{
				"flowDefinitions/MyShittyFlow.flowDefinition": []byte(`
					<?xml version="1.0" encoding="UTF-8"?>
					<FlowDefinition xmlns="http://soap.sforce.com/2006/04/metadata">
    				<activeVersionNumber>2</activeVersionNumber>
					</FlowDefinition>`),
				"flows/MyShittyFlow-2.flow": []byte("poop"),
				"flows/MyShittyFlow-1.flow": []byte("bums"),
			}

			targetExistingMetadata := map[string][]byte{
			}

			transformedMetadata := project.TransformDeployToIncludeNewFlowVersionsOnly(
				sourceMetadata,
				targetExistingMetadata,
			)

			Ω(transformedMetadata).Should(HaveKey("flowDefinitions/MyShittyFlow.flowDefinition"))
			Ω(transformedMetadata).Should(HaveKey("flows/MyShittyFlow-2.flow"))
			Ω(transformedMetadata).Should(HaveKey("flows/MyShittyFlow-1.flow"))
	})
})
