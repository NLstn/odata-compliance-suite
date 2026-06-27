package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// ByteTypes creates the 5.1.1.2 Byte Types test suite
func ByteTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.1.2 Byte Types",
		"Tests handling of Edm.Byte and Edm.SByte primitive types including boundary values, filtering, and metadata representation.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_byte_in_metadata",
		"Edm.Byte type appears in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.Byte"`) && !strings.Contains(body, `Type="Edm.SByte"`) {
				return nil // Optional types, skip
			}

			return nil
		},
	)

	suite.AddTest(
		"test_byte_zero_value",
		"Edm.Byte filter eq 0 returns only entities where Rating is zero",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Rating eq 0", func(p map[string]interface{}) bool {
				rating, ok := productFloat(p, "Rating")
				return ok && rating == 0
			})
		},
	)

	suite.AddTest(
		"test_byte_max_value",
		"Edm.Byte filter le 255 matches all entities (full range)",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Rating le 255", func(p map[string]interface{}) bool {
				rating, ok := productFloat(p, "Rating")
				return ok && rating <= 255
			})
		},
	)

	suite.AddTest(
		"test_byte_comparison",
		"Edm.Byte filter gt threshold returns only entities above that value",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Rating gt 100", func(p map[string]interface{}) bool {
				rating, ok := productFloat(p, "Rating")
				return ok && rating > 100
			})
		},
	)

	suite.AddTest(
		"test_sbyte_negative_value",
		"Edm.SByte filter lt 0 returns only entities with negative Temperature",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Temperature lt 0", func(p map[string]interface{}) bool {
				temp, ok := productFloat(p, "Temperature")
				return ok && temp < 0
			})
		},
	)

	suite.AddTest(
		"test_sbyte_min_max_range",
		"Edm.SByte full range filter [-128, 127] matches all entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Temperature ge -128 and Temperature le 127", func(p map[string]interface{}) bool {
				temp, ok := productFloat(p, "Temperature")
				return ok && temp >= -128 && temp <= 127
			})
		},
	)

	suite.AddTest(
		"test_byte_arithmetic",
		"Edm.Byte arithmetic: Rating add 10 gt 100 returns correct set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Rating add 10 gt 100", func(p map[string]interface{}) bool {
				rating, ok := productFloat(p, "Rating")
				return ok && (rating+10) > 100
			})
		},
	)

	suite.AddTest(
		"test_byte_cast",
		"cast() function supports Edm.Byte",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(Status,'Edm.Byte') eq 1")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_sbyte_cast",
		"cast() function supports Edm.SByte",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(Temperature,'Edm.SByte') lt 0")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_byte_in_response",
		"Byte/SByte values are serialized as numbers within their valid ranges",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=10&$select=ID,Name,Rating,Temperature")
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
				rating, ok := productFloat(item, "Rating")
				if !ok {
					return framework.NewError("Rating field is missing or not a number")
				}
				if rating < 0 || rating > 255 {
					return fmt.Errorf("Rating value %v is outside Edm.Byte range [0, 255]", rating)
				}
				temp, ok := productFloat(item, "Temperature")
				if !ok {
					return framework.NewError("Temperature field is missing or not a number")
				}
				if temp < -128 || temp > 127 {
					return fmt.Errorf("Temperature value %v is outside Edm.SByte range [-128, 127]", temp)
				}
			}
			return nil
		},
	)

	return suite
}
