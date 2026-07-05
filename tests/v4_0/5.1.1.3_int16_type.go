package v4_0

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// Int16Type creates the 5.1.1.3 Int16 Type test suite
func Int16Type() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.1.3 Int16 Type",
		"Tests handling of Edm.Int16 primitive type including boundary values, filtering, and metadata representation.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_int16_in_metadata",
		"Edm.Int16 type appears in metadata as a genuine Property declaration",
		func(ctx *framework.TestContext) error {
			refs, err := propertiesDeclaredWithType(ctx, "Edm.Int16")
			if err != nil {
				return err
			}
			if len(refs) == 0 {
				return ctx.Skip("Edm.Int16 is an optional primitive type not used by this model")
			}
			for _, ref := range refs {
				if ref.Property == "" {
					return framework.NewError("EntityType " + ref.EntityType + " declares an Edm.Int16 property with no Name attribute")
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_int16_zero_value",
		"Edm.Int16 filter eq 0 returns only entities where Quantity is zero",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Quantity eq 0", func(p map[string]interface{}) bool {
				q, ok := productFloat(p, "Quantity")
				return ok && q == 0
			})
		},
	)

	suite.AddTest(
		"test_int16_positive_value",
		"Edm.Int16 filter gt large value returns correct set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Quantity gt 1000", func(p map[string]interface{}) bool {
				q, ok := productFloat(p, "Quantity")
				return ok && q > 1000
			})
		},
	)

	suite.AddTest(
		"test_int16_negative_value",
		"Edm.Int16 filter lt negative value returns correct set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Quantity lt -1000", func(p map[string]interface{}) bool {
				q, ok := productFloat(p, "Quantity")
				return ok && q < -1000
			})
		},
	)

	suite.AddTest(
		"test_int16_min_boundary",
		"Edm.Int16 filter ge -32768 matches all entities (minimum boundary)",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Quantity ge -32768", func(p map[string]interface{}) bool {
				q, ok := productFloat(p, "Quantity")
				return ok && q >= -32768
			})
		},
	)

	suite.AddTest(
		"test_int16_max_boundary",
		"Edm.Int16 filter le 32767 matches all entities (maximum boundary)",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Quantity le 32767", func(p map[string]interface{}) bool {
				q, ok := productFloat(p, "Quantity")
				return ok && q <= 32767
			})
		},
	)

	suite.AddTest(
		"test_int16_arithmetic",
		"Edm.Int16 arithmetic: Quantity mul 2 gt 100 returns correct set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Quantity mul 2 gt 100", func(p map[string]interface{}) bool {
				q, ok := productFloat(p, "Quantity")
				return ok && (q*2) > 100
			})
		},
	)

	suite.AddTest(
		"test_int16_cast",
		"cast() function supports Edm.Int16",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(Quantity,'Edm.Int16') gt 0")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_int16_orderby",
		"Edm.Int16 orderby returns entities in ascending order by Quantity",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=Quantity")
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
			return ctx.AssertEntitiesSortedByFloat(items, "Quantity", true)
		},
	)

	suite.AddTest(
		"test_int16_in_response",
		"Int16 values are serialized as numbers within the valid range [-32768, 32767]",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=10&$select=ID,Name,Quantity")
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
				q, ok := productFloat(item, "Quantity")
				if !ok {
					return framework.NewError("Quantity field is missing or not a number")
				}
				if q < -32768 || q > 32767 {
					return fmt.Errorf("Quantity value %v is outside Edm.Int16 range [-32768, 32767]", q)
				}
			}
			return nil
		},
	)

	return suite
}
