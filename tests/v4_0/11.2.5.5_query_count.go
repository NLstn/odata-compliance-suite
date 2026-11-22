package v4_0

import (
	"encoding/json"
	"fmt"
	"math"
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

			return assertCountMatchesCollection(ctx, resp)
		},
	)

	// Test 2: $count=false is a SHOULD NOT hint, not a prohibition.
	suite.AddTest(
		"test_count_false",
		"$count=false returns a valid collection (count is a SHOULD NOT hint)",
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

			if _, ok := result["value"].([]interface{}); !ok {
				return fmt.Errorf("response missing value array")
			}
			if count, ok := result["@odata.count"]; ok {
				if number, ok := count.(float64); !ok || number < 0 || number != float64(int64(number)) {
					return fmt.Errorf("invalid optional @odata.count: %v", count)
				}
			}
			return nil
		},
	)

	// Test 3: $count with $filter returns filtered count
	suite.AddTestWithCapabilities(
		"test_count_with_filter",
		"$count with $filter returns filtered count",
		[]framework.RequiredCapability{framework.Require(framework.CapFilter, "Products")},
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
			count, err := readInlineCount(ctx, resp)
			if err != nil {
				return err
			}

			items, err := collectEntityPages(ctx, resp)
			if err != nil {
				return err
			}

			// Cross-check @odata.count against the independently computed oracle
			// total, not merely against len(value) — a server that filters wrong
			// but reports a self-consistent count would otherwise still pass.
			if count != float64(len(expected)) {
				return fmt.Errorf("@odata.count=%v but %d product(s) actually satisfy Price gt 100", count, len(expected))
			}
			if len(items) != len(expected) {
				return fmt.Errorf("response contains %d item(s) but %d product(s) actually satisfy Price gt 100", len(items), len(expected))
			}

			// Verify the returned set is exactly the expected set (soundness + completeness).
			got := map[string]bool{}
			for i, item := range items {
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
	suite.AddTestWithCapabilities(
		"test_count_with_search",
		"$count with $search returns search-filtered count",
		[]framework.RequiredCapability{framework.Require(framework.CapSearch, "Products")},
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
			countFloat, err := readInlineCount(ctx, resp)
			if err != nil {
				return err
			}

			items, err := collectEntityPages(ctx, resp)
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return fmt.Errorf("expected at least one search result for 'Laptop'")
			}

			if countFloat != float64(len(items)) {
				return fmt.Errorf("count=%v but response contains %d items", countFloat, len(items))
			}

			got := map[string]bool{}
			for _, item := range items {
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

	for _, tc := range []struct {
		name, description, query string
		required                 []framework.RequiredCapability
		wantItems                int
	}{
		{"test_count_with_top", "$count includes entities outside $top", "&$top=1",
			[]framework.RequiredCapability{framework.Require(framework.CapTop, "Products")}, 1},
		{"test_count_with_top_zero", "$count includes entities when $top=0 returns no items", "&$top=0",
			[]framework.RequiredCapability{framework.Require(framework.CapTop, "Products")}, 0},
		{"test_count_with_skip", "$count includes entities excluded by $skip", "&$skip=2&$top=1",
			[]framework.RequiredCapability{framework.Require(framework.CapSkip, "Products"), framework.Require(framework.CapTop, "Products")}, 1},
	} {
		suite.AddTestWithCapabilities(tc.name, tc.description, tc.required, func(ctx *framework.TestContext) error {
			all, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}
			if len(all) < 3 {
				return fmt.Errorf("expected at least three seeded Products, got %d", len(all))
			}
			return assertCountAndSize(ctx, "/Products?$count=true"+tc.query, len(all), tc.wantItems)
		})
	}

	suite.AddTestWithCapabilities(
		"test_count_with_no_matches",
		"$count=true includes a zero count for an empty filtered collection",
		[]framework.RequiredCapability{framework.Require(framework.CapFilter, "Products")},
		func(ctx *framework.TestContext) error {
			return assertCountAndSize(ctx, "/Products?$count=true&$filter=false", 0, 0)
		},
	)

	return suite
}

func assertCountMatchesCollection(ctx *framework.TestContext, resp *framework.HTTPResponse) error {
	count, err := readInlineCount(ctx, resp)
	if err != nil {
		return err
	}
	items, err := collectEntityPages(ctx, resp)
	if err != nil {
		return err
	}
	if count != float64(len(items)) {
		return fmt.Errorf("@odata.count=%v but the collection contains %d entities across all pages", count, len(items))
	}
	return nil
}

func assertCountAndSize(ctx *framework.TestContext, path string, wantCount, wantItems int) error {
	resp, err := ctx.GET(path)
	if err != nil {
		return err
	}
	count, err := readInlineCount(ctx, resp)
	if err != nil {
		return err
	}
	if count != float64(wantCount) {
		return fmt.Errorf("@odata.count=%v, want %d", count, wantCount)
	}
	items, err := collectEntityPages(ctx, resp)
	if err != nil {
		return err
	}
	if len(items) != wantItems {
		return fmt.Errorf("collection contains %d entities, want %d across all pages", len(items), wantItems)
	}
	return nil
}

func readInlineCount(ctx *framework.TestContext, resp *framework.HTTPResponse) (float64, error) {
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return 0, err
	}
	var payload map[string]interface{}
	if err := ctx.GetJSON(resp, &payload); err != nil {
		return 0, err
	}
	count, ok := payload["@odata.count"].(float64)
	if !ok || count < 0 || math.Trunc(count) != count {
		return 0, fmt.Errorf("expected a non-negative integer @odata.count, got %v", payload["@odata.count"])
	}
	return count, nil
}
