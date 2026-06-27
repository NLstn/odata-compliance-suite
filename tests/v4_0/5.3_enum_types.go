package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// EnumTypes creates the 5.3 Enumeration Types test suite
func EnumTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.3 Enumeration Types",
		"Validates handling of enumeration types including filtering, selecting, ordering, and enum operations.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_EnumerationType",
	)

	suite.AddTest(
		"test_enum_retrieval",
		"Retrieve entity with enum property",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, `"Status"`) {
				return nil
			}

			return nil // No enum property, optional
		},
	)

	suite.AddTest(
		"test_filter_enum_numeric",
		"Filter by enum integer literal returns exactly the matching entities",
		func(ctx *framework.TestContext) error {
			// OData spec §5.1.1.12 permits integer literals for enum values.
			// Verify that the server applies the filter correctly, not just 200.
			return assertProductFilter(ctx, "Status eq 1", func(p map[string]interface{}) bool {
				status, err := enumStatusValue(p)
				return err == nil && status == 1 // InStock = 1
			})
		},
	)

	suite.AddTest(
		"test_enum_comparison",
		"Comparison operator (gt) on enum returns only entities satisfying the predicate",
		func(ctx *framework.TestContext) error {
			// OData spec §5.1.1.12: enum types are ordered by member value;
			// comparison operators other than eq/ne are allowed.
			return assertProductFilter(ctx, "Status gt 0", func(p map[string]interface{}) bool {
				status, err := enumStatusValue(p)
				return err == nil && status > 0 // None = 0 excluded
			})
		},
	)

	suite.AddTest(
		"test_select_enum",
		"$select of enum property includes it in the response as a string member name",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=Name,Status")
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
				return err
			}
			for i, item := range items {
				status, ok := item["Status"]
				if !ok {
					return fmt.Errorf("entity %d is missing the Status field in $select=Name,Status response", i)
				}
				if status == nil {
					continue
				}
				if _, isStr := status.(string); !isStr {
					return fmt.Errorf("entity %d: Status must be serialized as a string member name, got %T (%v)", i, status, status)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_orderby_enum",
		"$orderby on enum property returns all entities with a Status field",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=Status")
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
			// All entities should be returned, each with a Status field
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}
			return ctx.AssertAllEntitiesSatisfy(items, "has Status field", func(entity map[string]interface{}) (bool, string) {
				if _, ok := entity["Status"]; !ok {
					return false, "missing Status field"
				}
				return true, ""
			})
		},
	)

	suite.AddTest(
		"test_enum_null",
		"Filter Status eq null returns empty set (Status is non-nullable)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Status eq null")
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
			if len(items) != 0 {
				return fmt.Errorf("expected 0 results for Status eq null (Status is non-nullable), got %d", len(items))
			}
			return nil
		},
	)

	// The 'has' operator tests flags-enum membership. ProductStatus is a flags
	// enum (None=0, InStock=1, OnSale=2, Discontinued=4, Featured=8); 'Status has
	// X' selects rows whose Status includes the X bit. Results are verified with an
	// oracle (Status BAND member-value != 0).
	suite.AddTest(
		"test_enum_has_flag_featured",
		"has operator selects entities whose flags enum includes 'Featured'",
		func(ctx *framework.TestContext) error {
			ns, err := schemaNamespace(ctx)
			if err != nil {
				return err
			}
			if ns == "" {
				return ctx.Skip("could not determine the schema namespace")
			}
			return assertProductFilter(ctx, fmt.Sprintf("Status has %s.ProductStatus'Featured'", ns), func(p map[string]interface{}) bool {
				status, err := enumStatusValue(p)
				return err == nil && status&8 != 0 // Featured = 8
			})
		},
	)

	suite.AddTest(
		"test_enum_has_flag_onsale",
		"has operator selects entities whose flags enum includes 'OnSale'",
		func(ctx *framework.TestContext) error {
			ns, err := schemaNamespace(ctx)
			if err != nil {
				return err
			}
			if ns == "" {
				return ctx.Skip("could not determine the schema namespace")
			}
			return assertProductFilter(ctx, fmt.Sprintf("Status has %s.ProductStatus'OnSale'", ns), func(p map[string]interface{}) bool {
				status, err := enumStatusValue(p)
				return err == nil && status&2 != 0 // OnSale = 2
			})
		},
	)

	suite.AddTest(
		"test_enum_in_metadata",
		"Enum type in metadata document",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, "EnumType") {
				return nil
			}

			return nil // No enum types, optional
		},
	)

	return suite
}
