package v4_0

import (
	"net/url"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// QueryFilter creates the 11.2.5.1 System Query Option $filter test suite
func QueryFilter() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.1 System Query Option $filter",
		"Tests $filter query option according to OData v4 specification, including equality, comparison, and logical operators.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionfilter",
	)

	// Test 1: Basic eq (equals) operator with string
	suite.AddTest(
		"test_filter_eq",
		"$filter with eq operator",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Name eq 'Laptop'", func(p map[string]interface{}) bool {
				return productString(p, "Name") == "Laptop"
			})
		},
	)

	// Test 2: gt (greater than) operator
	suite.AddTest(
		"test_filter_gt",
		"$filter with gt operator",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price gt 100", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > 100
			})
		},
	)

	// Test 3: String contains function
	suite.AddTest(
		"test_filter_contains",
		"$filter with contains() function",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "contains(Name,'Laptop')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "Laptop")
			})
		},
	)

	// Test 4: Boolean operators (and)
	suite.AddTest(
		"test_filter_and",
		"$filter with 'and' operator",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price gt 10 and Price lt 1000", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > 10 && price < 1000
			})
		},
	)

	// Test 5: Boolean operators (or)
	suite.AddTest(
		"test_filter_or",
		"$filter with 'or' operator",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Name eq 'Laptop' or Name eq 'Wireless Mouse'", func(p map[string]interface{}) bool {
				name := productString(p, "Name")
				return name == "Laptop" || name == "Wireless Mouse"
			})
		},
	)

	// Test 6: Parentheses for grouping
	suite.AddTest(
		"test_filter_parentheses",
		"$filter with parentheses",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "(Price gt 100) and (Price lt 1000)", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > 100 && price < 1000
			})
		},
	)

	suite.AddTest(
		"test_filter_escaped_single_quote_literal",
		"$filter accepts escaped single quotes in string literals",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "contains(Name,'''')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "'")
			})
		},
	)

	suite.AddTest(
		"test_filter_negative_numeric_literal",
		"$filter accepts negative numeric literals",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price gt -1", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > -1
			})
		},
	)

	suite.AddTest(
		"test_filter_type_mismatch_rejected",
		"$filter rejects incompatible literal types",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Name eq 1")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	suite.AddTest(
		"test_filter_empty_expression_rejected",
		"$filter with empty expression returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=")
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	return suite
}
