package v4_0

import (
	"fmt"
	"math"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

func fetchComparisonItems(ctx *framework.TestContext, filterExpr string) ([]map[string]interface{}, error) {
	filter := url.QueryEscape(filterExpr)
	resp, err := ctx.GET("/Products?$filter=" + filter)
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, err
	}

	return ctx.ParseEntityCollection(resp)
}

// FilterComparisonOperators creates the 11.3.6 Comparison Operators test suite
func FilterComparisonOperators() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.6 Comparison Operators in $filter",
		"Tests comparison operators (eq, ne, gt, ge, lt, le) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_ComparisonOperators",
	)

	// Test 1: eq (equals) operator
	suite.AddTest(
		"test_eq_operator",
		"eq (equals) operator: every returned product has Status=1 (InStock)",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Status eq 1", wantStatus(func(status int) bool {
				return status == 1
			}))
		},
	)

	// Test 2: ne (not equals) operator
	suite.AddTest(
		"test_ne_operator",
		"ne (not equals) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			// Status must be an OData enum member-name string; value 0 is "None".
			return assertProductFilter(ctx, "Status ne 0", wantStatus(func(status int) bool {
				return status != 0
			}))
		},
	)

	// Test 3: gt (greater than) operator
	suite.AddTest(
		"test_gt_operator",
		"gt (greater than) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price gt 50", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > 50
			})
		},
	)

	// Test 4: ge (greater than or equal) operator
	suite.AddTest(
		"test_ge_operator",
		"ge (greater than or equal) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price ge 50", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price >= 50
			})
		},
	)

	// Test 5: lt (less than) operator
	suite.AddTest(
		"test_lt_operator",
		"lt (less than) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price lt 100", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price < 100
			})
		},
	)

	// Test 6: le (less than or equal) operator
	suite.AddTest(
		"test_le_operator",
		"le (less than or equal) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price le 100", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price <= 100
			})
		},
	)

	// Test 7: eq with string
	suite.AddTest(
		"test_eq_string",
		"eq operator with strings: every returned product has Name='Laptop'",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Name eq 'Laptop'", func(p map[string]interface{}) bool {
				return productString(p, "Name") == "Laptop"
			})
		},
	)

	// Test 8: ne with string
	suite.AddTest(
		"test_ne_string",
		"ne operator works with strings: no returned product has Name='Laptop'",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Name ne 'Laptop'", func(p map[string]interface{}) bool {
				return productString(p, "Name") != "Laptop"
			})
		},
	)

	// Test 9: Comparison with decimal numbers
	suite.AddTest(
		"test_decimal_comparison",
		"Comparison operators work with decimal numbers",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price eq 99.99", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && math.Abs(price-99.99) < 0.001
			})
		},
	)

	// Test 10: Comparison with null
	suite.AddTest(
		"test_null_comparison",
		"Comparison with null value",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "CategoryID eq null", func(p map[string]interface{}) bool {
				v, exists := p["CategoryID"]
				return exists && v == nil
			})
		},
	)

	// Test 11: Multiple comparisons combined
	suite.AddTest(
		"test_multiple_comparisons",
		"Multiple comparison operators combined",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price ge 10 and Price le 100", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price >= 10 && price <= 100
			})
		},
	)

	// Test 12: Invalid comparison operator returns error
	suite.AddTest(
		"test_invalid_operator",
		"Invalid comparison operator returns 400",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price equals 50")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	suite.AddTest(
		"test_ne_null_on_non_nullable_property_returns_only_non_null_values",
		"ne null returns only entities where a populated property is non-null",
		func(ctx *framework.TestContext) error {
			items, err := fetchComparisonItems(ctx, "Name ne null")
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("Name ne null returned no items: %w", err)
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Name ne null", func(entity map[string]interface{}) (bool, string) {
				if value, ok := entity["Name"]; !ok || value == nil {
					return false, "Name is missing or null"
				}
				return true, ""
			})
		},
	)

	suite.AddTest(
		"test_string_numeric_type_mismatch_returns_error",
		"comparison between string property and numeric literal returns 400",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Name gt 10")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	suite.AddTest(
		"test_numeric_string_type_mismatch_returns_error",
		"comparison between numeric property and string literal returns 400",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price lt 'expensive'")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	return suite
}
