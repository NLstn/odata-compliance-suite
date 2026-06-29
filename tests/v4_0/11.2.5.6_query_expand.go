package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// QueryExpand creates the 11.2.5.6 System Query Option $expand test suite
func QueryExpand() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.6 System Query Option $expand",
		"Tests $expand query option for expanding related entities according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionexpand",
	)

	// Test 1: Basic $expand returns related entities inline
	suite.AddTest(
		"test_expand_basic",
		"$expand includes related entities inline",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions")
			resp, err := ctx.GET("/Products?$expand=" + expand + "&$top=1")
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
				return fmt.Errorf("response contains no items: %w", err)
			}

			item := items[0]

			// Verify Descriptions field is present
			descriptions, ok := item["Descriptions"]
			if !ok {
				return fmt.Errorf("descriptions field is missing")
			}

			// Verify Descriptions is an array (expanded data)
			if _, ok := descriptions.([]interface{}); !ok {
				return fmt.Errorf("descriptions field is not an array (not properly expanded)")
			}

			return nil
		},
	)

	// Test 2: $expand with $select on expanded entity
	suite.AddTest(
		"test_expand_with_select",
		"$expand with nested $select",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($select=Description)")
			filter := url.QueryEscape("Name eq 'Laptop'")
			resp, err := ctx.GET("/Products?$filter=" + filter + "&$expand=" + expand + "&$top=1")
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
				return fmt.Errorf("response contains no items: %w", err)
			}

			item := items[0]

			// Verify Descriptions field is present and expanded
			descriptions, ok := item["Descriptions"]
			if !ok {
				return fmt.Errorf("descriptions field is missing")
			}

			// Verify it's an array
			descArray, ok := descriptions.([]interface{})
			if !ok {
				return fmt.Errorf("descriptions field is not an array")
			}

			if len(descArray) == 0 {
				return fmt.Errorf("expected at least one expanded description for Laptop fixture")
			}

			desc, ok := descArray[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("first description is not an object")
			}
			if _, ok := desc["Description"]; !ok {
				return fmt.Errorf("expanded Descriptions missing Description field")
			}
			// Key properties may be present even when not listed in nested $select.
			// The assertion here focuses on ensuring the selected non-key field is included.

			return nil
		},
	)

	// Test 3: $expand on single entity
	suite.AddTest(
		"test_expand_single_entity",
		"$expand on single entity request",
		func(ctx *framework.TestContext) error {
			// First get a product ID
			allResp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(allResp, 200); err != nil {
				return fmt.Errorf("failed to get products: %w", err)
			}

			items, err := ctx.ParseEntityCollection(allResp)
			if err != nil {
				return fmt.Errorf("failed to parse products JSON: %w", err)
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("no products available")
			}

			firstItem := items[0]

			productID := firstItem["ID"]

			// Now test $expand on single entity
			expand := url.QueryEscape("Descriptions")
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)?$expand=%s", productID, expand))
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

			// Verify Descriptions field is present and expanded
			descriptions, ok := result["Descriptions"]
			if !ok {
				return fmt.Errorf("descriptions field is missing")
			}

			// Verify it's expanded (should be an array)
			if _, ok := descriptions.([]interface{}); !ok {
				return fmt.Errorf("descriptions not expanded as array")
			}

			return nil
		},
	)

	// Test 4: $expand with parent $select should still include single-valued navigation entity
	suite.AddTest(
		"test_expand_with_parent_select_belongs_to",
		"$expand single-valued navigation works when parent $select omits foreign key",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("CategoryID ne null")
			expand := url.QueryEscape("Category($select=Name)")
			resp, err := ctx.GET("/Products?$filter=" + filter + "&$top=1&$select=Name&$expand=" + expand)
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
				return fmt.Errorf("response missing 'value' array or contains no items: %w", err)
			}

			item := items[0]

			categoryValue, exists := item["Category"]
			if !exists {
				return fmt.Errorf("category field missing from expanded response")
			}
			if categoryValue == nil {
				return fmt.Errorf("category field is null; expected expanded entity")
			}

			category, ok := categoryValue.(map[string]interface{})
			if !ok {
				return fmt.Errorf("category field is not an object")
			}

			if err := ctx.AssertEntityHasFields(category, "Name"); err != nil {
				return fmt.Errorf("expanded category missing selected Name property: %w", err)
			}

			return nil
		},
	)

	// Test 5: $expand with $count inside expand options
	suite.AddTest(
		"test_expand_collection_with_count",
		"$expand with nested $count returns odata.count annotation",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($count=true)")
			resp, err := ctx.GET("/Products?$top=1&$expand=" + expand)
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
				return fmt.Errorf("response contains no items: %w", err)
			}

			item := items[0]

			// OData spec requires Descriptions@odata.count annotation when $count=true
			countVal, ok := item["Descriptions@odata.count"]
			if !ok {
				return fmt.Errorf("expected Descriptions@odata.count annotation in response (required by §11.2.5.6 when $count=true)")
			}

			// The count must be a non-negative number
			switch v := countVal.(type) {
			case float64:
				if v < 0 {
					return fmt.Errorf("Descriptions@odata.count is negative (%v)", v)
				}
			default:
				return fmt.Errorf("Descriptions@odata.count has unexpected type %T (expected number)", countVal)
			}

			return nil
		},
	)

	// Test 6: $expand with $top inside expand options
	suite.AddTest(
		"test_expand_collection_with_top",
		"$expand with nested $top limits expanded collection size",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($top=1)")
			resp, err := ctx.GET("/Products?$top=1&$expand=" + expand)
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
				return fmt.Errorf("response contains no items: %w", err)
			}

			item := items[0]

			descriptions, ok := item["Descriptions"]
			if !ok {
				return fmt.Errorf("Descriptions field is missing from expanded response")
			}

			descArray, ok := descriptions.([]interface{})
			if !ok {
				return fmt.Errorf("Descriptions field is not an array")
			}

			// $top=1 inside $expand must limit the collection to at most 1 item
			if len(descArray) > 1 {
				return fmt.Errorf("expected at most 1 Descriptions item due to $top=1, got %d", len(descArray))
			}

			return nil
		},
	)

	// Test 7: $expand with $filter inside expand options
	suite.AddTest(
		"test_expand_collection_with_filter",
		"$expand with nested $filter restricts expanded collection members",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($filter=LanguageKey eq 'EN')")
			resp, err := ctx.GET("/Products?$expand=" + expand)
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

			// Check every returned product's Descriptions — all must have LanguageKey='EN'
			checkedAny := false
			for _, item := range items {
				descriptions, ok := item["Descriptions"]
				if !ok {
					continue
				}
				descArray, ok := descriptions.([]interface{})
				if !ok || len(descArray) == 0 {
					continue
				}
				for i, d := range descArray {
					desc, ok := d.(map[string]interface{})
					if !ok {
						return fmt.Errorf("Descriptions[%d] is not an object", i)
					}
					lk, ok := desc["LanguageKey"]
					if !ok {
						return fmt.Errorf("Descriptions[%d] missing LanguageKey field", i)
					}
					if lk != "EN" {
						return fmt.Errorf("Descriptions[%d] has LanguageKey=%q, expected 'EN' (filter should exclude other languages)", i, lk)
					}
					checkedAny = true
				}
			}

			if !checkedAny {
				return ctx.Skip("no products with Descriptions found — cannot verify filter behaviour")
			}

			return nil
		},
	)

	// Test 8: $expand=Nav/$ref — server must return entity references per OData spec §5.1.3
	suite.AddTest(
		"test_expand_nav_ref",
		"$expand=Category/$ref returns entity references (OData spec §5.1.3)",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Category/$ref")
			resp, err := ctx.GET("/Products?$top=1&$expand=" + expand)
			if err != nil {
				return err
			}

			// Any non-200 status is a spec violation: the server must support $expand=Nav/$ref
			if resp.StatusCode != 200 {
				return fmt.Errorf(
					"server rejected $expand=Category/$ref with status %d — "+
						"OData 4.0 spec §5.1.3 (Part 2) requires servers to support this syntax "+
						"to return entity references instead of full entities (body: %s)",
					resp.StatusCode, string(resp.Body),
				)
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("response contains no items: %w", err)
			}

			// Each item that has an expanded Category must contain @odata.id (entity reference)
			for i, item := range items {
				catVal, ok := item["Category"]
				if !ok || catVal == nil {
					continue // null/missing Category is fine
				}
				cat, ok := catVal.(map[string]interface{})
				if !ok {
					return fmt.Errorf("items[%d].Category is not an object", i)
				}
				if _, hasRef := cat["@odata.id"]; !hasRef {
					return fmt.Errorf(
						"items[%d].Category is missing @odata.id — "+
							"$expand=Nav/$ref must return entity references containing @odata.id",
						i,
					)
				}
			}

			return nil
		},
	)

	// Test 9: $expand with $orderby inside expand options
	suite.AddTest(
		"test_expand_with_orderby",
		"$expand with nested $orderby sorts the expanded collection",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($orderby=LanguageKey)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
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

			checkedAny := false
			for _, item := range items {
				descriptions, ok := item["Descriptions"]
				if !ok {
					continue
				}
				descArray, ok := descriptions.([]interface{})
				if !ok || len(descArray) < 2 {
					// Need at least 2 entries to verify ordering
					continue
				}

				// Collect LanguageKey values
				keys := make([]string, 0, len(descArray))
				for i, d := range descArray {
					desc, ok := d.(map[string]interface{})
					if !ok {
						return fmt.Errorf("Descriptions[%d] is not an object", i)
					}
					lk, ok := desc["LanguageKey"]
					if !ok {
						return fmt.Errorf("Descriptions[%d] missing LanguageKey field", i)
					}
					lkStr, ok := lk.(string)
					if !ok {
						return fmt.Errorf("Descriptions[%d].LanguageKey is not a string", i)
					}
					keys = append(keys, lkStr)
				}

				// Verify ascending order
				sorted := make([]string, len(keys))
				copy(sorted, keys)
				sort.Strings(sorted)
				for i := range keys {
					if keys[i] != sorted[i] {
						return fmt.Errorf(
							"Descriptions are not sorted by LanguageKey ascending: got %v, expected %v",
							keys, sorted,
						)
					}
				}
				checkedAny = true
			}

			if !checkedAny {
				return ctx.Skip("no products with multiple Descriptions found — cannot verify $orderby behaviour")
			}

			return nil
		},
	)

	return suite
}
