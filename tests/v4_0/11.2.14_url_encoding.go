package v4_0

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// URLEncoding creates the 11.2.14 URL Encoding test suite
func URLEncoding() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.14 URL Encoding",
		"Tests proper handling of URL encoding in resource paths, query parameters, and special characters according to RFC 3986.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html",
	)

	// Test 1: String literal with spaces in filter
	suite.AddTest(
		"test_filter_with_spaces",
		"Filter with URL-encoded spaces",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Name eq 'Gaming Laptop'", func(p map[string]interface{}) bool {
				return productString(p, "Name") == "Gaming Laptop"
			})
		},
	)

	// Test 2: String literal with special characters in filter
	suite.AddTest(
		"test_filter_special_chars",
		"Filter with encoded special characters (&)",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "contains(Name,'&')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "&")
			})
		},
	)

	// Test 3: Query option with encoded $ symbol
	suite.AddTest(
		"test_encoded_dollar_sign",
		"Percent-encoded dollar sign in query key is handled and $top is actually applied",
		func(ctx *framework.TestContext) error {
			all, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}

			// HTTP percent-decoding happens before OData processing: %24top is decoded
			// to $top by the HTTP layer, so the OData layer sees a valid $top=5 option.
			resp, err := ctx.GET("/Products?%24top=5")
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

			wantCount := len(all)
			if wantCount > 5 {
				wantCount = 5
			}
			if len(items) != wantCount {
				return fmt.Errorf("%%24top=5 should decode to $top=5: expected %d item(s), got %d", wantCount, len(items))
			}
			return nil
		},
	)

	// Test 4: Filter with URL-encoded operators
	suite.AddTest(
		"test_encoded_operators",
		"Filter with URL-encoded operators",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price gt 50 and Price lt 200", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > 50 && price < 200
			})
		},
	)

	// Test 5: String literal with single quote (escaped)
	suite.AddTest(
		"test_filter_single_quote",
		"Filter with single quote in string literal",
		func(ctx *framework.TestContext) error {
			// Single quotes in OData string literals are escaped by doubling them.
			return assertProductFilter(ctx, "contains(Name,'''')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "'")
			})
		},
	)

	// Test 6: Parentheses in filter expressions
	suite.AddTest(
		"test_parentheses_encoding",
		"Filter with URL-encoded parentheses",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "(Price gt 100) and (Status lt 10)", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				if !ok {
					return false
				}
				status, err := enumStatusValue(p)
				return err == nil && price > 100 && status < 10
			})
		},
	)

	// Test 7: Mixed encoded and unencoded characters
	suite.AddTest(
		"test_mixed_encoding",
		"Mixed encoded and unencoded parameters",
		func(ctx *framework.TestContext) error {
			all, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}
			expected := 0
			for _, p := range all {
				if price, ok := productFloat(p, "Price"); ok && price > 50 {
					expected++
				}
			}

			path := "/Products?$filter=" + url.QueryEscape("Price gt 50") + "&$top=10"
			resp, err := ctx.GET(path)
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

			wantCount := expected
			if wantCount > 10 {
				wantCount = 10
			}
			if len(items) != wantCount {
				return fmt.Errorf("expected %d item(s) for $filter=Price gt 50&$top=10, got %d", wantCount, len(items))
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Price gt 50", func(entity map[string]interface{}) (bool, string) {
				price, ok := productFloat(entity, "Price")
				if !ok || price <= 50 {
					return false, fmt.Sprintf("entity does not satisfy Price gt 50, got Price=%v", entity["Price"])
				}
				return true, ""
			})
		},
	)

	// Test 8: Plus sign encoding
	suite.AddTest(
		"test_plus_sign",
		"Plus sign handling in URL",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price gt 0", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > 0
			})
		},
	)

	// Test 9: Percent encoding in string literals
	suite.AddTest(
		"test_percent_in_string",
		"Percent sign in filter string",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "contains(Name,'%')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "%")
			})
		},
	)

	// Test 10: Reserved characters in query string
	suite.AddTest(
		"test_reserved_chars",
		"A reserved character used where an operator is expected is rejected as invalid filter syntax",
		func(ctx *framework.TestContext) error {
			// A semicolon is not a valid $filter connective (spec requires 'and'/'or');
			// this must be rejected as a syntax error, not silently accepted.
			path := "/Products?$filter=" + url.QueryEscape("Price gt 10;id lt 100")
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	// Test 11: Unicode characters in filter
	suite.AddTest(
		"test_unicode_characters",
		"Unicode characters in filter",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "contains(Name,'é')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "é")
			})
		},
	)

	suite.AddTest(
		"test_encoded_ampersand_stays_in_string_literal",
		"Encoded ampersand inside a string literal is not treated as a query separator",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Name eq 'Laptop & Mouse'", func(p map[string]interface{}) bool {
				return productString(p, "Name") == "Laptop & Mouse"
			})
		},
	)

	suite.AddTest(
		"test_encoded_slash_stays_in_string_literal",
		"Encoded slash inside a string literal is not treated as a path separator",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Name eq 'Laptop/Tablet'", func(p map[string]interface{}) bool {
				return productString(p, "Name") == "Laptop/Tablet"
			})
		},
	)

	suite.AddTest(
		"test_encoded_comma_in_select_list",
		"Encoded comma separates $select properties correctly",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=ID%2CName&$top=1")
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
			if err := ctx.AssertEntityHasFields(items[0], "ID", "Name"); err != nil {
				return err
			}
			return ctx.AssertEntityOnlyAllowedFields(items[0], "ID", "Name")
		},
	)

	return suite
}
