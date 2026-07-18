package capabilities

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// DeleteRestrictions creates tests for the Capabilities.DeleteRestrictions annotation
// Tests that entity sets annotated with DeleteRestrictions properly enforce delete capabilities
func DeleteRestrictions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities.DeleteRestrictions Annotation",
		"Validates that entity sets annotated with Org.OData.Capabilities.V1.DeleteRestrictions properly advertise and enforce delete capabilities.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Capabilities.V1.md#DeleteRestrictions",
	)

	suite.AddTest(
		"metadata_includes_delete_restrictions",
		"Metadata document includes Capabilities.DeleteRestrictions annotations where defined",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.deleteRestricted) == 0 {
				return fmt.Errorf("no entity sets with Deletable=false found in metadata")
			}

			return nil
		},
	)

	suite.AddTest(
		"non_deletable_entity_rejects_delete",
		"DELETE request to entity in an entity set with Deletable=false returns appropriate error",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.deleteRestricted) == 0 {
				return fmt.Errorf("no entity sets with Deletable=false found in metadata")
			}

			for _, setInfo := range metadataInfo.deleteRestricted {
				entity, err := fetchFirstEntity(ctx, setInfo.name)
				if err != nil {
					return err
				}
				key, err := buildEntityKey(entity, setInfo.keyProps)
				if err != nil {
					return err
				}

				resp, err := ctx.DELETE(fmt.Sprintf("/%s%s", setInfo.name, key))
				if err != nil {
					return err
				}
				if err := ctx.AssertODataError(resp, 405, ""); err != nil {
					return fmt.Errorf("non-deletable entity set %s: %w", setInfo.name, err)
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"deletable_entity_accepts_delete",
		"DELETE request to entity in a deletable (unrestricted) entity set succeeds",
		func(ctx *framework.TestContext) error {
			// Create a disposable entity in the unrestricted Products set so the
			// delete does not depend on (or destroy) seed data other tests rely on.
			createResp, err := ctx.POST("/Products", map[string]interface{}{"Name": "Capabilities deletable test", "Price": 1.0})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(createResp, 201); err != nil {
				return fmt.Errorf("failed to create disposable Products entity: %w", err)
			}

			var created map[string]interface{}
			if err := ctx.GetJSON(createResp, &created); err != nil {
				return err
			}
			id, ok := created["ID"].(string)
			if !ok || id == "" {
				return fmt.Errorf("created entity did not return an ID")
			}

			resp, err := ctx.DELETE(fmt.Sprintf("/Products(%s)", id))
			if err != nil {
				return err
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("expected 2xx DELETE on deletable entity set Products, got %d: %s", resp.StatusCode, string(resp.Body))
			}

			return nil
		},
	)

	return suite
}
