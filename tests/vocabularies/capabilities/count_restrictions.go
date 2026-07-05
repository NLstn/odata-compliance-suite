package capabilities

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// CountRestrictions creates tests for the Capabilities.CountRestrictions annotation
// Tests that entity sets annotated with Countable=false reject $count requests
func CountRestrictions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities.CountRestrictions Annotation",
		"Validates that entity sets annotated with Org.OData.Capabilities.V1.CountRestrictions properly advertise and enforce count capabilities.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Capabilities.V1.md#CountRestrictions",
	)

	suite.AddTest(
		"non_countable_entity_set_rejects_count",
		"$count=true on an entity set with Countable=false returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.countRestricted) == 0 {
				return ctx.Skip("no entity sets with Countable=false found in metadata (CountRestrictions is an optional annotation)")
			}

			for _, setInfo := range metadataInfo.countRestricted {
				resp, err := ctx.GET(fmt.Sprintf("/%s?$count=true", setInfo.name))
				if err != nil {
					return err
				}
				if resp.StatusCode != 400 {
					return fmt.Errorf("expected 400 for $count on non-countable entity set %s, got %d: %s", setInfo.name, resp.StatusCode, string(resp.Body))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"countable_entity_set_accepts_count",
		"$count=true on an entity set with no CountRestrictions (or Countable=true) succeeds",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
