package v4_0

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// FilterExpandedProperties creates a test suite for filtering on expanded properties
func FilterExpandedProperties() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.8 Filter on Expanded Properties",
		"Tests filtering entities based on properties of expanded navigation entities",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_ExpandSystemQueryOption",
	)
	RegisterFilterExpandedPropertiesTests(suite)
	return suite
}

// RegisterFilterExpandedPropertiesTests registers tests for filtering on expanded navigation properties
func RegisterFilterExpandedPropertiesTests(suite *framework.TestSuite) {
	suite.AddTest(
		"Filter on collection navigation with any()",
		"Filter entities based on properties of collection navigation using any()",
		testFilterAnyOnNavigation,
	)

	suite.AddTest(
		"Filter on collection navigation with all()",
		"Filter entities using all() operator on collection navigation",
		testFilterAllOnNavigation,
	)

	suite.AddTest(
		"Filter with any() and complex condition",
		"Use any() with compound boolean expressions on navigation properties",
		testFilterAnyComplex,
	)

	suite.AddTest(
		"Expand with filter on expanded entities",
		"Apply $filter to expanded navigation collection",
		testExpandWithNestedFilter,
	)

	suite.AddTest(
		"Filter main and expanded entities",
		"Combine filter on main entity with filter on expanded entities",
		testFilterBothLevels,
	)

	suite.AddTest(
		"Any with string function on navigation",
		"Use string functions within any() lambda expression",
		testAnyWithStringFunction,
	)

	suite.AddTest(
		"Multiple any() filters on same navigation",
		"Apply multiple any() conditions on same collection navigation",
		testMultipleAnyFilters,
	)

	suite.AddTest(
		"Navigation filter with or condition",
		"Use or operator within any() lambda expression",
		testNavigationFilterOr,
	)

	suite.AddTest(
		"Nested condition in any() with function",
		"Combine functions and comparisons within any() expression",
		testNestedAnyCondition,
	)

	suite.AddTest(
		"Expand and filter same navigation property",
		"Apply both filter and expand to same navigation collection",
		testExpandAndFilterSameNav,
	)

	suite.AddTest(
		"Filter with not and any on navigation",
		"Use not operator with any() on navigation property",
		testNotAnyOnNavigation,
	)

	suite.AddTest(
		"Complex filter combining entity and navigation",
		"Combine entity property filters with navigation property filters",
		testComplexCombinedFilter,
	)
}

// --- helpers ---------------------------------------------------------------

// fetchProductsExpandDescriptions issues GET /Products?<query> (with Descriptions
// expanded) and returns the parsed collection, requiring HTTP 200.
func fetchProductsExpandDescriptions(ctx *framework.TestContext, query string) ([]map[string]interface{}, error) {
	resp, err := ctx.GET("/Products?" + query)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(resp.Body))
	}
	return ctx.ParseEntityCollection(resp)
}

// productDescriptions (extracts the expanded Descriptions collection) is shared
// from filter_oracle_helpers.go.

func descriptionLanguage(d map[string]interface{}) string {
	s, _ := d["LanguageKey"].(string)
	return s
}

func descriptionText(d map[string]interface{}) string {
	s, _ := d["Description"].(string)
	return s
}

func productIDValue(entity map[string]interface{}) (string, bool) {
	s, ok := entity["ID"].(string)
	return s, ok
}

// assertProductNavFilter verifies that GET /Products?$filter=<filter> returns
// exactly the set of products for which matches(product) is true, using the full
// (unfiltered) collection expanded with Descriptions as the oracle. This checks
// both soundness (every returned product matches) and completeness (no matching
// product is missing), so a server that ignores or mis-applies the filter fails.
func assertProductNavFilter(ctx *framework.TestContext, filter string, matches func(product map[string]interface{}) bool) error {
	all, err := fetchProductsExpandDescriptions(ctx, "$expand=Descriptions")
	if err != nil {
		return err
	}

	expected := map[string]bool{}
	for _, p := range all {
		id, ok := productIDValue(p)
		if !ok {
			return fmt.Errorf("product in baseline collection is missing its ID")
		}
		if matches(p) {
			expected[id] = true
		}
	}
	if len(expected) == 0 {
		return fmt.Errorf("test precondition failed: no seed product satisfies %q", filter)
	}

	got, err := fetchProductsExpandDescriptions(ctx, "$expand=Descriptions&$filter="+url.QueryEscape(filter))
	if err != nil {
		return err
	}

	gotIDs := map[string]bool{}
	for _, p := range got {
		id, ok := productIDValue(p)
		if !ok {
			return fmt.Errorf("returned product is missing its ID")
		}
		gotIDs[id] = true
		if !matches(p) {
			return fmt.Errorf("product %s was returned but does not satisfy %q (filter not applied correctly)", id, filter)
		}
	}
	for id := range expected {
		if !gotIDs[id] {
			return fmt.Errorf("product %s satisfies %q but was not returned (filter over-restricted or ignored)", id, filter)
		}
	}
	return nil
}

