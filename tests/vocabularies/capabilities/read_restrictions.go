package capabilities

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// ReadRestrictions validates the Capabilities.ReadRestrictions annotation on
// the primary Products fixture.
func ReadRestrictions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities.ReadRestrictions Annotation",
		"Validates that advertised read support is represented correctly and agrees with collection and entity reads.",
		"https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Capabilities.V1.html#ReadRestrictions",
	)

	suite.AddTest(
		"metadata_declares_products_readable",
		"Products declares ReadRestrictions.Readable=true",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}
			value, found, err := capabilityBoolean(metadataXML, "Products", "ReadRestrictions", "Readable")
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("Products is missing Capabilities.ReadRestrictions.Readable")
			}
			if !value {
				return fmt.Errorf("Products advertises Readable=false but is the required readable reference collection")
			}
			return nil
		},
	)

	suite.AddTest(
		"advertised_collection_read_succeeds",
		"Products collection can be read when Readable=true",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"advertised_entity_read_succeeds",
		"An individual Product can be read when Readable=true",
		func(ctx *framework.TestContext) error {
			entity, err := fetchFirstEntity(ctx, "Products")
			if err != nil {
				return err
			}
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}
			setInfo, err := entitySetInfoFromMetadata(metadataXML, "Products")
			if err != nil {
				return err
			}
			key, err := buildEntityKey(entity, setInfo.keyProps)
			if err != nil {
				return err
			}
			resp, err := ctx.GET("/Products" + key)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
