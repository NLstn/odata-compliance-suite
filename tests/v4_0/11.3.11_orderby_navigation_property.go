package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// OrderByNavigationProperty creates the test suite for $orderby on single-entity navigation property paths.
// Per OData v4.01 spec section 5.1.1.15, properties of entities related with cardinality 0..1 or 1
// can be used as operands in $orderby expressions using a slash-separated path (e.g., Author/Name).
func OrderByNavigationProperty() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.11 $orderby on Single-Entity Navigation Property Paths",
		"Validates that $orderby accepts single-entity navigation property paths (e.g., $orderby=Category/Name)",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_PathExpressions",
	)
	RegisterOrderByNavigationPropertyTests(suite)
	return suite
}

// RegisterOrderByNavigationPropertyTests registers all $orderby navigation property path tests.
func RegisterOrderByNavigationPropertyTests(suite *framework.TestSuite) {
	suite.AddTest(
		"orderby_navigation_property_asc",
		"$orderby on navigation property path ascending returns results sorted by Category/Name",
		testOrderByNavigationPropertyAsc,
	)

	suite.AddTest(
		"orderby_navigation_property_desc",
		"$orderby on navigation property path descending returns 200 and sorted results",
		testOrderByNavigationPropertyDesc,
	)

	suite.AddTest(
		"orderby_navigation_property_with_expand",
		"$orderby on navigation property path combined with $expand on the same property",
		testOrderByNavigationPropertyWithExpand,
	)

	suite.AddTest(
		"orderby_navigation_property_with_filter",
		"$orderby on navigation property path combined with $filter",
		testOrderByNavigationPropertyWithFilter,
	)

	suite.AddTest(
		"orderby_navigation_property_multiple_clauses",
		"$orderby with navigation property path and regular property as multiple clauses",
		testOrderByNavigationPropertyMultipleClauses,
	)

	suite.AddTest(
		"orderby_collection_navigation_rejected",
		"$orderby directly on a collection-valued navigation property is rejected with 400",
		testOrderByCollectionNavigationRejected,
	)
}

// navSortEntry captures the two fields these tests order by: the expanded
// Category.Name (possibly absent) and the product's own Name.
type navSortEntry struct {
	categoryName string
	hasCategory  bool
	name         string
}