func anyDescription(p map[string]interface{}, pred func(d map[string]interface{}) bool) bool {
	for _, d := range productDescriptions(p) {
		if pred(d) {
			return true
		}
	}
	return false
}

func allDescriptions(p map[string]interface{}, pred func(d map[string]interface{}) bool) bool {
	for _, d := range productDescriptions(p) {
		if !pred(d) {
			return false
		}
	}
	return true
}

// --- tests -----------------------------------------------------------------

func testFilterAnyOnNavigation(ctx *framework.TestContext) error {
	return assertProductNavFilter(ctx, "Descriptions/any(d: d/LanguageKey eq 'EN')", func(p map[string]interface{}) bool {
		return anyDescription(p, func(d map[string]interface{}) bool { return descriptionLanguage(d) == "EN" })
	})
}

func testFilterAllOnNavigation(ctx *framework.TestContext) error {
	return assertProductNavFilter(ctx, "Descriptions/all(d: d/LanguageKey ne 'XX')", func(p map[string]interface{}) bool {
		return allDescriptions(p, func(d map[string]interface{}) bool { return descriptionLanguage(d) != "XX" })
	})
}

func testFilterAnyComplex(ctx *framework.TestContext) error {
	// contains() is case-sensitive (OData Part 2 §5.1.1.8); the seed EN
	// descriptions contain "laptop" in lower case.
	return assertProductNavFilter(ctx, "Descriptions/any(d: d/LanguageKey eq 'EN' and contains(d/Description, 'laptop'))", func(p map[string]interface{}) bool {
		return anyDescription(p, func(d map[string]interface{}) bool {
			return descriptionLanguage(d) == "EN" && strings.Contains(descriptionText(d), "laptop")
		})
	})
}

func testAnyWithStringFunction(ctx *framework.TestContext) error {
	return assertProductNavFilter(ctx, "Descriptions/any(d: contains(d/Description, 'laptop'))", func(p map[string]interface{}) bool {
		return anyDescription(p, func(d map[string]interface{}) bool { return strings.Contains(descriptionText(d), "laptop") })
	})
}

func testMultipleAnyFilters(ctx *framework.TestContext) error {
	filter := "Descriptions/any(d: d/LanguageKey eq 'EN') and Descriptions/any(d: d/LanguageKey eq 'DE')"
	return assertProductNavFilter(ctx, filter, func(p map[string]interface{}) bool {
		hasEN := anyDescription(p, func(d map[string]interface{}) bool { return descriptionLanguage(d) == "EN" })
		hasDE := anyDescription(p, func(d map[string]interface{}) bool { return descriptionLanguage(d) == "DE" })
		return hasEN && hasDE
	})
}

func testNavigationFilterOr(ctx *framework.TestContext) error {
	filter := "Descriptions/any(d: d/LanguageKey eq 'EN' or d/LanguageKey eq 'DE')"
	return assertProductNavFilter(ctx, filter, func(p map[string]interface{}) bool {
		return anyDescription(p, func(d map[string]interface{}) bool {
			return descriptionLanguage(d) == "EN" || descriptionLanguage(d) == "DE"
		})
	})
}

func testNestedAnyCondition(ctx *framework.TestContext) error {
	filter := "Descriptions/any(d: contains(d/Description, 'laptop') and d/LanguageKey eq 'EN')"
	return assertProductNavFilter(ctx, filter, func(p map[string]interface{}) bool {
		return anyDescription(p, func(d map[string]interface{}) bool {
			return strings.Contains(descriptionText(d), "laptop") && descriptionLanguage(d) == "EN"
		})
	})
}

func testNotAnyOnNavigation(ctx *framework.TestContext) error {
	return assertProductNavFilter(ctx, "not Descriptions/any(d: d/LanguageKey eq 'FR')", func(p map[string]interface{}) bool {
		return !anyDescription(p, func(d map[string]interface{}) bool { return descriptionLanguage(d) == "FR" })
	})
}

