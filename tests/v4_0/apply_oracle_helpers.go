package v4_0

import (
	"fmt"
	"math"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// applyRows runs /Products?$apply=<expr> and returns the result rows.
func applyRows(ctx *framework.TestContext, expr string) ([]map[string]interface{}, error) {
	resp, err := ctx.GET("/Products?$apply=" + url.QueryEscape(expr))
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, err
	}
	return ctx.ParseEntityCollection(resp)
}

// applyAggregateRow runs an aggregate transformation that yields a single row.
func applyAggregateRow(ctx *framework.TestContext, expr string) (map[string]interface{}, error) {
	rows, err := applyRows(ctx, expr)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("$apply=%s: expected exactly 1 aggregate row, got %d", expr, len(rows))
	}
	return rows[0], nil
}

// numField reads a numeric aggregate field by name.
func numField(row map[string]interface{}, name string) (float64, error) {
	raw, ok := row[name]
	if !ok {
		return 0, fmt.Errorf("aggregate row missing field %q", name)
	}
	f, ok := raw.(float64)
	if !ok {
		return 0, fmt.Errorf("aggregate field %q is not numeric (%T)", name, raw)
	}
	return f, nil
}

// assertNumField checks that a numeric aggregate field equals want within tolerance.
func assertNumField(row map[string]interface{}, name string, want float64) error {
	got, err := numField(row, name)
	if err != nil {
		return err
	}
	if !aggApproxEqual(got, want) {
		return fmt.Errorf("aggregate %q = %v, expected %v", name, got, want)
	}
	return nil
}

// aggApproxEqual compares floats with a small absolute+relative tolerance to absorb
// double-precision and rounding differences.
func aggApproxEqual(a, b float64) bool {
	return math.Abs(a-b) <= 0.01+1e-9*math.Abs(b)
}

// priceStats holds aggregate statistics over the Products' Price column.
type priceStats struct {
	sum, avg, min, max float64
	count              int
}

// computePriceStats computes Price aggregates in Go over the products matching keep.
// keep == nil includes every product.
func computePriceStats(ctx *framework.TestContext, keep func(map[string]interface{}) bool) (priceStats, error) {
	all, err := fetchAllProducts(ctx)
	if err != nil {
		return priceStats{}, err
	}
	stats := priceStats{}
	for _, p := range all {
		if keep != nil && !keep(p) {
			continue
		}
		price, ok := productFloat(p, "Price")
		if !ok {
			continue
		}
		if stats.count == 0 || price < stats.min {
			stats.min = price
		}
		if stats.count == 0 || price > stats.max {
			stats.max = price
		}
		stats.sum += price
		stats.count++
	}
	if stats.count > 0 {
		stats.avg = stats.sum / float64(stats.count)
	}
	return stats, nil
}

// computeCategoryGroups computes per-CategoryID Price sum and row count in Go.
func computeCategoryGroups(ctx *framework.TestContext) (sums map[string]float64, counts map[string]int, err error) {
	all, err := fetchAllProducts(ctx)
	if err != nil {
		return nil, nil, err
	}
	sums = map[string]float64{}
	counts = map[string]int{}
	for _, p := range all {
		cat := productString(p, "CategoryID")
		counts[cat]++
		if price, ok := productFloat(p, "Price"); ok {
			sums[cat] += price
		}
	}
	return sums, counts, nil
}
