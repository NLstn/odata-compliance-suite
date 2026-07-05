package v4_0

import (
	"encoding/json"
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
		"groupby with multiple grouping properties returns distinct (CategoryID,Status) pairs",
		func(ctx *framework.TestContext) error {
			all, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}
			// Compute distinct (CategoryID,Status) pairs in Go.
			type pair struct{ cat, status string }
			distinctPairs := map[pair]struct{}{}
			for _, p := range all {
				cat := productString(p, "CategoryID")
				status := fmt.Sprintf("%v", p["Status"])
				distinctPairs[pair{cat, status}] = struct{}{}
			}
			rows, err := applyRows(ctx, "groupby((CategoryID,Status))")
			if err != nil {
				return err
			}
			if len(rows) != len(distinctPairs) {
				return fmt.Errorf("groupby((CategoryID,Status)): expected %d distinct pairs, got %d rows",
					len(distinctPairs), len(rows))
			}
			for i, row := range rows {
				if _, ok := row["CategoryID"]; !ok {
					return fmt.Errorf("row %d missing CategoryID", i)
				}
				if _, ok := row["Status"]; !ok {
					return fmt.Errorf("row %d missing Status", i)
				}
			}
			return nil
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
		"filter() before groupby/aggregate: per-group count matches oracle",
		func(ctx *framework.TestContext) error {
			all, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}
			// Oracle: count products with Price > 50 per CategoryID.
			expectedCounts := map[string]int{}
			for _, p := range all {
				price, ok := productFloat(p, "Price")
				if !ok || price <= 50 {
					continue
				}
				cat := productString(p, "CategoryID")
				expectedCounts[cat]++
			}
			rows, err := applyRows(ctx, "filter(Price gt 50)/groupby((CategoryID),aggregate($count as Count))")
			if err != nil {
				return err
			}
			if len(rows) != len(expectedCounts) {
				return fmt.Errorf("filter+groupby: expected %d category groups, got %d", len(expectedCounts), len(rows))
			}
			for i, row := range rows {
				cat := productString(row, "CategoryID")
				want, ok := expectedCounts[cat]
				if !ok {
					return fmt.Errorf("row %d: unexpected CategoryID %q in result", i, cat)
				}
				got, err := numField(row, "Count")
				if err != nil {
					return fmt.Errorf("row %d: %w", i, err)
				}
				if !aggApproxEqual(got, float64(want)) {
					return fmt.Errorf("row %d CategoryID=%q: Count=%v, expected %d", i, cat, got, want)
				}
			}
			return nil
		},
	)

	// Test 6: Multiple transformations in sequence
	suite.AddTest(
		"test_transformation_pipeline",
		"filter/groupby/filter pipeline: only categories with >1 product (Price>10) returned",
		func(ctx *framework.TestContext) error {
			all, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}
			// Oracle: count products with Price > 10 per CategoryID;
			// keep only categories with count > 1.
			groupCounts := map[string]int{}
			for _, p := range all {
				price, ok := productFloat(p, "Price")
				if !ok || price <= 10 {
					continue
				}
				groupCounts[productString(p, "CategoryID")]++
			}
			expectedCats := map[string]struct{}{}
			for cat, cnt := range groupCounts {
				if cnt > 1 {
					expectedCats[cat] = struct{}{}
				}
			}
			rows, err := applyRows(ctx, "filter(Price gt 10)/groupby((CategoryID))/filter($count gt 1)")
			if err != nil {
				return err
			}
			if len(rows) != len(expectedCats) {
				return fmt.Errorf("pipeline: expected %d categories with >1 product, got %d rows",
					len(expectedCats), len(rows))
			}
			for i, row := range rows {
				cat := productString(row, "CategoryID")
				if _, ok := expectedCats[cat]; !ok {
					return fmt.Errorf("row %d: CategoryID=%q unexpected in filtered result", i, cat)
				}
			}
			return nil
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
		"$apply with $top: returns at most 2 rows",
		func(ctx *framework.TestContext) error {
			rows, err := applyRows(ctx, "groupby((CategoryID),aggregate($count as Count))")
			if err != nil {
				return err
			}
			totalGroups := len(rows)

			applyFilter := url.QueryEscape("groupby((CategoryID),aggregate($count as Count))")
			resp, err := ctx.GET("/Products?$apply=" + applyFilter + "&$top=2")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			topRows, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if len(topRows) > 2 {
				return fmt.Errorf("$top=2 returned %d rows (more than 2)", len(topRows))
			}
			if totalGroups >= 2 && len(topRows) < 2 {
				return fmt.Errorf("$top=2 with %d total groups should return 2 rows, got %d", totalGroups, len(topRows))
			}
			return nil
		},
	)

	// Test 11: $apply with $orderby
	suite.AddTest(
		"test_apply_with_orderby",
		"$apply with $orderby: groups are sorted descending by Total",
		func(ctx *framework.TestContext) error {
			applyFilter := url.QueryEscape("groupby((CategoryID),aggregate(Price with sum as Total))")
			orderby := url.QueryEscape("Total desc")
			resp, err := ctx.GET("/Products?$apply=" + applyFilter + "&$orderby=" + orderby)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			rows, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			for i := 1; i < len(rows); i++ {
				prev, err := numField(rows[i-1], "Total")
				if err != nil {
					return fmt.Errorf("row %d Total: %w", i-1, err)
				}
				curr, err := numField(rows[i], "Total")
				if err != nil {
					return fmt.Errorf("row %d Total: %w", i, err)
				}
				if prev < curr {
					return fmt.Errorf("$orderby Total desc violated at index %d: %v < %v", i, prev, curr)
				}
			}
			return nil
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

	// Test: aggregate over a navigation property path (OData Data Aggregation §6.2.1)
	// 'Descriptions/aggregate($count as DescCount)' counts related Descriptions per product
	// in a groupby, or in a top-level apply produces one row per product.
	suite.AddTest(
		"test_aggregate_over_navigation_path",
		"aggregate() over a navigation property path counts related entities (§6.2.1)",
		func(ctx *framework.TestContext) error {
			// Baseline: total number of description rows.
			baselineResp, err := ctx.GET("/ProductDescriptions?$count=true&$top=0")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(baselineResp, 200); err != nil {
				return err
			}
			var baseline map[string]interface{}
			if err := json.Unmarshal(baselineResp.Body, &baseline); err != nil {
				return fmt.Errorf("failed to parse /ProductDescriptions count: %w", err)
			}
			totalDescs, ok := baseline["@odata.count"].(float64)
			if !ok {
				return framework.NewError("baseline /ProductDescriptions missing @odata.count")
			}

			// aggregate(Descriptions/$count as DescCount) should produce the total
			// description count across all products in a single aggregate row.
			resp, err := ctx.GET("/Products?$apply=" + url.QueryEscape("aggregate(Descriptions/$count as DescCount)"))
			if err != nil {
				return err
			}
			if resp.StatusCode == 400 || resp.StatusCode == 501 {
				return ctx.Skip("aggregate over navigation path not supported (400/501)")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var body struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if len(body.Value) == 0 {
				return framework.NewError("aggregate over navigation path returned empty value array")
			}
			rawDesc, ok := firstPresent(body.Value[0], "DescCount", "desccount")
			if !ok {
				return framework.NewError("aggregate(Descriptions/$count as DescCount) response missing DescCount")
			}
			descCount, ok := rawDesc.(float64)
			if !ok {
				return fmt.Errorf("DescCount is not numeric: %T", rawDesc)
			}
			if int(descCount) != int(totalDescs) {
				return fmt.Errorf("navigation aggregate DescCount=%d expected %d (total description rows)", int(descCount), int(totalDescs))
			}
			return nil
		},
	)

	// Test: $apply=groupby()/aggregate() combined with an external $filter system query option.
	// Per OData Data Aggregation §6.4, system query options such as $filter are applied AFTER
	// $apply on the transformed result set. This test distinguishes internal pipeline filters
	// (filter() inside $apply) from the external $filter option — the server must allow $filter
	// to reference properties introduced by the aggregation (e.g. Total from aggregate()).
	suite.AddTest(
		"test_external_filter_on_apply_result",
		"$filter=Total gt N applied outside $apply filters the aggregated result rows (§6.4)",
		func(ctx *framework.TestContext) error {
			sums, _, err := computeCategoryGroups(ctx)
			if err != nil {
				return err
			}

			// Oracle: which categories have a per-category Price sum > 100?
			expectedGroups := map[string]float64{}
			for cat, sum := range sums {
				if sum > 100 {
					expectedGroups[cat] = sum
				}
			}

			// $apply groups+aggregates; external $filter filters the aggregated rows.
			applyExpr := url.QueryEscape("groupby((CategoryID),aggregate(Price with sum as Total))")
			filterExpr := url.QueryEscape("Total gt 100")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr + "&$filter=" + filterExpr)
			if err != nil {
				return err
			}
			if resp.StatusCode == 400 || resp.StatusCode == 501 {
				return ctx.Skip("$filter on aggregated property not supported after $apply (400/501)")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var body struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("external $filter response not valid JSON: %w", err)
			}

			if len(body.Value) != len(expectedGroups) {
				return fmt.Errorf("$apply+$filter: expected %d groups with Total > 100, got %d",
					len(expectedGroups), len(body.Value))
			}
			for i, row := range body.Value {
				total, err := numField(row, "Total")
				if err != nil {
					return fmt.Errorf("row %d: %w", i, err)
				}
				if total <= 100 {
					return fmt.Errorf("row %d: $filter=Total gt 100 not applied — Total=%v leaked through", i, total)
				}
				cat := productString(row, "CategoryID")
				want, ok := expectedGroups[cat]
				if !ok {
					return fmt.Errorf("row %d: CategoryID=%q not in oracle (unexpected group)", i, cat)
				}
				if !aggApproxEqual(total, want) {
					return fmt.Errorf("row %d: CategoryID=%q Total=%v expected %v", i, cat, total, want)
				}
			}
			return nil
		},
	)

	return suite
}