func testComplexCombinedFilter(ctx *framework.TestContext) error {
	filter := "Price gt 100 and Descriptions/any(d: d/LanguageKey eq 'EN')"
	return assertProductNavFilter(ctx, filter, func(p map[string]interface{}) bool {
		price, ok := p["Price"].(float64)
		if !ok || price <= 100 {
			return false
		}
		return anyDescription(p, func(d map[string]interface{}) bool { return descriptionLanguage(d) == "EN" })
	})
}

func testExpandAndFilterSameNav(ctx *framework.TestContext) error {
	// Top-level filter selects products that have an EN description; the (unfiltered)
	// $expand=Descriptions must still include every description of each match.
	filter := "Descriptions/any(d: d/LanguageKey eq 'EN')"
	if err := assertProductNavFilter(ctx, filter, func(p map[string]interface{}) bool {
		return anyDescription(p, func(d map[string]interface{}) bool { return descriptionLanguage(d) == "EN" })
	}); err != nil {
		return err
	}

	// Confirm the expansion is actually present on the matched products.
	got, err := fetchProductsExpandDescriptions(ctx, "$expand=Descriptions&$filter="+url.QueryEscape(filter))
	if err != nil {
		return err
	}
	for _, p := range got {
		if _, ok := p["Descriptions"]; !ok {
			id, _ := productIDValue(p)
			return fmt.Errorf("product %s was matched but its Descriptions navigation was not expanded", id)
		}
	}
	return nil
}

func testExpandWithNestedFilter(ctx *framework.TestContext) error {
	// $expand=Descriptions($filter=LanguageKey eq 'EN') with no top-level filter:
	// every expanded description that comes back must be EN, and at least one must
	// be returned (so we know the nested filter did not strip everything).
	return assertNestedDescriptionFilter(ctx,
		"$expand="+url.QueryEscape("Descriptions($filter=LanguageKey eq 'EN')"),
		func(d map[string]interface{}) bool { return descriptionLanguage(d) == "EN" },
		"LanguageKey eq 'EN'")
}

func testFilterBothLevels(ctx *framework.TestContext) error {
	// Top-level filter (Price gt 100) plus a nested $filter on the expansion.
	// Every returned product must have Price > 100, and every expanded description
	// must be EN.
	query := "$filter=" + url.QueryEscape("Price gt 100") +
		"&$expand=" + url.QueryEscape("Descriptions($filter=LanguageKey eq 'EN')")
	items, err := fetchProductsExpandDescriptions(ctx, query)
	if err != nil {
		return err
	}
	if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
		return fmt.Errorf("seed data has products priced over 100: %w", err)
	}

	sawDescription := false
	for _, p := range items {
		price, ok := p["Price"].(float64)
		if !ok || price <= 100 {
			id, _ := productIDValue(p)
			return fmt.Errorf("product %s with Price=%v was returned but does not satisfy Price gt 100", id, p["Price"])
		}
		for _, d := range productDescriptions(p) {
			sawDescription = true
			if descriptionLanguage(d) != "EN" {
				id, _ := productIDValue(p)
				return fmt.Errorf("product %s has an expanded description with LanguageKey=%q; nested $filter=LanguageKey eq 'EN' was not applied", id, descriptionLanguage(d))
			}
		}
	}
	if !sawDescription {
		return fmt.Errorf("no expanded descriptions were returned; cannot confirm the nested $filter was applied")
	}
	return nil
}

// assertNestedDescriptionFilter verifies that for GET /Products?<query>, every
// expanded Description satisfies pred and at least one description is returned.
func assertNestedDescriptionFilter(ctx *framework.TestContext, query string, pred func(d map[string]interface{}) bool, predDesc string) error {
	items, err := fetchProductsExpandDescriptions(ctx, query)
	if err != nil {
		return err
	}

	sawDescription := false
	for _, p := range items {
		for _, d := range productDescriptions(p) {
			sawDescription = true
			if !pred(d) {
				id, _ := productIDValue(p)
				return fmt.Errorf("product %s has an expanded description with LanguageKey=%q that violates nested $filter (%s)", id, descriptionLanguage(d), predDesc)
			}
		}
	}
	if !sawDescription {
		return fmt.Errorf("no expanded descriptions were returned; cannot confirm the nested $filter (%s) was applied", predDesc)
	}
	return nil
}
