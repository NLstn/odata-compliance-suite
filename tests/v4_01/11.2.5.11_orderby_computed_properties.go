package v4_01

import (
	"fmt"
	"net/http"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// OrderByComputedProperties creates the 11.2.5.11 OrderBy with Computed Properties test suite
func OrderByComputedProperties() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.11 OrderBy with Computed Properties",
		"Validates $orderby functionality with computed properties from $compute query option (OData v4.01 feature).",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptioncompute",
	)

	// Test 1: Compute a property and order by it — verify ascending sort order.
	suite.AddTest(
		"test_orderby_computed",
		"OrderBy computed property returns entities in ascending computed-value order",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.1 as TaxedPrice&$orderby=TaxedPrice")
			if err != nil {
				return err
			}
			if err := requireStatusOK(resp); err != nil {
				return err
			}
			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}
			if err := ensureComputedProperties(entities, "TaxedPrice"); err != nil {
				return err
			}
			return assertComputedSortOrder(entities, "TaxedPrice", true)
		},
	)

	// Test 2: Multiple computed properties — verify descending sort on DiscountPrice.
	suite.AddTest(
		"test_orderby_multiple_computed",
		"OrderBy with multiple computed properties sorts by the specified computed field",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 0.9 as DiscountPrice,Price mul 1.1 as TaxedPrice&$orderby=DiscountPrice desc")
			if err != nil {
				return err
			}
			if err := requireStatusOK(resp); err != nil {
				return err
			}
			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}
			if err := ensureComputedProperties(entities, "DiscountPrice", "TaxedPrice"); err != nil {
				return err
			}
			return assertComputedSortOrder(entities, "DiscountPrice", false)
		},
	)

	// Test 3: Descending sort on computed property.
	suite.AddTest(
		"test_orderby_computed_desc",
		"OrderBy computed with desc direction returns entities in descending order",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 2 as DoublePrice&$orderby=DoublePrice desc")
			if err != nil {
				return err
			}
			if err := requireStatusOK(resp); err != nil {
				return err
			}
			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}
			if err := ensureComputedProperties(entities, "DoublePrice"); err != nil {
				return err
			}
			return assertComputedSortOrder(entities, "DoublePrice", false)
		},
	)

	// Test 4: OrderBy mixing computed and regular properties — verify the
	// compound sort: CategoryID ascending, then MarkedUpPrice descending
	// within each CategoryID group.
	suite.AddTest(
		"test_orderby_mixed",
		"OrderBy mixing computed and regular properties respects the compound sort order",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.2 as MarkedUpPrice&$orderby=CategoryID,MarkedUpPrice desc")
			if err != nil {
				return err
			}

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			if err := ensureComputedProperties(entities, "MarkedUpPrice"); err != nil {
				return err
			}
			return assertGroupedSortOrder(entities, "CategoryID", "MarkedUpPrice", false)
		},
	)

	// Test 5: OrderBy computed with select — verify ascending sort order.
	suite.AddTest(
		"test_orderby_computed_with_select",
		"OrderBy computed with $select returns entities in ascending computed order",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.08 as FinalPrice&$select=Name,FinalPrice&$orderby=FinalPrice")
			if err != nil {
				return err
			}

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			if err := ensureComputedProperties(entities, "FinalPrice"); err != nil {
				return err
			}
			return assertComputedSortOrder(entities, "FinalPrice", true)
		},
	)

	// Test 6: OrderBy computed with filter — every entity must satisfy
	// SalePrice gt 50 AND entities must be in ascending SalePrice order.
	suite.AddTest(
		"test_orderby_computed_with_filter",
		"OrderBy computed with $filter: results satisfy filter and are sorted",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 0.8 as SalePrice&$filter=SalePrice gt 50&$orderby=SalePrice")
			if err != nil {
				return err
			}
			if err := requireStatusOK(resp); err != nil {
				return err
			}
			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}
			if err := ensureComputedProperties(entities, "SalePrice"); err != nil {
				return err
			}
			// Verify filter predicate satisfied.
			for i, e := range entities {
				sp, err := entityFloat(e, "SalePrice")
				if err != nil {
					return fmt.Errorf("entity %d SalePrice: %w", i, err)
				}
				if sp <= 50 {
					return framework.NewError(
						fmt.Sprintf("entity %d has SalePrice=%v but filter was SalePrice gt 50", i, sp))
				}
			}
			return assertComputedSortOrder(entities, "SalePrice", true)
		},
	)

	// Test 7: OrderBy computed with top — verify descending order and the
	// $top=3 bound.
	suite.AddTest(
		"test_orderby_computed_with_top",
		"OrderBy computed with $top respects both the sort order and the page size",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price div 2 as HalfPrice&$orderby=HalfPrice desc&$top=3")
			if err != nil {
				return err
			}

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			if err := ensureComputedProperties(entities, "HalfPrice"); err != nil {
				return err
			}
			if len(entities) > 3 {
				return framework.NewError(fmt.Sprintf("$top=3 but got %d entities", len(entities)))
			}
			return assertComputedSortOrder(entities, "HalfPrice", false)
		},
	)

	// Test 8: OrderBy regular property still works with compute present —
	// verify Name is actually in ascending order.
	suite.AddTest(
		"test_orderby_regular_with_compute",
		"OrderBy regular property with compute present sorts by the regular property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.5 as HighPrice&$orderby=Name")
			if err != nil {
				return err
			}

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			if err := ensureComputedProperties(entities, "HighPrice"); err != nil {
				return err
			}
			return assertStringSortOrder(entities, "Name", true)
		},
	)

	// Test 9: Response includes computed properties when ordered — verify
	// the computed value is actually Price*2, not just present.
	suite.AddTest(
		"test_response_includes_computed",
		"Response includes a correctly-computed DoublePrice value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 2 as DoublePrice&$select=Name,Price,DoublePrice&$orderby=DoublePrice&$top=1")
			if err != nil {
				return err
			}

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			if err := ensureComputedProperties(entities, "DoublePrice"); err != nil {
				return err
			}
			price, err := entityFloat(entities[0], "Price")
			if err != nil {
				return fmt.Errorf("entity Price: %w", err)
			}
			doublePrice, err := entityFloat(entities[0], "DoublePrice")
			if err != nil {
				return fmt.Errorf("entity DoublePrice: %w", err)
			}
			if diff := doublePrice - price*2; diff > 0.01 || diff < -0.01 {
				return framework.NewError(fmt.Sprintf("DoublePrice=%v does not equal Price*2=%v", doublePrice, price*2))
			}
			return nil
		},
	)

	// Test 10: OrderBy without including computed in select
	suite.AddTest(
		"test_orderby_computed_not_selected",
		"OrderBy computed not in $select",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.3 as MarkedPrice&$select=Name,Price&$orderby=MarkedPrice")
			if err != nil {
				return err
			}

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			for i, entity := range entities {
				if _, ok := entity["MarkedPrice"]; ok {
					return framework.NewError(fmt.Sprintf("entity %d unexpectedly included computed property \"MarkedPrice\"", i))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_orderby_computed_version_negotiation_4_01_vs_4_0",
		"$orderby on computed properties is accepted with OData-MaxVersion 4.01 and 4.0",
		func(ctx *framework.TestContext) error {
			query := "/Products?$compute=Price mul 2 as DoublePrice&$orderby=DoublePrice&$top=1"

			v401Headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			v401Resp, err := ctx.GET(query, v401Headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(v401Resp, http.StatusOK); err != nil {
				return framework.NewError(fmt.Sprintf("4.01 negotiated computed-orderby request should succeed: %v", err))
			}

			v40Headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.0"}}
			v40Resp, err := ctx.GET(query, v40Headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(v40Resp, http.StatusOK); err != nil {
				return framework.NewError(fmt.Sprintf("supported 4.01 URL syntax must work regardless of OData-MaxVersion: %v", err))
			}

			return nil
		},
	)

	return suite
}
