package v4_0

import (
	"fmt"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// FilterOnSingleEntityNavigationProperties creates test suite for filtering on single-entity navigation properties
// Per OData v4.01 spec 5.1.1.15, properties of entities related with cardinality 0..1 or 1 can be accessed directly
func FilterOnSingleEntityNavigationProperties() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.10 Filter on Single-Entity Navigation Properties",
		"Tests filtering on single-entity (cardinality 0..1 or 1) navigation properties",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_PathExpressions",
	)
	RegisterFilterOnSingleEntityNavigationPropertiesTests(suite)
	return suite
}

// RegisterFilterOnSingleEntityNavigationPropertiesTests registers tests for filtering on single-entity navigation properties
func RegisterFilterOnSingleEntityNavigationPropertiesTests(suite *framework.TestSuite) {
	suite.AddTest(
		"Filter by single-entity navigation property",
		"Filter entities by properties of related entities with cardinality 0..1 or 1",
		testFilterBySingleEntityNavigationProperty,
	)

	suite.AddTest(
		"Filter with navigation property path in comparison",
		"Use navigation property paths in filter comparisons",
		testNavigationPropertyPathComparison,
	)

	suite.AddTest(
		"Collection navigation properties still require lambda operators",
		"Verify that collection navigation properties cannot be used without any/all",
		testCollectionNavigationRequiresLambda,
	)

	suite.AddTest(
		"Combine single-entity navigation filter with expand",
		"Filter and expand the same navigation property",
		testCombineFilterAndExpandNavigation,
	)
}

func testFilterBySingleEntityNavigationProperty(ctx *framework.TestContext) error {
	// Filter by a property of a single-entity navigation property and verify the
	// filter was actually applied: every returned product must belong to the
	// Electronics category (Part 2 §5.1.1.15). Expanding Category lets us check.
	resp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("Category/Name eq 'Electronics'") + "&$expand=Category")
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(resp.Body))
	}

	items, err := ctx.ParseEntityCollection(resp)
	if err != nil {
		return err
	}
	if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
		return fmt.Errorf("seed data has Electronics products, expected at least one match: %w", err)
	}

	return ctx.AssertAllEntitiesSatisfy(items, "Category/Name eq 'Electronics'", func(entity map[string]interface{}) (bool, string) {
		cat, ok := entity["Category"].(map[string]interface{})
		if !ok {
			return false, "expanded Category is missing; cannot confirm the navigation-property filter was applied"
		}
		if cat["Name"] != "Electronics" {
			return false, fmt.Sprintf("expected Category/Name 'Electronics', got %v", cat["Name"])
		}
		return true, ""
	})
}

func testNavigationPropertyPathComparison(ctx *framework.TestContext) error {
	// A navigation-property path that matches nothing must return an empty set,
	// not the whole collection. This catches a server that silently ignores a
	// navigation-property filter (which would pass a status-only assertion).
	resp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("Category/Name eq 'NoSuchCategoryXYZ'"))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	items, err := ctx.ParseEntityCollection(resp)
	if err != nil {
		return err
	}
	if len(items) != 0 {
		return fmt.Errorf("filter on a non-existent category should return 0 products; got %d (server may be ignoring the navigation-property filter)", len(items))
	}

	return nil
}

func testCollectionNavigationRequiresLambda(ctx *framework.TestContext) error {
	// Verify that collection navigation properties still require any/all operators
	// This ensures we maintain spec compliance for collection-valued navigation properties

	// Try to filter by a collection navigation property without lambda operator
	// This should return a 400 Bad Request error
	resp, err := ctx.GET("/Products?$filter=Descriptions/LanguageKey eq 'EN'")
	if err != nil {
		return framework.NewError("Products entity set not available or doesn't have Descriptions collection")
	}

	// This should fail because Descriptions is a collection
	if resp.StatusCode == 200 {
		return fmt.Errorf("expected error for collection navigation property without lambda, got status 200")
	}

	// We expect a 400 Bad Request
	if resp.StatusCode != 400 {
		return fmt.Errorf("expected status 400 for collection navigation without lambda, got %d", resp.StatusCode)
	}

	return nil
}

func testCombineFilterAndExpandNavigation(ctx *framework.TestContext) error {
	// Test that we can both filter on and expand the same navigation property
	// Per spec, this should work seamlessly

	escapedFilter := url.QueryEscape("Category/Name eq 'Electronics'")
	resp, err := ctx.GET("/Products?$filter=" + escapedFilter + "&$expand=Category")
	if err != nil {
		return framework.NewError("Products entity set not available")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(resp.Body))
	}

	items, err := ctx.ParseEntityCollection(resp)
	if err != nil {
		return err
	}
	if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
		return fmt.Errorf("expected at least one Electronics product when filtering and expanding Category: %w", err)
	}

	// Every returned product must both be in the Electronics category (filter
	// applied) and carry the expanded Category object (expand applied).
	return ctx.AssertAllEntitiesSatisfy(items, "Category/Name eq 'Electronics' with $expand=Category", func(entity map[string]interface{}) (bool, string) {
		cat, ok := entity["Category"].(map[string]interface{})
		if !ok {
			return false, "expected Category navigation property to be expanded in response"
		}
		if cat["Name"] != "Electronics" {
			return false, fmt.Sprintf("expected expanded Category/Name 'Electronics', got %v", cat["Name"])
		}
		return true, ""
	})
}
