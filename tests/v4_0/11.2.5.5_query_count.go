package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// QueryCount creates the 11.2.5.5 System Query Option $count test suite
func QueryCount() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.5 System Query Option $count",
		"Tests $count query option according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptioncount",
	)

	// Test 1: $count=true includes @odata.count and reports the true collection size
	suite.AddTest(
		"test_count_true",
		"$count=true includes @odata.count matching the collection size",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true")
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

			// @odata.count must be present and a non-negative integer.
			countVal, ok := result["@odata.count"]
			if !ok {
				return fmt.Errorf("@odata.count field is missing")
			}
			countFloat, ok := countVal.(float64)
			if !ok {
				return fmt.Errorf("@odata.count is not a number, got %T", countVal)
			}
			count := int(countFloat)
			if count < 0 {
				return fmt.Errorf("@odata.count must be non-negative, got %d", count)
			}

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			// @odata.count is the TOTAL count and must never be smaller than the
			// number of items in the current page.
			if count < len(value) {
				return fmt.Errorf("@odata.count=%d is smaller than the %d items returned", count, len(value))
			}
			// Without a next link the full collection is in this page, so the count
			// must equal the number of items returned (Part 1 §11.2.5.5).
			if _, paged := result["@odata.nextLink"]; !paged && count != len(value) {
				return fmt.Errorf("@odata.count=%d but the (unpaged) response contains %d items", count, len(value))
			}

			return nil
		},
	)

	// Test 2: $count=false does not include @odata.count
	suite.AddTest(
		"test_count_false",
		"$count=false excludes @odata.count",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=false")
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

			// Verify @odata.count is NOT present
			if _, ok := result["@odata.count"]; ok {
				return fmt.Errorf("@odata.count should not be present when $count=false")
			}

			return nil
		},
	)

	// Test 3: $count with $filter returns filtered count
	suite.AddTest(
		"test_count_with_filter",
		"$count with $filter returns filtered count",
		func(ctx *framework.TestContext) error {
			all, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}
			expected := map[string]bool{}
			for _, p := range all {
				if price, ok := productFloat(p, "Price"); ok && price > 100 {
					expected[productID(p)] = true
				}
			}

			resp, err := ctx.GET("/Products?$count=true&$filter=Price%20gt%20100")
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

			countVal, ok := result["@odata.count"]
			if !ok {
				return fmt.Errorf("@odata.count field is missing")
			}
			count, ok := countVal.(float64)
			if !ok {
				return fmt.Errorf("@odata.count is not a number")
			}

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			// Cross-check @odata.count against the independently computed oracle
			// total, not merely against len(value) — a server that filters wrong
			// but reports a self-consistent count would otherwise still pass.
			if int(count) != len(expected) {
				return fmt.Errorf("@odata.count=%d but %d product(s) actually satisfy Price gt 100", int(count), len(expected))
			}
			if len(value) != len(expected) {
				return fmt.Errorf("response contains %d item(s) but %d product(s) actually satisfy Price gt 100", len(value), len(expected))
			}

			// Verify the returned set is exactly the expected set (soundness + completeness).
			got := map[string]bool{}
			for i, v := range value {
				item, ok := v.(map[string]interface{})
				if !ok {
					return fmt.Errorf("item %d is not an object", i)
				}
				got[productID(item)] = true
				price, ok := item["Price"].(float64)
				if !ok {
					return fmt.Errorf("item %d missing Price field or not a number", i)
				}
				if price <= 100 {
					return fmt.Errorf("found item with Price=%.2f which is not > 100", price)
				}
			}
			for id := range expected {
				if !got[id] {
					return fmt.Errorf("product %s satisfies Price gt 100 but was not returned", id)
				}
			}

			return nil
		},
	)

	// Test 4: $count with $search returns search-filtered count
	suite.AddTest(
		"test_count_with_search",
		"$count with $search returns search-filtered count",
		func(ctx *framework.TestContext) error {
			all, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}
			// The spec doesn't mandate which fields $search covers, so this
			// establishes a lower bound only: every product whose Name matches
			// must be included, even if the service legitimately searches other
			// fields too and returns a superset.
			mustMatch := map[string]bool{}
			for _, p := range all {
				if strings.Contains(strings.ToLower(productString(p, "Name")), "laptop") {
					mustMatch[productID(p)] = true
				}
			}

			resp, err := ctx.GET("/Products?$count=true&$search=Laptop")
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

			countVal, ok := result["@odata.count"]
			if !ok {
				return fmt.Errorf("@odata.count field is missing")
			}

			countFloat, ok := countVal.(float64)
			if !ok {
				return fmt.Errorf("@odata.count is not a number")
			}

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			if len(value) == 0 {
				return fmt.Errorf("expected at least one search result for 'Laptop'")
			}

			if int(countFloat) != len(value) {
				return fmt.Errorf("count=%d but response contains %d items", int(countFloat), len(value))
			}

			got := map[string]bool{}
			for i, v := range value {
				item, ok := v.(map[string]interface{})
				if !ok {
					return fmt.Errorf("item %d is not an object", i)
				}
				got[productID(item)] = true
			}
			for id := range mustMatch {
				if !got[id] {
					return fmt.Errorf("product %s has 'Laptop' in its Name but was not returned by $search=Laptop", id)
				}
			}

			return nil
		},
	)

	// Test 5: $count with $top still returns total count
	suite.AddTest(
		"test_count_with_top",
		"$count with $top returns total count, not page count",
		func(ctx *framework.TestContext) error {
			// First get total count without $top
			totalResp, err := ctx.GET("/Products?$count=true")
			if err != nil {
				return err
			}
			if totalResp.StatusCode != 200 {
				return fmt.Errorf("failed to get total count: status %d", totalResp.StatusCode)
			}

			var totalResult map[string]interface{}
			if err := json.Unmarshal(totalResp.Body, &totalResult); err != nil {
				return fmt.Errorf("failed to parse total count JSON: %w", err)
			}

			totalCountVal, ok := totalResult["@odata.count"]
			if !ok {
				return fmt.Errorf("total response missing @odata.count")
			}
			totalCountFloat, ok := totalCountVal.(float64)
			if !ok {
				return fmt.Errorf("@odata.count is not a number")
			}
			totalCount := int(totalCountFloat)

			// Now get with $top=1
			resp, err := ctx.GET("/Products?$count=true&$top=1")
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

			// Verify @odata.count is present
			countVal, ok := result["@odata.count"]
			if !ok {
				return fmt.Errorf("@odata.count field is missing")
			}

			countFloat, ok := countVal.(float64)
			if !ok {
				return fmt.Errorf("@odata.count is not a number")
			}
			count := int(countFloat)

			// Get the items
			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			items := len(value)

			// The @odata.count should be the TOTAL count, not the page count
			if count != totalCount {
				return fmt.Errorf("count=%d but total count is %d (should match total, not page size)", count, totalCount)
			}

			// The number of items in the response should be 1 (due to $top=1)
			if items != 1 {
				return fmt.Errorf("expected 1 item in response (due to $top=1), got %d", items)
			}

			// The count should be greater than or equal to the items
			if count < items {
				return fmt.Errorf("count=%d should be >= items=%d", count, items)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_count_with_skip",
		"$count with $skip returns the total count before paging",
		func(ctx *framework.TestContext) error {
			baseResp, err := ctx.GET("/Products?$count=true")
			if err != nil {
				return err
			}
			var base map[string]interface{}
			if err := json.Unmarshal(baseResp.Body, &base); err != nil {
				return fmt.Errorf("failed to parse baseline count: %w", err)
			}
			total, ok := base["@odata.count"].(float64)
			if !ok {
				return fmt.Errorf("baseline response missing numeric @odata.count")
			}

			resp, err := ctx.GET("/Products?$count=true&$skip=2&$top=1")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse paged count: %w", err)
			}
			count, ok := result["@odata.count"].(float64)
			if !ok || count != total {
				return fmt.Errorf("@odata.count with $skip = %v, want total %v", result["@odata.count"], total)
			}
			return nil
		},
	)

	return suite
}
