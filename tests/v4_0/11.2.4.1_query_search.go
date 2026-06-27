package v4_0

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// QuerySearch creates the 11.2.4.1 System Query Option $search test suite
func QuerySearch() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.4.1 System Query Option $search",
		"Tests $search query option for free-text search according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionsearch",
	)

	// Test 1: a single term returns the products that contain it
	suite.AddTest(
		"test_basic_search",
		"$search with a single term returns the matching products",
		func(ctx *framework.TestContext) error {
			names, err := searchProductNames(ctx, "Laptop")
			if err != nil {
				return err
			}
			// Both products with "Laptop" in their name must be returned.
			for _, want := range []string{"Laptop", "Premium Laptop Pro"} {
				if !names[want] {
					return fmt.Errorf("$search=Laptop did not return %q; got %v", want, keys(names))
				}
			}
			return nil
		},
	)

	// Test 2: multiple terms are combined with implicit AND (narrows the result)
	suite.AddTest(
		"test_search_multiple_terms",
		"$search with multiple terms (implicit AND) narrows to products matching all terms",
		func(ctx *framework.TestContext) error {
			laptop, err := searchProductNames(ctx, "Laptop")
			if err != nil {
				return err
			}
			both, err := searchProductNames(ctx, "Laptop Pro")
			if err != nil {
				return err
			}

			// Implicit AND must return a subset of the single-term result and must
			// still include the product matching both terms.
			if !both["Premium Laptop Pro"] {
				return fmt.Errorf("$search=\"Laptop Pro\" should match 'Premium Laptop Pro'; got %v", keys(both))
			}
			if len(both) > len(laptop) {
				return fmt.Errorf("implicit-AND result (%d) is larger than the single-term result (%d); AND should narrow", len(both), len(laptop))
			}
			for name := range both {
				if !laptop[name] {
					return fmt.Errorf("product %q matched \"Laptop Pro\" but not \"Laptop\"; implicit AND is not narrowing correctly", name)
				}
			}
			return nil
		},
	)

	// Test 3: explicit AND of disjoint terms returns no results
	suite.AddTest(
		"test_search_and_operator",
		"$search with AND of disjoint terms returns the empty intersection",
		func(ctx *framework.TestContext) error {
			names, err := searchProductNames(ctx, "Laptop AND Mouse")
			if err != nil {
				return err
			}
			// No seed product is both a Laptop and a Mouse.
			if len(names) != 0 {
				return fmt.Errorf("$search=\"Laptop AND Mouse\" should return 0 products (disjoint terms), got %v", keys(names))
			}
			return nil
		},
	)

	// Test 4: $search with OR operator returns the union of both terms
	suite.AddTest(
		"test_search_or_operator",
		"$search with OR operator returns results matching either term",
		func(ctx *framework.TestContext) error {
			laptop, err := searchProductNames(ctx, "Laptop")
			if err != nil {
				return err
			}
			mouse, err := searchProductNames(ctx, "Mouse")
			if err != nil {
				return err
			}
			if len(laptop) == 0 || len(mouse) == 0 {
				return fmt.Errorf("single-term searches returned no results; seed data may be missing")
			}

			or, err := searchProductNames(ctx, "Laptop OR Mouse")
			if err != nil {
				return err
			}

			// The OR result must contain the union of both term results.
			for name := range laptop {
				if !or[name] {
					return fmt.Errorf("OR result is missing %q from the 'Laptop' set; OR is not a union", name)
				}
			}
			for name := range mouse {
				if !or[name] {
					return fmt.Errorf("OR result is missing %q from the 'Mouse' set; OR is not a union", name)
				}
			}
			// "Laptop" and "Mouse" are disjoint in the seed data, so the union size
			// must be exactly the sum.
			if len(or) != len(laptop)+len(mouse) {
				return fmt.Errorf("OR result size=%d, expected %d (Laptop=%d + Mouse=%d, disjoint)", len(or), len(laptop)+len(mouse), len(laptop), len(mouse))
			}
			return nil
		},
	)

	// Test 5: a quoted phrase matches the phrase, not the individual terms
	suite.AddTest(
		"test_search_phrase",
		"$search with a quoted phrase matches the whole phrase",
		func(ctx *framework.TestContext) error {
			phrase, err := searchProductNames(ctx, `"Wireless Mouse"`)
			if err != nil {
				return err
			}
			// The phrase must match "Wireless Mouse"...
			if !phrase["Wireless Mouse"] {
				return fmt.Errorf("phrase search did not return 'Wireless Mouse'; got %v", keys(phrase))
			}
			// ...but not "Gaming Mouse Ultra", which contains "Mouse" but not the
			// contiguous phrase "Wireless Mouse".
			if phrase["Gaming Mouse Ultra"] {
				return fmt.Errorf("phrase search matched 'Gaming Mouse Ultra'; a phrase must not match on individual terms")
			}
			return nil
		},
	)

	// Test 6: $search combined with $filter applies both constraints
	suite.AddTest(
		"test_search_with_filter",
		"$search combined with $filter applies both constraints",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$search=Laptop&$filter=Price gt 100&$select=Name,Price")
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("expected status 200 but received %d", resp.StatusCode)
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			// Both seed laptops are priced over 100, so the result must be non-empty
			// and every product must satisfy the $filter.
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Price gt 100", func(entity map[string]interface{}) (bool, string) {
				price, ok := entity["Price"].(float64)
				if !ok {
					return false, "Price missing or not numeric"
				}
				if price <= 100 {
					return false, fmt.Sprintf("Price=%.2f is not > 100; $filter not applied alongside $search", price)
				}
				return true, ""
			})
		},
	)

	return suite
}

// searchProductNames runs GET /Products?$search=<expr> and returns the set of
// returned product Names. It requires HTTP 200.
func searchProductNames(ctx *framework.TestContext, search string) (map[string]bool, error) {
	resp, err := ctx.GET("/Products?$select=Name&$search=" + url.QueryEscape(search))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("$search=%q: expected 200, got %d: %s", search, resp.StatusCode, string(resp.Body))
	}
	items, err := ctx.ParseEntityCollection(resp)
	if err != nil {
		return nil, err
	}
	names := make(map[string]bool, len(items))
	for _, item := range items {
		if name, ok := item["Name"].(string); ok {
			names[name] = true
		}
	}
	return names, nil
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
