package v4_0

import (
	"fmt"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// QuerySelectWithNavigationFilter creates test suite for combining $select with navigation property filters
// This tests that SELECT clauses properly qualify column names when JOINs are present
func QuerySelectWithNavigationFilter() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.11 System Query Option $select with Navigation Property Filters",
		"Tests $select combined with $filter on navigation properties to ensure proper SQL generation with qualified column names",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptionselect",
	)
	RegisterQuerySelectWithNavigationFilterTests(suite)
	return suite
}

// RegisterQuerySelectWithNavigationFilterTests registers tests for $select with navigation filters
func RegisterQuerySelectWithNavigationFilterTests(suite *framework.TestSuite) {
	suite.AddTest(
		"Select single property with navigation filter",
		"Combine $select with $filter on navigation property to ensure ambiguous column errors don't occur",
		testSelectWithNavigationFilter,
	)

	suite.AddTest(
		"Select multiple properties with navigation filter",
		"Test $select with multiple properties and navigation filter",
		testSelectMultipleWithNavigationFilter,
	)

	suite.AddTest(
		"Select with navigation filter on different properties",
		"Test various navigation property comparisons",
		testSelectWithVariousNavigationFilters,
	)

	suite.AddTest(
		"Select with navigation filter and expand",
		"Combine $select, $filter, and $expand on navigation properties",
		testSelectWithNavigationFilterAndExpand,
	)
}

// fetchAllProductsWithCategory returns every Product with its Category expanded,
// used to build a client-side oracle for $filter expressions over Category/Name.
func fetchAllProductsWithCategory(ctx *framework.TestContext) ([]map[string]interface{}, error) {
	resp, err := ctx.GET("/Products?$expand=Category&$top=1000")
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, err
	}
	return ctx.ParseEntityCollection(resp)
}

// productCategoryName reads the Name of an expanded Category navigation property.
// ok is false when Category is absent, null, or malformed.
func productCategoryName(p map[string]interface{}) (string, bool) {
	cat, ok := p["Category"].(map[string]interface{})
	if !ok {
		return "", false
	}
	name, ok := cat["Name"].(string)
	return name, ok
}

// assertNavigationFilter runs /Products?$select=<select>&$filter=<expr> and asserts
// the returned entity-ID set exactly matches want(), evaluated against a full fetch
// with Category expanded — verifying both that the navigation-property filter is
// applied correctly (soundness + completeness) and that $select still works when a
// JOIN is introduced for the filter.
func assertNavigationFilter(ctx *framework.TestContext, selectClause, expr string, want func(p map[string]interface{}) bool) error {
	all, err := fetchAllProductsWithCategory(ctx)
	if err != nil {
		return err
	}
	expected := map[string]bool{}
	for _, p := range all {
		if want(p) {
			expected[productID(p)] = true
		}
	}

	path := "/Products?$select=" + url.QueryEscape(selectClause) + "&$filter=" + url.QueryEscape(expr)
	resp, err := ctx.GET(path)
	if err != nil {
		return err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return fmt.Errorf("filter %q with $select=%q: %w (body: %s)", expr, selectClause, err, string(resp.Body))
	}

	items, err := ctx.ParseEntityCollection(resp)
	if err != nil {
		return err
	}
	got := map[string]bool{}
	for _, p := range items {
		got[productID(p)] = true
	}

	for id := range expected {
		if !got[id] {
			return fmt.Errorf("filter %q: product %s satisfies Category/Name predicate but was not returned", expr, id)
		}
	}
	for id := range got {
		if !expected[id] {
			return fmt.Errorf("filter %q: product %s was returned but does not satisfy the Category/Name predicate", expr, id)
		}
	}
	return nil
}

func testSelectWithNavigationFilter(ctx *framework.TestContext) error {
	// /Products?$select=Name&$filter=Category/Name eq 'Electronics'
	// This also exercises that when a JOIN is added for the filter, the SELECT
	// clause uses qualified column names to avoid ambiguous column references.
	if err := assertNavigationFilter(ctx, "Name", "Category/Name eq 'Electronics'", func(p map[string]interface{}) bool {
		name, ok := productCategoryName(p)
		return ok && name == "Electronics"
	}); err != nil {
		return err
	}

	// Confirm $select still behaves correctly under the JOIN: only the
	// selected property (plus the always-returned key) comes back.
	filter := url.QueryEscape("Category/Name eq 'Electronics'")
	select_ := url.QueryEscape("Name")
	resp, err := ctx.GET("/Products?$select=" + select_ + "&$filter=" + filter)
	if err != nil {
		return err
	}
	items, err := ctx.ParseEntityCollection(resp)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	if err := ctx.AssertEntityHasFields(items[0], "Name"); err != nil {
		return err
	}
	return ctx.AssertEntityOnlyAllowedFields(items[0], "ID", "Name")
}

