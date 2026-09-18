package core

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

const productNameLongDescription = "The customer-facing product name used in catalog listings and search results."

// LongDescriptionAnnotation validates the richer Core.LongDescription text on
// the Product Name property in both XML and JSON CSDL.
func LongDescriptionAnnotation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Core.LongDescription Annotation",
		"Validates the detailed Product Name documentation in XML and JSON metadata.",
		"https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Core.V1.html#LongDescription",
	)

	suite.AddTest(
		"xml_metadata_includes_long_description",
		"Product/Name has the required Core.LongDescription value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			namespace, err := metadataNamespace(resp.Body)
			if err != nil {
				return err
			}
			hits, err := findAnnotationsByTerm(resp.Body, "Core.LongDescription")
			if err != nil {
				return err
			}
			target := namespace + ".Product/Name"
			for _, hit := range hits {
				if hit.Target == target && hit.String != nil && *hit.String == productNameLongDescription {
					return nil
				}
			}
			return fmt.Errorf("expected Core.LongDescription=%q on %s", productNameLongDescription, target)
		},
	)

	suite.AddTest(
		"long_description_is_more_detailed_than_description",
		"The long description is distinct from the short Product Name description",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			descriptions, err := findAnnotationsByTerm(resp.Body, "Core.Description")
			if err != nil {
				return err
			}
			for _, hit := range descriptions {
				if strings.HasSuffix(hit.Target, ".Product/Name") && hit.String != nil {
					if *hit.String == productNameLongDescription || len(*hit.String) >= len(productNameLongDescription) {
						return fmt.Errorf("Core.LongDescription must add detail beyond Core.Description")
					}
					return nil
				}
			}
			return fmt.Errorf("Product/Name is missing Core.Description")
		},
	)

	suite.AddTest(
		"json_metadata_includes_long_description",
		"JSON CSDL preserves the required Core.LongDescription value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}
			if resp.StatusCode == 406 {
				return ctx.Skip("Service does not support JSON metadata format")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			body := string(resp.Body)
			if !strings.Contains(body, `"@Org.OData.Core.V1.LongDescription"`) &&
				!strings.Contains(body, `"@Core.LongDescription"`) {
				return fmt.Errorf("JSON CSDL is missing Core.LongDescription")
			}
			if !strings.Contains(body, productNameLongDescription) {
				return fmt.Errorf("JSON CSDL is missing the required Product Name long description")
			}
			return nil
		},
	)

	return suite
}
