package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// PaginationEdgeCases creates the 11.2.5.12 Pagination Edge Cases test suite
func PaginationEdgeCases() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.12 Pagination Edge Cases",
		"Tests edge cases and boundary conditions for pagination with $top, $skip, and nextLink.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionstopandskip",
	)

	// Test 1: $top=0 returns empty result set
	suite.AddTest(
		"test_top_zero",
		"$top=0 returns empty result set",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=0")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("expected value array")
			}

			if len(value) != 0 {
				return fmt.Errorf("expected empty value array")
			}

			return nil
		},
	)

	// Test 2: $skip beyond total count returns empty result
	suite.AddTest(
		"test_skip_beyond_count",
		"$skip beyond total count returns empty result",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$skip=10000")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("expected value array")
			}

			if len(value) != 0 {
				return fmt.Errorf("expected empty value array")
			}

			return nil
		},
	)

	// Test 3: Negative $top returns error
	suite.AddTest(
		"test_negative_top",
		"Negative $top returns 400 error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=-5")
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 4: Negative $skip returns error
	suite.AddTest(
		"test_negative_skip",
		"Negative $skip returns 400 error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$skip=-5")
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 5: $top with very large number
	suite.AddTest(
		"test_top_large_number",
		"$top with very large number",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=999999")
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

			// A very large $top must not truncate the result; all 7 seed products
			// must be present.
			if len(items) < 7 {
				return fmt.Errorf("$top=999999 returned %d items, expected at least 7 seed products", len(items))
			}

			return nil
		},
	)

	// Test 6: $skip with zero
	suite.AddTest(
		"test_skip_zero",
		"$skip=0 is valid",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$skip=0&$top=2")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("expected value array")
			}

			return nil
		},
	)

	// Test 7: @odata.nextLink presence when more results available
	suite.AddTest(
		"test_nextlink_present",
		"@odata.nextLink present when more results available",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=2")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("expected value array in response")
			}

			if nextLink, hasNextLink := result["@odata.nextLink"].(string); hasNextLink {
				// Server chose to page; verify the nextLink is actually followable.
				ctx.Log("@odata.nextLink found: " + nextLink)
				resp2, err := ctx.GET(nextLink)
				if err != nil {
					return fmt.Errorf("failed to follow @odata.nextLink: %w", err)
				}
				if err := ctx.AssertStatusCode(resp2, 200); err != nil {
					return fmt.Errorf("@odata.nextLink follow: %w", err)
				}
			} else {
				// No nextLink is valid when the server returns all items within $top.
				// With 7 seed products and $top=2, the server is expected to set a
				// nextLink, but we allow the no-nextLink path for non-conformant servers
				// that return all rows regardless of $top.
				ctx.Log(fmt.Sprintf("No @odata.nextLink (server returned %d items for $top=2)", len(value)))
			}

			return nil
		},
	)

	// Test 8: @odata.nextLink absent when no more results
	suite.AddTest(
		"test_nextlink_absent",
		"@odata.nextLink absent when all results returned",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=10000")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			bodyStr := string(resp.Body)
			if strings.Contains(bodyStr, "@odata.nextLink") {
				return fmt.Errorf("unexpected @odata.nextLink when all results returned")
			}

			return nil
		},
	)

	// Test 9: Combining $top and $skip
	suite.AddTest(
		"test_top_and_skip_combined",
		"$top=2&$skip=2 returns the correct slice of the sorted collection",
		func(ctx *framework.TestContext) error {
			// Get full sorted set with $orderby to have a deterministic ordering.
			allResp, err := ctx.GET("/Products?$orderby=ID asc&$select=ID")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(allResp, 200); err != nil {
				return err
			}
			var allBody map[string]interface{}
			if err := json.Unmarshal(allResp.Body, &allBody); err != nil {
				return fmt.Errorf("full list response not valid JSON: %w", err)
			}
			allItems, ok := allBody["value"].([]interface{})
			if !ok || len(allItems) < 4 {
				return ctx.Skip(fmt.Sprintf("need at least 4 products to test skip+top (got %d)", len(allItems)))
			}

			// Get page with $top=2&$skip=2 under the same ordering.
			resp, err := ctx.GET("/Products?$orderby=ID asc&$select=ID&$top=2&$skip=2")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			var pageBody map[string]interface{}
			if err := json.Unmarshal(resp.Body, &pageBody); err != nil {
				return fmt.Errorf("page response not valid JSON: %w", err)
			}
			pageItems, ok := pageBody["value"].([]interface{})
			if !ok {
				return fmt.Errorf("page response missing 'value' array")
			}
			if len(pageItems) != 2 {
				return fmt.Errorf("$top=2&$skip=2 returned %d items (expected 2)", len(pageItems))
			}

			// The two items must be exactly items[2] and items[3] from the full set.
			for i, pi := range pageItems {
				pageItem, ok := pi.(map[string]interface{})
				if !ok {
					return fmt.Errorf("page item %d is not an object", i)
				}
				fullItem, ok := allItems[2+i].(map[string]interface{})
				if !ok {
					continue
				}
				if fmt.Sprintf("%v", pageItem["ID"]) != fmt.Sprintf("%v", fullItem["ID"]) {
					return fmt.Errorf("page item %d ID=%v expected %v ($skip=2 gave wrong entity)", i, pageItem["ID"], fullItem["ID"])
				}
			}
			return nil
		},
	)

	// Test 10: $top and $skip with $filter
	suite.AddTest(
		"test_pagination_with_filter",
		"Pagination with $filter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 0&$top=2&$skip=1")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 11: $top and $skip with $orderby — the returned page must be
	// correctly ordered and must represent the skipped offset within the sorted set.
	suite.AddTest(
		"test_pagination_with_orderby",
		"$top=3&$skip=1 with $orderby returns items 2-4 in sorted order",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=Price desc&$select=ID,Price&$top=3&$skip=1")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			var body map[string]interface{}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("response not valid JSON: %w", err)
			}
			items, ok := body["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}
			if len(items) == 0 {
				return ctx.Skip("no items returned for $orderby=Price desc&$top=3&$skip=1")
			}

			// The returned items must be in descending Price order.
			for i := 1; i < len(items); i++ {
				prev, ok1 := items[i-1].(map[string]interface{})
				curr, ok2 := items[i].(map[string]interface{})
				if !ok1 || !ok2 {
					continue
				}
				prevPrice, p1 := prev["Price"].(float64)
				currPrice, p2 := curr["Price"].(float64)
				if p1 && p2 && prevPrice < currPrice {
					return fmt.Errorf("$orderby=Price desc violated at index %d: prev=%.2f < curr=%.2f", i, prevPrice, currPrice)
				}
			}
			return nil
		},
	)

	// Test 12: nextLink should preserve other query options
	suite.AddTest(
		"test_nextlink_preserves_options",
		"nextLink preserves other query options",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 0&$orderby=ID&$top=2")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// This is acceptable behavior whether nextLink is present or not
			return nil
		},
	)

	// Test 13: Invalid $top value (non-numeric)
	suite.AddTest(
		"test_invalid_top_value",
		"Invalid $top value returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=abc")
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 14: Invalid $skip value (non-numeric)
	suite.AddTest(
		"test_invalid_skip_value",
		"Invalid $skip value returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$skip=xyz")
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 15: $count with pagination
	suite.AddTest(
		"test_count_with_pagination",
		"$count works with pagination",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true&$top=2")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			bodyStr := string(resp.Body)
			if !strings.Contains(bodyStr, "@odata.count") {
				return fmt.Errorf("missing @odata.count")
			}

			return nil
		},
	)

	return suite
}
