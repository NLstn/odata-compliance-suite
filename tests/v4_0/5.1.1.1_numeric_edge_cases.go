package v4_0

import (
	"github.com/nlstn/odata-compliance-suite/framework"
)

// NumericEdgeCases creates the 5.1.1.1 Numeric Edge Cases test suite
func NumericEdgeCases() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.1.1 Numeric Edge Cases",
		"Tests handling of numeric edge cases including very large numbers, precision limits, and boundary conditions.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_large_integer",
		"Very large integer values in filter return empty set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price gt 999999", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > 999999
			})
		},
	)

	suite.AddTest(
		"test_zero_value",
		"Zero value in numeric comparison returns matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price eq 0", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price == 0
			})
		},
	)

	suite.AddTest(
		"test_negative_numbers",
		"Negative numbers in filter match all positive-price entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price gt -1", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > -1
			})
		},
	)

	suite.AddTest(
		"test_decimal_precision",
		"Decimal precision: exact match on high-precision literal returns correct set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price eq 999.9999999", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price == 999.9999999
			})
		},
	)

	suite.AddTest(
		"test_small_decimals",
		"Very small decimal values match all entities above threshold",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price gt 0.001", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > 0.001
			})
		},
	)

	suite.AddTest(
		"test_integer_division",
		"Integer division (div) on Int16 field returns correct entity set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Quantity div 10 gt 5", func(p map[string]interface{}) bool {
				q, ok := productFloat(p, "Quantity")
				return ok && int(q)/10 > 5
			})
		},
	)

	suite.AddTest(
		"test_modulo_operation",
		"Modulo (mod) on Int16 field returns correct entity set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Quantity mod 10 eq 0", func(p map[string]interface{}) bool {
				q, ok := productFloat(p, "Quantity")
				return ok && int(q)%10 == 0
			})
		},
	)

	suite.AddTest(
		"test_numeric_null_comparison",
		"Numeric comparison combined with ne null returns non-null, positive-price entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price ne null and Price gt 0", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > 0
			})
		},
	)

	suite.AddTest(
		"test_complex_numeric_expression",
		"Complex numeric expressions with mul and div return correct entity set",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "(Price mul 2) gt 1000 and (Price div 10) lt 200", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && (price*2) > 1000 && (price/10) < 200
			})
		},
	)

	suite.AddTest(
		"test_int32_max_boundary",
		"Filter with Int32 max value matches all entities below boundary",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price lt 2147483647", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price < 2147483647
			})
		},
	)

	suite.AddTest(
		"test_arithmetic_precision",
		"Adding small increment always produces strictly larger value (arithmetic precision)",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price add 0.01 gt Price", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && (price+0.01) > price
			})
		},
	)

	suite.AddTest(
		"test_numeric_ordering",
		"Orderby Price descending returns entities in correct descending order",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=Price desc")
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
			return ctx.AssertEntitiesSortedByFloat(items, "Price", false)
		},
	)

	return suite
}
