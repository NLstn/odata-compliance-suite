package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// QueryTopSkip creates the 11.2.5.3 and 11.2.5.4 System Query Options $top and $skip test suite
func QueryTopSkip() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.3 System Query Options $top and $skip",
		"Tests $top and $skip query options for paging according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionstopandskip",
	)
	topCap := []framework.RequiredCapability{framework.Require(framework.CapTop, "Products")}
	skipCap := []framework.RequiredCapability{framework.Require(framework.CapSkip, "Products")}
	topSkipCaps := []framework.RequiredCapability{
		framework.Require(framework.CapTop, "Products"),
		framework.Require(framework.CapSkip, "Products"),
	}

	// Test 1: $top returns exactly min($top, total) items
	suite.AddTestWithCapabilities(
		"test_top_limit",
		"$top returns exactly min($top, total) items",
		topCap,
		func(ctx *framework.TestContext) error {
			total, err := collectionSize(ctx, "/Products")
			if err != nil {
				return err
			}

			const top = 2
			resp, err := ctx.GET(fmt.Sprintf("/Products?$top=%d", top))
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
				return fmt.Errorf("response missing 'value' array")
			}
			count := len(value)

			expected := top
			if total < top {
				expected = total
			}

			// $top must never return more than the limit.
			if count > top {
				return fmt.Errorf("returned %d items, $top=%d must return at most %d", count, top, top)
			}
			// It must return exactly the expected page size, unless the service is
			// applying server-driven paging (which it signals with @odata.nextLink).
			if count != expected {
				if _, paged := result["@odata.nextLink"]; paged && count < expected {
					return nil
				}
				return fmt.Errorf("returned %d items for $top=%d with total=%d; expected %d", count, top, total, expected)
			}

			return nil
		},
	)

	// Test 2: $skip skips the specified number of items
	suite.AddTestWithCapabilities(
		"test_skip_items",
		"$skip skips items",
		skipCap,
		func(ctx *framework.TestContext) error {
			// First get all products to know what to expect
			allResp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if allResp.StatusCode != 200 {
				return fmt.Errorf("failed to get all products: status %d", allResp.StatusCode)
			}

			var allResult map[string]interface{}
			if err := json.Unmarshal(allResp.Body, &allResult); err != nil {
				return fmt.Errorf("failed to parse all products JSON: %w", err)
			}

			allValue, ok := allResult["value"].([]interface{})
			if !ok || len(allValue) < 2 {
				// Not enough products to test $skip
				return nil
			}

			// Get the second item's ID from the full list
			secondItem, ok := allValue[1].(map[string]interface{})
			if !ok {
				return fmt.Errorf("second item is not an object")
			}
			secondID := secondItem["ID"]

			// Now test $skip=1&$top=1 - should return the second item
			resp, err := ctx.GET("/Products?$skip=1&$top=1")
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
				return fmt.Errorf("response missing 'value' array")
			}

			// Verify we got exactly 1 item
			if len(value) != 1 {
				return fmt.Errorf("expected exactly 1 item, got %d", len(value))
			}

			// Verify it's the second item (ID should match)
			item, ok := value[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("returned item is not an object")
			}
			returnedID := item["ID"]

			if fmt.Sprint(returnedID) != fmt.Sprint(secondID) {
				return fmt.Errorf("expected ID=%v (2nd item), got ID=%v", secondID, returnedID)
			}

			return nil
		},
	)

	// Test 3: $top=0 returns empty collection
	suite.AddTestWithCapabilities(
		"test_top_zero",
		"$top=0 returns empty collection",
		topCap,
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
				return fmt.Errorf("response missing 'value' array")
			}

			if len(value) != 0 {
				return fmt.Errorf("expected 0 items, got %d", len(value))
			}

			return nil
		},
	)

	// Test 4: Combine $skip and $top for paging, verifying the exact page slice
	suite.AddTestWithCapabilities(
		"test_skip_top_paging",
		"$skip and $top return the exact ordered page",
		topSkipCaps,
		func(ctx *framework.TestContext) error {
			// Use $orderby=ID so the ordering is deterministic and the page can be
			// compared against the corresponding slice of the full ordered list.
			allIDs, err := orderedProductIDs(ctx, "/Products?$orderby=ID")
			if err != nil {
				return err
			}
			totalCount := len(allIDs)

			const skip, top = 2, 3
			resp, err := ctx.GET(fmt.Sprintf("/Products?$orderby=ID&$skip=%d&$top=%d", skip, top))
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			pageIDs, err := pageProductIDs(resp)
			if err != nil {
				return err
			}

			// Expected slice: full ordered IDs[skip : skip+top] (clamped to length).
			start := skip
			if start > totalCount {
				start = totalCount
			}
			end := start + top
			if end > totalCount {
				end = totalCount
			}
			expected := allIDs[start:end]

			if len(pageIDs) != len(expected) {
				return fmt.Errorf("$skip=%d&$top=%d returned %d items, expected %d (total=%d)", skip, top, len(pageIDs), len(expected), totalCount)
			}
			for i := range expected {
				if pageIDs[i] != expected[i] {
					return fmt.Errorf("page item %d has ID %q, expected %q (page does not match the ordered slice)", i, pageIDs[i], expected[i])
				}
			}

			return nil
		},
	)

	return suite
}

// collectionSize returns the number of entities in a collection using the
// $count=true annotation (the true total, independent of server-driven paging).
func collectionSize(ctx *framework.TestContext, path string) (int, error) {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	resp, err := ctx.GET(path + sep + "$count=true")
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("expected status 200 fetching count, got %d", resp.StatusCode)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %w", err)
	}
	count, ok := result["@odata.count"].(float64)
	if !ok {
		return 0, fmt.Errorf("@odata.count missing or not a number")
	}
	return int(count), nil
}

// orderedProductIDs fetches the full ordered list of product IDs, following any
// server-driven paging next links so the complete order is captured.
func orderedProductIDs(ctx *framework.TestContext, path string) ([]string, error) {
	ids := []string{}
	next := path
	for next != "" {
		resp, err := ctx.GET(next)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		pageIDs, err := pageProductIDs(resp)
		if err != nil {
			return nil, err
		}
		ids = append(ids, pageIDs...)

		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
		nextLink, _ := result["@odata.nextLink"].(string)
		next = ""
		if nextLink != "" {
			next = nextLink
			if strings.HasPrefix(next, ctx.ServerURL()) {
				next = strings.TrimPrefix(next, ctx.ServerURL())
			}
		}
	}
	return ids, nil
}

// pageProductIDs extracts the ID values, in order, from a single response page.
func pageProductIDs(resp *framework.HTTPResponse) ([]string, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	value, ok := result["value"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("response missing 'value' array")
	}
	ids := make([]string, 0, len(value))
	for i, v := range value {
		item, ok := v.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("item %d is not an object", i)
		}
		ids = append(ids, fmt.Sprint(item["ID"]))
	}
	return ids, nil
}
