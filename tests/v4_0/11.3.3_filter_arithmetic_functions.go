package v4_0

import (
	"math"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// FilterArithmeticFunctions creates the 11.3.3 Arithmetic Functions test suite.
//
// Each test verifies the operator/function's actual numeric result: the filtered
// set is compared against an oracle computed in Go from a full fetch (see
// assertProductFilter), not merely checked for HTTP 200.
func FilterArithmeticFunctions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.3 Arithmetic Functions in $filter",
		"Tests arithmetic operators and math functions (add, sub, mul, div, mod, ceiling, floor, round) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations",
	)

	suite.AddTest("test_add_operator", "add performs addition",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price add 10 gt 100", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price+10 > 100
			})
		})

	suite.AddTest("test_sub_operator", "sub performs subtraction",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price sub 50 lt 100", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price-50 < 100
			})
		})

	suite.AddTest("test_mul_operator", "mul performs multiplication",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price mul 2 gt 200", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price*2 > 200
			})
		})

	suite.AddTest("test_div_operator", "div performs division",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price div 2 lt 100", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price/2 < 100
			})
		})

	suite.AddTest("test_mod_operator", "mod computes the remainder",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Rating mod 2 eq 0", func(p map[string]interface{}) bool {
				rating, ok := productFloat(p, "Rating")
				return ok && int(rating)%2 == 0
			})
		})

	suite.AddTest("test_ceiling_function", "ceiling() rounds toward positive infinity",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ceiling(Price) eq 1000", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && math.Ceil(price) == 1000
			})
		})

	suite.AddTest("test_floor_function", "floor() rounds toward negative infinity",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "floor(Price) eq 999", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && math.Floor(price) == 999
			})
		})

	suite.AddTest("test_round_function", "round() rounds to the nearest integer",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "round(Price) eq 1000", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && math.Round(price) == 1000
			})
		})

	suite.AddTest("test_combined_arithmetic", "operators chain with correct precedence",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price mul 2 sub 100 gt 0", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price*2-100 > 0
			})
		})

	suite.AddTest("test_arithmetic_comparison", "ge and le bound a numeric range",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price ge 50 and Price le 150", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price >= 50 && price <= 150
			})
		})

	return suite
}
