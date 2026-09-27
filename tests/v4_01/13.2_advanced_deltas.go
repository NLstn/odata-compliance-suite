package v4_01

import (
	"fmt"
	"maps"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// AdvancedDeltas checks features required at the OData 4.01 advanced level.
func AdvancedDeltas() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"13.2 OData 4.01 Advanced Deltas",
		"Validates 4.01 features that are required at the advanced conformance level.",
		"https://docs.oasis-open.org/odata/odata/v4.01/os/part1-protocol/odata-v4.01-os-part1-protocol.html#sec_OData401AdvancedConformanceLevel",
	)

	suite.AddTestWithCapabilities(
		"test_filtered_collection_count_in_expression",
		"A filtered collection count in a common expression selects exactly the matching categories",
		[]framework.RequiredCapability{
			framework.Require(framework.CapFilter, "Categories"),
		},
		func(ctx *framework.TestContext) error {
			headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			products, err := collectEntityCollection(ctx, "/Products?$select=ID,CategoryID,Price", headers...)
			if err != nil {
				return fmt.Errorf("baseline products: %w", err)
			}
			expected := map[string]bool{}
			for _, product := range products {
				price, ok := product["Price"].(float64)
				if !ok {
					return fmt.Errorf("baseline product has non-numeric Price %v", product["Price"])
				}
				if price <= 100 {
					continue
				}
				if categoryID := product["CategoryID"]; categoryID != nil {
					expected[fmt.Sprint(categoryID)] = true
				}
			}
			if len(expected) == 0 {
				return fmt.Errorf("reference products contain no category with Price > 100")
			}
			expression := "Products/$count($filter=Price gt 100) gt 0"
			path := "/Categories?$filter=" + url.QueryEscape(expression) + "&$select=ID"
			categories, err := collectEntityCollection(ctx, path, headers...)
			if err != nil {
				return fmt.Errorf("filtered collection count expression: %w", err)
			}
			actual, err := entityIDs(categories)
			if err != nil {
				return err
			}
			if !maps.Equal(actual, expected) {
				return fmt.Errorf("filtered collection count returned category IDs %v, want %v", actual, expected)
			}
			return nil
		},
	)

	return suite
}
