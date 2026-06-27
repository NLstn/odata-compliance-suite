package v4_0

import (
	"fmt"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// AdvancedApply creates the 11.2.5.4.1 Advanced $apply Transformations test suite
func AdvancedApply() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.4.1 Advanced $apply Transformations",
		"Tests advanced $apply query option transformations including nested groupby, multiple aggregations, filter before/after aggregation, and complex transformation pipelines.",
		"https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/odata-data-aggregation-ext-v4.0.html",
	)

	// Test 1: Multiple aggregations in single aggregate
	suite.AddTest(
		"test_multiple_aggregations",
		"Multiple aggregations compute correct sum/average/max",
		func(ctx *framework.TestContext) error {
			stats, err := computePriceStats(ctx, nil)
			if err != nil {
				return err
			}
			row, err := applyAggregateRow(ctx, "aggregate(Price with sum as TotalPrice,Price with average as AvgPrice,Price with max as MaxPrice)")
			if err != nil {
				return err
			}
			if err := assertNumField(row, "TotalPrice", stats.sum); err != nil {
				return err
			}
			if err := assertNumField(row, "AvgPrice", stats.avg); err != nil {
				return err
			}
			return assertNumField(row, "MaxPrice", stats.max)
		},
	)

	// Test 2: groupby with multiple properties
	suite.AddTest(
		"test_groupby_multiple_properties",
		"groupby with multiple grouping properties",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("groupby((CategoryID,Status))")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 3: groupby with multiple aggregate methods
	suite.AddTest(
		"test_groupby_with_multiple_aggregates",
		"groupby aggregates partition the data (group sums/counts reconcile to totals)",
		func(ctx *framework.TestContext) error {
			stats, err := computePriceStats(ctx, nil)
			if err != nil {
				return err
			}
			rows, err := applyRows(ctx, "groupby((CategoryID),aggregate(Price with sum as Total,Price with average as Average,$count as Count))")
			if err != nil {
				return err
			}
			var totalSum float64
			var totalCount int
			for _, row := range rows {
				groupTotal, err := numField(row, "Total")
				if err != nil {
					return err
				}
				groupCount, err := numField(row, "Count")
				if err != nil {
					return err
				}
				groupAvg, err := numField(row, "Average")
				if err != nil {
					return err
				}
				// Per-group average must be consistent with its own sum/count.
				if groupCount > 0 && !aggApproxEqual(groupAvg, groupTotal/groupCount) {
					return fmt.Errorf("group average %v inconsistent with Total %v / Count %v", groupAvg, groupTotal, groupCount)
				}
				totalSum += groupTotal
				totalCount += int(groupCount)
			}
			// Groups must partition the data: sums and counts reconcile to the whole.
			if !aggApproxEqual(totalSum, stats.sum) {
				return fmt.Errorf("group Totals sum to %v, expected overall %v", totalSum, stats.sum)
			}
			if totalCount != stats.count {
				return fmt.Errorf("group Counts sum to %d, expected overall %d", totalCount, stats.count)
			}
			return nil
		},
	)

	// Test 4: Filter before aggregation
	suite.AddTest(
		"test_filter_before_aggregate",
		"filter() before aggregate() restricts the aggregated rows",
		func(ctx *framework.TestContext) error {
			stats, err := computePriceStats(ctx, func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > 50
			})
			if err != nil {
				return err
			}
			row, err := applyAggregateRow(ctx, "filter(Price gt 50)/aggregate(Price with sum as Total)")
			if err != nil {
				return err
			}
			return assertNumField(row, "Total", stats.sum)
		},
	)

	// Test 5: Filter before groupby
	suite.AddTest(
		"test_filter_before_groupby",
		"Filter before groupby transformation",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("filter(Price gt 50)/groupby((CategoryID),aggregate($count as Count))")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 6: Multiple transformations in sequence
	suite.AddTest(
		"test_transformation_pipeline",
		"Transformation pipeline",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("filter(Price gt 10)/groupby((CategoryID))/filter($count gt 1)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 7: Aggregate with countdistinct
	suite.AddTest(
		"test_countdistinct",
		"countdistinct counts the distinct CategoryID values",
		func(ctx *framework.TestContext) error {
			sums, _, err := computeCategoryGroups(ctx)
			if err != nil {
				return err
			}
			row, err := applyAggregateRow(ctx, "aggregate(CategoryID with countdistinct as UniqueCategories)")
			if err != nil {
				return err
			}
			return assertNumField(row, "UniqueCategories", float64(len(sums)))
		},
	)

	// Test 8: groupby followed by filter
	suite.AddTest(
		"test_groupby_then_filter",
		"filter() after groupby/aggregate keeps only the groups whose aggregate matches",
		func(ctx *framework.TestContext) error {
			sums, _, err := computeCategoryGroups(ctx)
			if err != nil {
				return err
			}
			expected := map[string]float64{}
			for cat, sum := range sums {
				if sum > 100 {
					expected[cat] = sum
				}
			}
			rows, err := applyRows(ctx, "groupby((CategoryID),aggregate(Price with sum as Total))/filter(Total gt 100)")
			if err != nil {
				return err
			}
			if len(rows) != len(expected) {
				return fmt.Errorf("expected %d groups with Total > 100, got %d", len(expected), len(rows))
			}
			for _, row := range rows {
				total, err := numField(row, "Total")
				if err != nil {
					return err
				}
				if total <= 100 {
					return fmt.Errorf("group with Total %v should have been filtered out (Total gt 100)", total)
				}
				cat := productString(row, "CategoryID")
				if want, ok := expected[cat]; !ok || !aggApproxEqual(total, want) {
					return fmt.Errorf("group %q Total %v does not match expected %v", cat, total, want)
				}
			}
			return nil
		},
	)

	// Test 9: Min and max aggregation together
	suite.AddTest(
		"test_min_max_aggregate",
		"min and max aggregation return the extremes",
		func(ctx *framework.TestContext) error {
			stats, err := computePriceStats(ctx, nil)
			if err != nil {
				return err
			}
			row, err := applyAggregateRow(ctx, "aggregate(Price with min as MinPrice,Price with max as MaxPrice)")
			if err != nil {
				return err
			}
			if err := assertNumField(row, "MinPrice", stats.min); err != nil {
				return err
			}
			return assertNumField(row, "MaxPrice", stats.max)
		},
	)

	// Test 10: $apply with $top
	suite.AddTest(
		"test_apply_with_top",
		"$apply works with $top",
		func(ctx *framework.TestContext) error {
			applyFilter := url.QueryEscape("groupby((CategoryID),aggregate($count as Count))")
			resp, err := ctx.GET("/Products?$apply=" + applyFilter + "&$top=2")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 11: $apply with $orderby
	suite.AddTest(
		"test_apply_with_orderby",
		"$apply works with $orderby",
		func(ctx *framework.TestContext) error {
			applyFilter := url.QueryEscape("groupby((CategoryID),aggregate(Price with sum as Total))")
			orderby := url.QueryEscape("Total desc")
			resp, err := ctx.GET("/Products?$apply=" + applyFilter + "&$orderby=" + orderby)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 12: Average aggregation
	suite.AddTest(
		"test_average_aggregation",
		"average aggregation returns the mean Price",
		func(ctx *framework.TestContext) error {
			stats, err := computePriceStats(ctx, nil)
			if err != nil {
				return err
			}
			row, err := applyAggregateRow(ctx, "aggregate(Price with average as AvgPrice)")
			if err != nil {
				return err
			}
			return assertNumField(row, "AvgPrice", stats.avg)
		},
	)

	// Test 13: Invalid aggregation method
	suite.AddTest(
		"test_invalid_aggregation_method",
		"Invalid aggregation method returns error",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("aggregate(Price with invalid as Result)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