func testSelectMultipleWithNavigationFilter(ctx *framework.TestContext) error {
	// /Products?$select=Name,Price&$filter=Category/Name eq 'Electronics'
	if err := assertNavigationFilter(ctx, "Name,Price", "Category/Name eq 'Electronics'", func(p map[string]interface{}) bool {
		name, ok := productCategoryName(p)
		return ok && name == "Electronics"
	}); err != nil {
		return err
	}

	filter := url.QueryEscape("Category/Name eq 'Electronics'")
	select_ := url.QueryEscape("Name,Price")
	resp, err := ctx.GET("/Products?$select=" + select_ + "&$filter=" + filter)
	if err != nil {
		return err
	}
	items, err := ctx.ParseEntityCollection(resp)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	if err := ctx.AssertEntityHasFields(items[0], "Name", "Price"); err != nil {
		return err
	}
	return ctx.AssertEntityOnlyAllowedFields(items[0], "ID", "Name", "Price")
}

func testSelectWithVariousNavigationFilters(ctx *framework.TestContext) error {
	tests := []struct {
		filter string
		desc   string
		want   func(p map[string]interface{}) bool
	}{
		{
			filter: "Category/Name ne 'Books'",
			desc:   "not equals",
			want: func(p map[string]interface{}) bool {
				name, ok := productCategoryName(p)
				return ok && name != "Books"
			},
		},
		{
			filter: "Category/Name eq 'Electronics' or Category/Name eq 'Computers'",
			desc:   "logical or",
			want: func(p map[string]interface{}) bool {
				name, ok := productCategoryName(p)
				return ok && (name == "Electronics" || name == "Computers")
			},
		},
		{
			filter: "Category/Name eq 'Electronics' and Price gt 100",
			desc:   "combined with regular property",
			want: func(p map[string]interface{}) bool {
				name, ok := productCategoryName(p)
				if !ok || name != "Electronics" {
					return false
				}
				price, ok := productFloat(p, "Price")
				return ok && price > 100
			},
		},
	}

	for _, tt := range tests {
		if err := assertNavigationFilter(ctx, "Name", tt.filter, tt.want); err != nil {
			return fmt.Errorf("test %q: %w", tt.desc, err)
		}
	}

	return nil
}

func testSelectWithNavigationFilterAndExpand(ctx *framework.TestContext) error {
	// /Products?$select=Name,Category&$filter=Category/Name eq 'Electronics'&$expand=Category
	// Tests that $select, $filter, and $expand all work together with navigation
	// properties, and that Category is genuinely expanded (not just that the
	// request avoids an ambiguous-column error).
	all, err := fetchAllProductsWithCategory(ctx)
	if err != nil {
		return err
	}
	expected := map[string]bool{}
	for _, p := range all {
		if name, ok := productCategoryName(p); ok && name == "Electronics" {
			expected[productID(p)] = true
		}
	}

	filter := url.QueryEscape("Category/Name eq 'Electronics'")
	select_ := url.QueryEscape("Name,Category")
	expand := url.QueryEscape("Category")

	resp, err := ctx.GET("/Products?$select=" + select_ + "&$filter=" + filter + "&$expand=" + expand)
	if err != nil {
		return err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return fmt.Errorf("%w (body: %s)", err, string(resp.Body))
	}

	items, err := ctx.ParseEntityCollection(resp)
	if err != nil {
		return err
	}

	got := map[string]bool{}
	for _, p := range items {
		got[productID(p)] = true
	}
	for id := range expected {
		if !got[id] {
			return fmt.Errorf("product %s satisfies Category/Name eq 'Electronics' but was not returned", id)
		}
	}
	for id := range got {
		if !expected[id] {
			return fmt.Errorf("product %s was returned but does not satisfy Category/Name eq 'Electronics'", id)
		}
	}

	if len(items) == 0 {
		return nil
	}
	name, ok := productCategoryName(items[0])
	if !ok {
		return fmt.Errorf("Category was explicitly $select-ed and $expand-ed but is missing or not an object in the response")
	}
	if name != "Electronics" {
		return fmt.Errorf("expanded Category.Name = %q, want %q", name, "Electronics")
	}
	return nil
}
