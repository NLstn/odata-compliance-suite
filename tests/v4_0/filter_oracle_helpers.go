package v4_0

import (
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// productTypeNamespacePattern captures the schema namespace from the Products
// entity set's qualified EntityType (e.g. EntityType="ComplianceService.Product").
var productTypeNamespacePattern = regexp.MustCompile(`EntityType="([^"]+)\.Product"`)

// schemaNamespace discovers the model's schema namespace from $metadata. Returns
// "" when it cannot be determined (the model has no Product entity type).
func schemaNamespace(ctx *framework.TestContext) (string, error) {
	resp, err := ctx.GET("/$metadata")
	if err != nil {
		return "", err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return "", err
	}
	m := productTypeNamespacePattern.FindSubmatch(resp.Body)
	if m == nil {
		return "", nil
	}
	return string(m[1]), nil
}

// fetchAllProducts returns every Product entity as a decoded JSON object. It is
// used to build a client-side oracle for $filter / arithmetic / date-function
// semantics: the expected result of a server-side filter is computed in Go from
// the full set and compared against what the server actually returns.
func fetchAllProducts(ctx *framework.TestContext) ([]map[string]interface{}, error) {
	resp, err := ctx.GET("/Products?$top=1000")
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, err
	}
	return ctx.ParseEntityCollection(resp)
}

// assertProductFilter runs /Products?$filter=<expr> and asserts the returned
// entity-ID set is exactly the set of products for which want() returns true,
// where want() is evaluated in Go against a full fetch. This verifies both
// soundness (every returned row satisfies the predicate) and completeness (no
// satisfying row is missing) — far stronger than asserting only HTTP 200.
//
// want() must return false for rows where the relevant property is null, mirroring
// OData three-valued logic (a comparison involving null is excluded from results).
func assertProductFilter(ctx *framework.TestContext, expr string, want func(map[string]interface{}) bool) error {
	all, err := fetchAllProducts(ctx)
	if err != nil {
		return err
	}
	return assertProductFilterFrom(ctx, all, expr, want)
}

// assertProductLambdaFilter is assertProductFilter where the oracle needs each
// product's related Descriptions collection (for any()/all() lambda predicates).
// The full set is fetched with $expand=Descriptions so want() can inspect them.
func assertProductLambdaFilter(ctx *framework.TestContext, expr string, want func(map[string]interface{}) bool) error {
	resp, err := ctx.GET("/Products?$expand=Descriptions&$top=1000")
	if err != nil {
		return err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return err
	}
	all, err := ctx.ParseEntityCollection(resp)
	if err != nil {
		return err
	}
	return assertProductFilterFrom(ctx, all, expr, want)
}

// assertProductFilterFrom compares the server's $filter result against the oracle
// set computed by applying want() to the supplied full collection.
func assertProductFilterFrom(ctx *framework.TestContext, all []map[string]interface{}, expr string, want func(map[string]interface{}) bool) error {
	expected := map[string]bool{}
	for _, p := range all {
		if want(p) {
			expected[productID(p)] = true
		}
	}

	resp, err := ctx.GET("/Products?$filter=" + url.QueryEscape(expr))
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
	got := map[string]bool{}
	for _, p := range items {
		got[productID(p)] = true
	}

	for id := range expected {
		if !got[id] {
			return fmt.Errorf("filter %q: product %s satisfies the predicate but was not returned (got %d rows, expected %d)", expr, id, len(got), len(expected))
		}
	}
	for id := range got {
		if !expected[id] {
			return fmt.Errorf("filter %q: product %s was returned but does not satisfy the predicate (got %d rows, expected %d)", expr, id, len(got), len(expected))
		}
	}
	return nil
}

func productID(p map[string]interface{}) string {
	return fmt.Sprintf("%v", p["ID"])
}

// productTime parses a non-null DateTimeOffset ("2024-01-15T10:30:00Z") or Date
// ("2024-01-15") field. ok is false when the field is null/absent/unparseable,
// so callers can treat null rows as non-matching.
func productTime(p map[string]interface{}, field string) (t time.Time, ok bool) {
	s, isStr := p[field].(string)
	if !isStr || s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, s); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

// productFloat reads a numeric field as float64. ok is false when null/absent.
func productFloat(p map[string]interface{}, field string) (float64, bool) {
	f, ok := p[field].(float64)
	return f, ok
}

// productString reads a string field; returns "" when null/absent.
func productString(p map[string]interface{}, field string) string {
	s, _ := p[field].(string)
	return s
}

// productDescriptions returns a product's expanded Descriptions collection as a
// slice of decoded objects (empty when absent).
func productDescriptions(p map[string]interface{}) []map[string]interface{} {
	raw, ok := p["Descriptions"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		if d, ok := item.(map[string]interface{}); ok {
			out = append(out, d)
		}
	}
	return out
}
