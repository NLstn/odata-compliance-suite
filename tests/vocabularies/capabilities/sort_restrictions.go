package capabilities

import (
	"fmt"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// SortRestrictions creates tests for the Capabilities.SortRestrictions annotation
// Tests that entity sets annotated with Sortable=false reject $orderby requests
func SortRestrictions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities.SortRestrictions Annotation",
		"Validates that entity sets annotated with Org.OData.Capabilities.V1.SortRestrictions properly advertise and enforce sort capabilities.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Capabilities.V1.md#SortRestrictions",
	)

	suite.AddTest(
		"non_sortable_entity_set_rejects_orderby",
		"$orderby on an entity set with Sortable=false returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.sortRestricted) == 0 {
				return ctx.Skip("no entity sets with Sortable=false found in metadata (SortRestrictions is an optional annotation)")
			}

			for _, setInfo := range metadataInfo.sortRestricted {
				resp, err := ctx.GET(fmt.Sprintf("/%s?$orderby=%s", setInfo.name, url.QueryEscape("ID asc")))
				if err != nil {
					return err
				}
				if resp.StatusCode != 400 {
					return fmt.Errorf("expected 400 for $orderby on non-sortable entity set %s, got %d: %s", setInfo.name, resp.StatusCode, string(resp.Body))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"sortable_entity_set_accepts_orderby",
		"$orderby on an entity set with no SortRestrictions (or Sortable=true) succeeds",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=" + url.QueryEscape("Price asc"))
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
