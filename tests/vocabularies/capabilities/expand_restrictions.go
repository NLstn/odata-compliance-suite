package capabilities

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// ExpandRestrictions creates tests for the Capabilities.ExpandRestrictions annotation
// Tests that entity sets annotated with Expandable=false reject $expand requests
func ExpandRestrictions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities.ExpandRestrictions Annotation",
		"Validates that entity sets annotated with Org.OData.Capabilities.V1.ExpandRestrictions properly advertise and enforce expand capabilities.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Capabilities.V1.md#ExpandRestrictions",
	)

	suite.AddTest(
		"non_expandable_entity_set_rejects_expand",
		"$expand=* on an entity set with Expandable=false returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.expandRestricted) == 0 {
				return ctx.Skip("no entity sets with Expandable=false found in metadata (ExpandRestrictions is an optional annotation)")
			}

			for _, setInfo := range metadataInfo.expandRestricted {
				resp, err := ctx.GET(fmt.Sprintf("/%s?$expand=*", setInfo.name))
				if err != nil {
					return err
				}
				if resp.StatusCode != 400 {
					return fmt.Errorf("expected 400 for $expand on non-expandable entity set %s, got %d: %s", setInfo.name, resp.StatusCode, string(resp.Body))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"expandable_entity_set_accepts_expand",
		"$expand on Products (which declares a navigation property and no ExpandRestrictions) succeeds",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$expand=Category")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
