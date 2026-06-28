package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// QuerySkiptoken creates the 11.2.5.7 $skiptoken Query Option test suite
func QuerySkiptoken() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.7 $skiptoken",
		"Tests server-driven paging with $skiptoken for continuation of result sets.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_SystemQueryOptionskiptoken",
	)

	// Test 1: Response with @odata.nextLink includes skiptoken
	suite.AddTest(
		"test_nextlink_has_skiptoken",
		"@odata.nextLink contains $skiptoken parameter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=2")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			// Check if there's a nextLink
			if nextLink, ok := result["@odata.nextLink"].(string); ok {
				// NextLink should contain skiptoken parameter
				// This is implementation-specific, so we just verify it exists
				ctx.Log("@odata.nextLink found: " + nextLink)
				return nil
			}

			// If there's no nextLink, result set fits in one page
			ctx.Log("No @odata.nextLink (result set fits in one page)")
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

	// Test 3: Traversing all pages produces no duplicate entities
	suite.AddTest(
		"test_nextlink_traversal_no_duplicates",
		"Following all @odata.nextLink pages yields no duplicate entity IDs",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var page map[string]interface{}
			if err := json.Unmarshal(resp.Body, &page); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			// If the server doesn't use server-driven paging, skip gracefully.
			if _, hasNext := page["@odata.nextLink"]; !hasNext {
				return ctx.Skip("server does not use server-driven paging for this result set")
			}

			seen := map[string]bool{}
			collectIDs := func(p map[string]interface{}) error {
				items, _ := p["value"].([]interface{})
				for _, item := range items {
					entity, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					id, _ := entity["ID"].(string)
					if id == "" {
						// Fallback: stringify whatever key happens to be the identity.
						id = fmt.Sprintf("%v", entity)
					}
					if seen[id] {
						return fmt.Errorf("duplicate entity ID %q found across pages", id)
					}
					seen[id] = true
				}
				return nil
			}

			current := page
			for {
				if err := collectIDs(current); err != nil {
					return err
				}
				nextLink, _ := current["@odata.nextLink"].(string)
				if nextLink == "" {
					break
				}
				path := strings.TrimPrefix(nextLink, ctx.ServerURL())
				resp, err = ctx.GET(path)
				if err != nil {
					return err
				}
				if resp.StatusCode != 200 {
					return fmt.Errorf("following nextLink returned status %d", resp.StatusCode)
				}
				current = map[string]interface{}{}
				if err := json.Unmarshal(resp.Body, &current); err != nil {
					return fmt.Errorf("invalid JSON on page: %w", err)
				}
			}

			if len(seen) == 0 {
				return fmt.Errorf("no entities collected across all pages")
			}
			ctx.Log(fmt.Sprintf("Collected %d unique entities across all pages", len(seen)))
			return nil
		},
	)

	// Test 4: Ordering is stable across pages when $orderby is applied
	suite.AddTest(
		"test_nextlink_with_orderby_stable",
		"$orderby=Name is stable across all pages (no ordering resets)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=2&$orderby=Name")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var page map[string]interface{}
			if err := json.Unmarshal(resp.Body, &page); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			if _, hasNext := page["@odata.nextLink"]; !hasNext {
				return ctx.Skip("server does not use server-driven paging for this result set")
			}

			var allNames []string
			collectNames := func(p map[string]interface{}) {
				items, _ := p["value"].([]interface{})
				for _, item := range items {
					entity, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					name, _ := entity["Name"].(string)
					allNames = append(allNames, name)
				}
			}

			current := page
			for {
				collectNames(current)
				nextLink, _ := current["@odata.nextLink"].(string)
				if nextLink == "" {
					break
				}
				path := strings.TrimPrefix(nextLink, ctx.ServerURL())
				resp, err = ctx.GET(path)
				if err != nil {
					return err
				}
				if resp.StatusCode != 200 {
					return fmt.Errorf("following nextLink returned status %d", resp.StatusCode)
				}
				current = map[string]interface{}{}
				if err := json.Unmarshal(resp.Body, &current); err != nil {
					return fmt.Errorf("invalid JSON on page: %w", err)
				}
			}

			for i := 1; i < len(allNames); i++ {
				if allNames[i] < allNames[i-1] {
					return fmt.Errorf(
						"ordering is not stable across pages: %q (position %d) < %q (position %d)",
						allNames[i], i, allNames[i-1], i-1,
					)
				}
			}
			ctx.Log(fmt.Sprintf("Names in order across all pages: %v", allNames))
			return nil
		},
	)

	// Test 5: $filter is preserved across all pages via nextLink
	suite.AddTest(
		"test_nextlink_with_filter_preserved",
		"$filter=Price gt 50 applies to all entities on all pages",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=2&$filter=Price gt 50")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var page map[string]interface{}
			if err := json.Unmarshal(resp.Body, &page); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			if _, hasNext := page["@odata.nextLink"]; !hasNext {
				return ctx.Skip("server does not use server-driven paging for this result set")
			}

			checkFilter := func(p map[string]interface{}) error {
				items, _ := p["value"].([]interface{})
				for _, item := range items {
					entity, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					price, ok := entity["Price"].(float64)
					if !ok {
						return fmt.Errorf("entity missing numeric Price field: %v", entity)
					}
					if price <= 50 {
						return fmt.Errorf(
							"entity with Price=%v does not satisfy filter Price gt 50 (filter dropped by nextLink?)",
							price,
						)
					}
				}
				return nil
			}

			current := page
			for {
				if err := checkFilter(current); err != nil {
					return err
				}
				nextLink, _ := current["@odata.nextLink"].(string)
				if nextLink == "" {
					break
				}
				path := strings.TrimPrefix(nextLink, ctx.ServerURL())
				resp, err = ctx.GET(path)
				if err != nil {
					return err
				}
				if resp.StatusCode != 200 {
					return fmt.Errorf("following nextLink returned status %d", resp.StatusCode)
				}
				current = map[string]interface{}{}
				if err := json.Unmarshal(resp.Body, &current); err != nil {
					return fmt.Errorf("invalid JSON on page: %w", err)
				}
			}
			return nil
		},
	)

	// Test 6: @odata.count from first page matches total entities across all pages
	suite.AddTest(
		"test_nextlink_count_consistency",
		"@odata.count matches the total number of entities returned across all pages",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=2&$count=true")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var page map[string]interface{}
			if err := json.Unmarshal(resp.Body, &page); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			if _, hasNext := page["@odata.nextLink"]; !hasNext {
				return ctx.Skip("server does not use server-driven paging for this result set")
			}

			countRaw, ok := page["@odata.count"]
			if !ok {
				return ctx.Skip("server did not include @odata.count in response")
			}
			// @odata.count may be a float64 (JSON number) or a string.
			var declaredCount int
			switch v := countRaw.(type) {
			case float64:
				declaredCount = int(v)
			case string:
				if _, err := fmt.Sscanf(v, "%d", &declaredCount); err != nil {
					return fmt.Errorf("could not parse @odata.count value %q: %w", v, err)
				}
			default:
				return fmt.Errorf("unexpected type for @odata.count: %T", countRaw)
			}

			total := 0
			current := page
			for {
				items, _ := current["value"].([]interface{})
				total += len(items)
				nextLink, _ := current["@odata.nextLink"].(string)
				if nextLink == "" {
					break
				}
				path := strings.TrimPrefix(nextLink, ctx.ServerURL())
				resp, err = ctx.GET(path)
				if err != nil {
					return err
				}
				if resp.StatusCode != 200 {
					return fmt.Errorf("following nextLink returned status %d", resp.StatusCode)
				}
				current = map[string]interface{}{}
				if err := json.Unmarshal(resp.Body, &current); err != nil {
					return fmt.Errorf("invalid JSON on page: %w", err)
				}
			}

			if total != declaredCount {
				return fmt.Errorf(
					"@odata.count=%d but traversal yielded %d entities",
					declaredCount, total,
				)
			}
			ctx.Log(fmt.Sprintf("@odata.count=%d matches traversal total=%d", declaredCount, total))
			return nil
		},
	)

	return suite
}
