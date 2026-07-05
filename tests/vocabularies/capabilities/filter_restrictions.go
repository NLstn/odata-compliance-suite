package capabilities

import (
	"fmt"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// FilterRestrictions creates tests for the Capabilities.FilterRestrictions annotation
// Tests that entity sets annotated with Filterable=false reject $filter requests
func FilterRestrictions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities.FilterRestrictions Annotation",
		"Validates that entity sets annotated with Org.OData.Capabilities.V1.FilterRestrictions properly advertise and enforce filter capabilities.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Capabilities.V1.md#FilterRestrictions",
	)

	suite.AddTest(
		"non_filterable_entity_set_rejects_filter",
		"$filter on an entity set with Filterable=false returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.filterRestricted) == 0 {
				return ctx.Skip("no entity sets with Filterable=false found in metadata (FilterRestrictions is an optional annotation)")
			}

			for _, setInfo := range metadataInfo.filterRestricted {
				resp, err := ctx.GET(fmt.Sprintf("/%s?$filter=%s", setInfo.name, url.QueryEscape("ID ne null")))
				if err != nil {
					return err
				}
				if resp.StatusCode != 400 {
					return fmt.Errorf("expected 400 for $filter on non-filterable entity set %s, got %d: %s", setInfo.name, resp.StatusCode, string(resp.Body))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"filterable_entity_set_accepts_filter",
		"$filter on an entity set with no FilterRestrictions (or Filterable=true) succeeds",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("Price gt 0"))
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