func extractCategorySortEntries(values []interface{}) ([]navSortEntry, error) {
	entries := make([]navSortEntry, 0, len(values))
	for i, raw := range values {
		entity, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("item %d is not an object", i)
		}
		var e navSortEntry
		if name, ok := entity["Name"].(string); ok {
			e.name = name
		}
		if catRaw, has := entity["Category"]; has && catRaw != nil {
			cat, ok := catRaw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("item %d: Category is not an object", i)
			}
			if name, ok := cat["Name"].(string); ok {
				e.categoryName = name
				e.hasCategory = true
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// assertSortedByCategoryName checks entries are ordered by Category.Name.
// Products without an expanded category are excluded from the comparison
// (their placement relative to categorized products is not asserted here);
// only the relative order among categorized products is verified.
func assertSortedByCategoryName(entries []navSortEntry, ascending bool) error {
	var prev string
	havePrev := false
	for i, e := range entries {
		if !e.hasCategory {
			continue
		}
		if havePrev {
			if ascending && e.categoryName < prev {
				return fmt.Errorf("ordering violation at index %d: %q after %q (not ascending)", i, e.categoryName, prev)
			}
			if !ascending && e.categoryName > prev {
				return fmt.Errorf("ordering violation at index %d: %q after %q (not descending)", i, e.categoryName, prev)
			}
		}
		prev = e.categoryName
		havePrev = true
	}
	return nil
}

// assertSortedByCategoryNameThenName checks entries are ordered by
// Category.Name ascending, then by Name ascending within a tied category.
func assertSortedByCategoryNameThenName(entries []navSortEntry) error {
	var prev navSortEntry
	havePrev := false
	for i, e := range entries {
		if !e.hasCategory {
			continue
		}
		if havePrev {
			if e.categoryName < prev.categoryName {
				return fmt.Errorf("ordering violation at index %d: category %q after %q (not ascending)", i, e.categoryName, prev.categoryName)
			}
			if e.categoryName == prev.categoryName && e.name < prev.name {
				return fmt.Errorf("ordering violation at index %d: within category %q, name %q after %q (not ascending)", i, e.categoryName, e.name, prev.name)
			}
		}
		prev = e
		havePrev = true
	}
	return nil
}

func testOrderByNavigationPropertyAsc(ctx *framework.TestContext) error {
	resp, err := ctx.GET("/Products?$expand=" + url.QueryEscape("Category($select=Name)") + "&$orderby=" + url.QueryEscape("Category/Name asc"))
	if err != nil {
		return err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return fmt.Errorf("$orderby=Category/Name asc should be accepted: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	values, ok := result["value"].([]interface{})
	if !ok {
		return fmt.Errorf("response 'value' is not an array")
	}

	entries, err := extractCategorySortEntries(values)
	if err != nil {
		return err
	}
	return assertSortedByCategoryName(entries, true)
}

func testOrderByNavigationPropertyDesc(ctx *framework.TestContext) error {
	resp, err := ctx.GET("/Products?$expand=" + url.QueryEscape("Category($select=Name)") + "&$orderby=" + url.QueryEscape("Category/Name desc"))
	if err != nil {
		return err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return fmt.Errorf("$orderby=Category/Name desc should be accepted: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	values, ok := result["value"].([]interface{})
	if !ok {
		return fmt.Errorf("response 'value' is not an array")
	}

	entries, err := extractCategorySortEntries(values)
	if err != nil {
		return err
	}
	return assertSortedByCategoryName(entries, false)
}

func testOrderByNavigationPropertyWithExpand(ctx *framework.TestContext) error {
	resp, err := ctx.GET("/Products?$expand=Category&$orderby=" + url.QueryEscape("Category/Name"))
	if err != nil {
		return err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return fmt.Errorf("$expand=Category&$orderby=Category/Name should be accepted: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	values, ok := result["value"].([]interface{})
	if !ok {
		return fmt.Errorf("response 'value' is not an array")
	}

	entries, err := extractCategorySortEntries(values)
	if err != nil {
		return err
	}
	return assertSortedByCategoryName(entries, true)
}

func testOrderByNavigationPropertyWithFilter(ctx *framework.TestContext) error {
	// Combine $filter on a navigation property path with $orderby on the same path.
	all, err := fetchAllProductsWithCategory(ctx)
	if err != nil {
		return err
	}
	expected := map[string]bool{}
	for _, p := range all {
		if name, ok := productCategoryName(p); ok && name != "unknown" {
			expected[productID(p)] = true
		}
	}

	filterExpr := url.QueryEscape("Category/Name ne 'unknown'")
	orderExpr := url.QueryEscape("Category/Name asc")
	expandExpr := url.QueryEscape("Category($select=Name)")
	resp, err := ctx.GET("/Products?$expand=" + expandExpr + "&$filter=" + filterExpr + "&$orderby=" + orderExpr)
	if err != nil {
		return err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return fmt.Errorf("combined $filter and $orderby on Category/Name should be accepted: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	values, ok := result["value"].([]interface{})
	if !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	entries, err := extractCategorySortEntries(values)
	if err != nil {
		return err
	}
	if err := assertSortedByCategoryName(entries, true); err != nil {
		return err
	}

	got := map[string]bool{}
	for i, raw := range values {
		entity, ok := raw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("item %d is not an object", i)
		}
		got[productID(entity)] = true
	}
	for id := range expected {
		if !got[id] {
			return fmt.Errorf("product %s satisfies Category/Name ne 'unknown' but was not returned", id)
		}
	}
	for id := range got {
		if !expected[id] {
			return fmt.Errorf("product %s was returned but does not satisfy Category/Name ne 'unknown'", id)
		}
	}
	return nil
}

func testOrderByNavigationPropertyMultipleClauses(ctx *framework.TestContext) error {
	// Multiple $orderby clauses: navigation property path + regular property.
	expandExpr := url.QueryEscape("Category($select=Name)")
	orderExpr := url.QueryEscape("Category/Name asc,Name asc")
	resp, err := ctx.GET("/Products?$expand=" + expandExpr + "&$orderby=" + orderExpr)
	if err != nil {
		return err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return fmt.Errorf("$orderby=Category/Name asc,Name asc should be accepted: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	values, ok := result["value"].([]interface{})
	if !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	entries, err := extractCategorySortEntries(values)
	if err != nil {
		return err
	}
	return assertSortedByCategoryNameThenName(entries)
}

func testOrderByCollectionNavigationRejected(ctx *framework.TestContext) error {
	// Collection-valued navigation properties (e.g., Descriptions on Products) must NOT be
	// directly usable in $orderby without a lambda/ aggregation — this is invalid per spec.
	resp, err := ctx.GET("/Products?$orderby=Descriptions/LanguageKey")
	if err != nil {
		return err
	}

	// Must reject with 400 Bad Request.
	if resp.StatusCode != 400 {
		return fmt.Errorf("expected 400 for $orderby on collection navigation property, got %d", resp.StatusCode)
	}

	return nil
}
