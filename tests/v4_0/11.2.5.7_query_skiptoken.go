package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// QuerySkiptoken creates the 11.2.5.7 $skiptoken Query Option test suite
func QuerySkiptoken() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.7 $skiptoken",
		"Tests server-driven paging with $skiptoken for continuation of result sets.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_SystemQueryOptionskiptoken",
	)

	// Test 1: Response with @odata.nextLink includes skiptoken; follow it and
	// verify that the two pages together cover all 7 seed products.
	suite.AddTest(
		"test_nextlink_has_skiptoken",
		"@odata.nextLink contains $skiptoken parameter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			page1Items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}

			nextLink, hasNextLink := result["@odata.nextLink"].(string)
			if !hasNextLink {
				// Server returned all items without a nextLink; valid for small collections.
				// Assert we still received all 7 seed products in one shot.
				if len(page1Items) != 7 {
					return fmt.Errorf("no @odata.nextLink on $top=3 and page returned %d items (expected 7 for full single-page response)", len(page1Items))
				}
				ctx.Log("No @odata.nextLink — server returned all 7 products in a single page (conformant)")
				return nil
			}

			ctx.Log("@odata.nextLink found: " + nextLink)

			// Follow the nextLink and verify it is usable.
			resp2, err := ctx.GET(nextLink)
			if err != nil {
				return fmt.Errorf("failed to follow @odata.nextLink: %w", err)
			}
			if err := ctx.AssertStatusCode(resp2, 200); err != nil {
				return fmt.Errorf("@odata.nextLink response: %w", err)
			}

			page2Items, err := ctx.ParseEntityCollection(resp2)
			if err != nil {
				return fmt.Errorf("@odata.nextLink response body: %w", err)
			}
			if len(page2Items) == 0 {
				return fmt.Errorf("@odata.nextLink response has an empty value array; expected at least one item on the second page")
			}

			// Build ID sets for both pages; they must be disjoint.
			page1IDs := make(map[interface{}]bool, len(page1Items))
			for _, item := range page1Items {
				page1IDs[item["ID"]] = true
			}
			for _, item := range page2Items {
				if page1IDs[item["ID"]] {
					return fmt.Errorf("item ID=%v appears on both page 1 and page 2; $skiptoken did not advance correctly", item["ID"])
				}
			}

			// Combined count must equal total seed products (7).
			combined := len(page1Items) + len(page2Items)
			if combined != 7 {
				// Allow the combined total to exceed 7 if the server itself has additional
				// data, but it must never be less than 7.
				if combined < 7 {
					return fmt.Errorf("combined pages contain %d items, expected 7 total seed products", combined)
				}
			}

			return nil
		},
	)

	// Test 2: Invalid skiptoken returns error
	suite.AddTest(
		"test_invalid_skiptoken",
		"Invalid $skiptoken returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$skiptoken=invalid_token_xyz")
			if err != nil {
				return err
			}

			// Should return 400 for invalid skiptoken
			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	return suite
}
