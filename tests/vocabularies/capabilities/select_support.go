package capabilities

import (
	"fmt"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// SelectSupport creates tests for the Capabilities.SelectSupport annotation
// Tests that entity sets annotated with Supported=false reject $select requests
func SelectSupport() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities.SelectSupport Annotation",
		"Validates that entity sets annotated with Org.OData.Capabilities.V1.SelectSupport properly advertise and enforce $select capabilities.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Capabilities.V1.md#SelectSupport",
	)

	suite.AddTest(
		"non_selectable_entity_set_rejects_select",
		"$select on an entity set with SelectSupport.Supported=false returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.selectRestricted) == 0 {
				return ctx.Skip("no entity sets with SelectSupport.Supported=false found in metadata (SelectSupport is an optional annotation)")
			}

			for _, setInfo := range metadataInfo.selectRestricted {
				resp, err := ctx.GET(fmt.Sprintf("/%s?$select=%s", setInfo.name, url.QueryEscape("ID")))
				if err != nil {
					return err
				}
				if err := ctx.AssertODataError(resp, 400, ""); err != nil {
					return fmt.Errorf("entity set %s with SelectSupport.Supported=false: %w", setInfo.name, err)
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"selectable_entity_set_accepts_select",
		"$select on an entity set with no SelectSupport restriction succeeds",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=" + url.QueryEscape("ID,Name"))
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
