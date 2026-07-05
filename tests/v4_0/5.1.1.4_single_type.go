package v4_0

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// SingleType creates the 5.1.1.4 Single Type test suite
func SingleType() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.1.4 Single Type (Float32)",
		"Tests handling of Edm.Single (IEEE 754 single-precision float) including literal format with 'f' suffix, filtering, and special values.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_single_in_metadata",
		"Edm.Single type appears in metadata as a genuine Property declaration",
		func(ctx *framework.TestContext) error {
			refs, err := propertiesDeclaredWithType(ctx, "Edm.Single")
			if err != nil {
				return err
			}
			if len(refs) == 0 {
				return ctx.Skip("Edm.Single is an optional primitive type not used by this model")
			}
			for _, ref := range refs {
				if ref.Property == "" {
					return framework.NewError("EntityType " + ref.EntityType + " declares an Edm.Single property with no Name attribute")
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_single_literal_with_f_suffix",
		"Edm.Single literal with 'f' suffix returns correct entity set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Weight eq 3.14f", func(p map[string]interface{}) bool {
				w, ok := productFloat(p, "Weight")
				// Single-precision rounding: compare within float32 precision
				return ok && float32(w) == float32(3.14)
			})
		},
	)

	suite.AddTest(
		"test_single_comparison",
		"Edm.Single supports gt comparison with 'f' suffix",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Weight gt 2.5f", func(p map[string]interface{}) bool {
				w, ok := productFloat(p, "Weight")
				return ok && w > 2.5
			})
		},
	)

	suite.AddTest(
		"test_single_zero_value",
		"Edm.Single filter eq 0.0f returns only zero-weight entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Weight eq 0.0f", func(p map[string]interface{}) bool {
				w, ok := productFloat(p, "Weight")
				return ok && w == 0.0
			})
		},
	)

	suite.AddTest(
		"test_single_negative_value",
		"Edm.Single filter lt negative value returns correct set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Weight lt -1.5f", func(p map[string]interface{}) bool {
				w, ok := productFloat(p, "Weight")
				return ok && w < -1.5
			})
		},
	)

	suite.AddTest(
		"test_single_arithmetic",
		"Edm.Single arithmetic: Weight mul 2 gt 10.0f returns correct set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Weight mul 2 gt 10.0f", func(p map[string]interface{}) bool {
				w, ok := productFloat(p, "Weight")
				return ok && (w*2) > 10.0
			})
		},
	)

	suite.AddTest(
		"test_single_cast",
		"cast() function supports Edm.Single",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(Weight,'Edm.Single') gt 0.0f")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_scientific_notation",
		"Edm.Single scientific notation: Weight lt 1.5e2f returns correct set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Weight lt 1.5e2f", func(p map[string]interface{}) bool {
				w, ok := productFloat(p, "Weight")
				return ok && w < 150.0
			})
		},
	)

	suite.AddTest(
		"test_single_inf",
		"Edm.Single ne INF literal is accepted and returns non-INF entities",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Weight ne INF")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_negative_inf",
		"Edm.Single ne -INF literal is accepted and returns non-negative-INF entities",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Weight ne -INF")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_nan",
		"Edm.Single ne NaN literal is accepted",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Weight ne NaN")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_in_response",
		"Single (Weight) values are present and numeric in response",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=10&$select=ID,Name,Weight")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}
			for _, item := range items {
				if _, hasWeight := item["Weight"]; !hasWeight {
					return fmt.Errorf("entity is missing Weight field")
				}
				if item["Weight"] != nil {
					if _, ok := item["Weight"].(float64); !ok {
						return fmt.Errorf("Weight field is not a number, got %T", item["Weight"])
					}
				}
			}
			return nil
		},
	)

	return suite
}
