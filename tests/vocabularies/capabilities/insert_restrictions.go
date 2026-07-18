package capabilities

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// InsertRestrictions creates tests for the Capabilities.InsertRestrictions annotation
// Tests that entity sets annotated with InsertRestrictions properly enforce insert capabilities
func InsertRestrictions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities.InsertRestrictions Annotation",
		"Validates that entity sets annotated with Org.OData.Capabilities.V1.InsertRestrictions properly advertise and enforce insert capabilities.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Capabilities.V1.md#InsertRestrictions",
	)

	suite.AddTest(
		"metadata_includes_insert_restrictions",
		"Metadata document includes Capabilities.InsertRestrictions annotations where defined",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.insertRestricted) == 0 {
				return fmt.Errorf("no entity sets with Insertable=false found in metadata")
			}

			return nil
		},
	)

	suite.AddTest(
		"non_insertable_entity_set_rejects_post",
		"POST to entity set with Insertable=false returns appropriate error",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.insertRestricted) == 0 {
				return fmt.Errorf("no entity sets with Insertable=false found in metadata")
			}

			for _, setInfo := range metadataInfo.insertRestricted {
				// Send an otherwise-valid payload so a rejection can only be
				// attributed to Insertable=false, not to a missing required field.
				payload := buildValidPayload(setInfo)
				resp, err := ctx.POST(fmt.Sprintf("/%s", setInfo.name), payload)
				if err != nil {
					return err
				}
				if err := ctx.AssertODataError(resp, 405, ""); err != nil {
					return fmt.Errorf("non-insertable entity set %s: %w", setInfo.name, err)
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"insertable_entity_set_accepts_post",
		"POST to entity set with Insertable=true or no restriction succeeds",
		func(ctx *framework.TestContext) error {
			payload := `{"Name": "Capabilities Test Product", "Price": 79.99}`

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}

			// Should succeed for insertable entity sets
			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return fmt.Errorf("expected status 201 for insertable entity set: %w", err)
			}

			return nil
		},
	)

	return suite
}
