package v4_0

import (
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// InvalidQueryParameters creates the 8.1.6 Invalid Query Parameters test suite
func InvalidQueryParameters() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.1.6 Invalid Query Parameters",
		"Tests that services properly reject invalid or unknown query parameters with 400 Bad Request according to OData specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptions",
	)

	suite.AddTest(
		"test_unknown_system_query_option",
		"Unknown system query option returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// $unknown is not a valid OData system query option
			resp, err := ctx.GET("/Products?$unknown=value")
			if err != nil {
				return err
			}

			// Per OData spec, unknown system query options (starting with $) MUST result in 400
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	suite.AddTest(
		"test_invalid_filter_syntax",
		"Invalid filter syntax returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// Invalid filter expression with unmatched parentheses
			resp, err := ctx.GET("/Products?$filter=(Price gt 10")
			if err != nil {
				return err
			}

			return ctx.AssertODataError(resp, 400, "")
		},
	)

	suite.AddTest(
		"test_invalid_orderby_syntax",
		"Invalid orderby syntax returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// Invalid orderby with unknown direction
			resp, err := ctx.GET("/Products?$orderby=Name invalid")
			if err != nil {
				return err
			}

			return ctx.AssertODataError(resp, 400, "")
		},
	)

	suite.AddTest(
		"test_invalid_top_value",
		"Invalid $top value returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// $top must be a non-negative integer
			resp, err := ctx.GET("/Products?$top=-5")
			if err != nil {
				return err
			}

			return ctx.AssertODataError(resp, 400, "")
		},
	)

	suite.AddTest(
		"test_invalid_skip_value",
		"Invalid $skip value returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// $skip must be a non-negative integer
			resp, err := ctx.GET("/Products?$skip=abc")
			if err != nil {
				return err
			}

			return ctx.AssertODataError(resp, 400, "")
		},
	)

	suite.AddTest(
		"test_duplicate_system_query_option",
		"Duplicate system query options should return 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// Per OData spec, duplicate system query options are not allowed
			resp, err := ctx.GET("/Products?$top=5&$top=10")
			if err != nil {
				return err
			}

			return ctx.AssertODataError(resp, 400, "")
		},
	)

	suite.AddTest(
		"test_invalid_expand_path",
		"Invalid $expand path returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// Try to expand a non-existent navigation property
			resp, err := ctx.GET("/Products?$expand=NonExistentNavProperty")
			if err != nil {
				return err
			}

			// Should return 400 for invalid navigation property
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	suite.AddTest(
		"test_invalid_select_path",
		"Invalid $select path returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// Try to select a non-existent property
			resp, err := ctx.GET("/Products?$select=NonExistentProperty")
			if err != nil {
				return err
			}

			// Should return 400 for invalid property
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	// $filter referencing a property that does not exist must return 400 per OData §11.2.5.1.
	// The service cannot evaluate an expression against an unknown property path.
	suite.AddTest(
		"test_filter_nonexistent_property",
		"$filter on nonexistent property returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("NonExistentProperty eq 'X'"))
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	// $orderby referencing a nonexistent property must return 400.
	suite.AddTest(
		"test_orderby_nonexistent_property",
		"$orderby on nonexistent property returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=" + url.QueryEscape("NonExistentProperty asc"))
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	// A date/time filter function applied to a property whose CSDL-declared
	// type is Edm.String is a literal/function type mismatch and MUST be
	// rejected with 400 per the $filter grammar (OData §5.1.1.6/§11.2.5.1),
	// not silently evaluated to an empty (or any) result.
	// Filed and fixed as NLstn/go-odata#800.
	suite.AddTest(
		"test_year_function_on_string_property_rejected",
		"year() applied to an Edm.String property returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			isString, err := entityTypeHasStringProperty(ctx, "Product", "Name")
			if err != nil {
				return err
			}
			if !isString {
				return ctx.Skip("Product.Name is not declared Edm.String in this model")
			}
			resp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("year(Name) eq 2024"))
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	// An unquoted date literal compared against an Edm.String property is a
	// literal/property type mismatch and MUST be rejected with 400, not
	// silently evaluated as if the literal were a string.
	// Filed and fixed as NLstn/go-odata#800.
	suite.AddTest(
		"test_date_literal_against_string_property_rejected",
		"Unquoted date literal compared to an Edm.String property returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			isString, err := entityTypeHasStringProperty(ctx, "Product", "Name")
			if err != nil {
				return err
			}
			if !isString {
				return ctx.Skip("Product.Name is not declared Edm.String in this model")
			}
			resp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("Name eq 2024-01-15"))
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	suite.AddTest(
		"test_duplicate_system_query_option_rejected",
		"A system query option appearing more than once returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price%20gt%2010&$filter=Price%20lt%201000")
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	return suite
}
