package v4_0

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// NavigationPropertyOperations creates the 11.4.6.1 Navigation Property Operations test suite
func NavigationPropertyOperations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.6.1 Navigation Property Operations",
		"Tests operations on navigation properties including accessing, filtering, and modifying.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_NavigationProperties",
	)

	invalidProductPath := nonExistingEntityPath("Products")

	// Test 1: Access navigation property collection
	suite.AddTest(
		"test_nav_property_collection",
		"Access navigation property collection",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return framework.NewError("Navigation property access returning 404, likely routing issue")
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 2: Navigation property returns collection structure
	suite.AddTest(
		"test_nav_property_collection_structure",
		"Navigation property returns collection structure",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return framework.NewError("Navigation property access returning 404, likely routing issue")
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "value")
		},
	)

	// Test 3: Navigation property with $filter — all returned descriptions must
	// have LanguageKey eq 'EN'.
	suite.AddTest(
		"test_nav_property_filter",
		"Navigation property $filter returns only items matching the predicate",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			filter := url.QueryEscape("LanguageKey eq 'EN'")
			resp, err := ctx.GET(prodPath + "/Descriptions?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode == 404 {
				return framework.NewError("Navigation property access returning 404, likely routing issue")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			for i, d := range items {
				lk, _ := d["LanguageKey"].(string)
				if lk != "EN" {
					return fmt.Errorf("Descriptions[%d].LanguageKey=%q but filter was LanguageKey eq 'EN'", i, lk)
				}
			}
			return nil
		},
	)

	// Test 4: Navigation property with $select — returned items must contain only
	// the requested fields (LanguageKey and Description).
	suite.AddTest(
		"test_nav_property_select",
		"Navigation property $select restricts fields to requested set",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions?$select=LanguageKey,Description")
			if err != nil {
				return err
			}
			if resp.StatusCode == 404 {
				return framework.NewError("Navigation property access returning 404, likely routing issue")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			// ProductID is a key property of Descriptions and is always returned per spec.
			allowed := []string{"ProductID", "LanguageKey", "Description", "@odata.etag", "@odata.id", "@odata.type"}
			for i, item := range items {
				if err := ctx.AssertEntityOnlyAllowedFields(item, allowed...); err != nil {
					return fmt.Errorf("Descriptions[%d]: %w", i, err)
				}
			}
			return nil
		},
	)

	// Test 5: Navigation property with $orderby — items must be sorted ascending
	// by LanguageKey.
	suite.AddTest(
		"test_nav_property_orderby",
		"Navigation property $orderby sorts results by the specified field",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions?$orderby=LanguageKey")
			if err != nil {
				return err
			}
			if resp.StatusCode == 404 {
				return framework.NewError("Navigation property access returning 404, likely routing issue")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			keys := make([]string, 0, len(items))
			for _, d := range items {
				lk, _ := d["LanguageKey"].(string)
				keys = append(keys, lk)
			}
			sorted := append([]string(nil), keys...)
			sort.Strings(sorted)
			for i := range keys {
				if keys[i] != sorted[i] {
					return fmt.Errorf("Descriptions not sorted by LanguageKey asc: position %d got %q, expected %q; full order: %v",
						i, keys[i], sorted[i], keys)
				}
			}
			return nil
		},
	)

	// Test 6: Navigation property with $top — at most 2 items must be returned.
	suite.AddTest(
		"test_nav_property_top",
		"Navigation property $top=2 returns at most 2 items",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions?$top=2")
			if err != nil {
				return err
			}
			if resp.StatusCode == 404 {
				return framework.NewError("Navigation property access returning 404, likely routing issue")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if len(items) > 2 {
				return fmt.Errorf("$top=2 must return at most 2 items; got %d", len(items))
			}
			return nil
		},
	)

	// Test 7: Navigation property with $count
	suite.AddTest(
		"test_nav_property_count",
		"Navigation property with $count",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions?$count=true")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return framework.NewError("Navigation property access returning 404, likely routing issue")
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "@odata.count")
		},
	)

	// Test 8: Navigation property on non-existent entity returns 404
	suite.AddTest(
		"test_nav_property_not_found",
		"Navigation property on invalid entity returns 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath + "/Descriptions")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 404)
		},
	)

	// Test 9: Navigation property with combined query options — verify filter,
	// select, and orderby all apply correctly at once.
	suite.AddTest(
		"test_nav_property_combined_options",
		"Navigation property with $filter, $select, $orderby applies all constraints simultaneously",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			filter := url.QueryEscape("LanguageKey eq 'EN'")
			resp, err := ctx.GET(prodPath + "/Descriptions?$filter=" + filter + "&$select=LanguageKey,Description&$orderby=LanguageKey")
			if err != nil {
				return err
			}
			if resp.StatusCode == 404 {
				return framework.NewError("Navigation property access returning 404, likely routing issue")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			// All returned items must satisfy the filter.
			for i, d := range items {
				lk, _ := d["LanguageKey"].(string)
				if lk != "EN" {
					return fmt.Errorf("Descriptions[%d].LanguageKey=%q violates $filter=LanguageKey eq 'EN'", i, lk)
				}
			}
			// $select=LanguageKey,Description — ProductID is always returned as key.
			allowed := []string{"ProductID", "LanguageKey", "Description", "@odata.etag", "@odata.id", "@odata.type"}
			for i, d := range items {
				if err := ctx.AssertEntityOnlyAllowedFields(d, allowed...); err != nil {
					return fmt.Errorf("Descriptions[%d]: %w", i, err)
				}
			}
			return nil
		},
	)

	// Test 10: Navigation property has proper @odata.context
	suite.AddTest(
		"test_nav_property_context",
		"Navigation property has @odata.context",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return framework.NewError("Navigation property access returning 404, likely routing issue")
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "@odata.context")
		},
	)

	return suite
}
