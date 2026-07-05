package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// CollectionProperties creates the 5.1.3 Collection Properties test suite
func CollectionProperties() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.3 Collection Properties",
		"Validates handling of collection-valued navigation properties, lambda operators (any/all), and $count on collections.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_CollectionValuedProperties",
	)

	suite.AddTest(
		"test_expand_collection",
		"Expand collection-valued navigation property includes Descriptions as an array on each entity",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$expand=Descriptions")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if len(result.Value) == 0 {
				return fmt.Errorf("$expand=Descriptions returned empty value array")
			}
			for i, entity := range result.Value {
				raw, ok := entity["Descriptions"]
				if !ok {
					return fmt.Errorf("entity %d missing Descriptions key after $expand", i)
				}
				if _, ok := raw.([]interface{}); !ok {
					return fmt.Errorf("entity %d Descriptions is not an array (got %T)", i, raw)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_collection_any_operator",
		"Filter with collection 'any' operator: every returned product has at least one description containing 'Laptop'",
		func(ctx *framework.TestContext) error {
			return assertProductLambdaFilter(ctx, "Descriptions/any(d:contains(d/Description,'Laptop'))",
				func(p map[string]interface{}) bool {
					for _, d := range productDescriptions(p) {
						if desc, _ := d["Description"].(string); strings.Contains(desc, "Laptop") {
							return true
						}
					}
					return false
				})
		},
	)

	suite.AddTest(
		"test_collection_all_operator",
		"Filter with collection 'all' operator: every returned product has all descriptions non-null",
		func(ctx *framework.TestContext) error {
			return assertProductLambdaFilter(ctx, "Descriptions/all(d:d/Description ne null)",
				func(p map[string]interface{}) bool {
					descs := productDescriptions(p)
					// Vacuous truth: products with no descriptions satisfy all() trivially.
					for _, d := range descs {
						if desc := d["Description"]; desc == nil {
							return false
						}
					}
					return true
				})
		},
	)

	suite.AddTest(
		"test_collection_count",
		"Count items in collection navigation",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions/$count")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Should return plain text number
			return nil
		},
	)

	suite.AddTest(
		"test_filter_with_count",
		"Filter using $count in expression: every returned product has more than 1 description",
		func(ctx *framework.TestContext) error {
			return assertProductLambdaFilter(ctx, "Descriptions/$count gt 1",
				func(p map[string]interface{}) bool {
					return len(productDescriptions(p)) > 1
				})
		},
	)

	suite.AddTest(
		"test_expand_with_count",
		"Expand with $count option",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$expand=Descriptions($count=true)")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return nil
		},
	)

	suite.AddTest(
		"test_expand_with_filter",
		"Expand with $filter on collection: every expanded description has LanguageKey='EN'",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$expand=Descriptions($filter=LanguageKey eq 'EN')")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			var result struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			for i, entity := range result.Value {
				for j, d := range productDescriptions(entity) {
					lk, _ := d["LanguageKey"].(string)
					if lk != "EN" {
						return fmt.Errorf("entity %d description %d has LanguageKey=%q, expected 'EN'", i, j, lk)
					}
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_select_navigation_property",
		"Select navigation property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=Name,Descriptions")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return nil
		},
	)

	return suite
}
