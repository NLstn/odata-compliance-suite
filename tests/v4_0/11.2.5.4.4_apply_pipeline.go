package v4_0

import (
	"encoding/json"
	"fmt"
	"github.com/nlstn/odata-compliance-suite/framework"
	"net/url"
	"reflect"
	"strings"
)

// QueryApplyPipeline tests Aggregation sections 3.1, 3.2.2 and 3.4.2.
func QueryApplyPipeline() *framework.TestSuite {
	s := framework.NewTestSuite("11.2.5.4.4 $apply pipeline", "Exact concatenation, alias scope and sequential evaluation.", "https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/cs04/odata-data-aggregation-ext-v4.0-cs04.html")
	for _, tc := range []struct{ name, expr, want string }{
		{"duplicates", "filter(Name eq 'Coffee Mug')/concat(aggregate($count as N),aggregate($count as N))", `[{"N":1},{"N":1}]`},
		{"heterogeneous", "filter(Name eq 'Coffee Mug')/concat(aggregate($count as N),aggregate(Price with sum as Total))", `[{"N":1},{"Total":15.5}]`},
		{"nested_concat", "filter(Name eq 'Coffee Mug')/concat(concat(aggregate($count as N),aggregate($count as N)),aggregate($count as N))", `[{"N":1},{"N":1},{"N":1}]`},
		{"successive_concat", "filter(Name eq 'Coffee Mug')/concat(identity,identity)/concat(aggregate($count as N),aggregate($count as N))", `[{"N":2},{"N":2}]`},
		{"aggregate_compute", "filter(Name eq 'Coffee Mug')/aggregate(Price with sum as Total)/compute(Total mul 2 as DoubleTotal)", `[{"Total":15.5,"DoubleTotal":31}]`},
		{"string_compute", "filter(Name eq 'Coffee Mug')/compute(toupper(Name) as UpperName)/filter(UpperName eq 'COFFEE MUG')/aggregate($count as N)", `[{"N":1}]`},
		{"compute_aggregate", "filter(Name eq 'Coffee Mug')/compute(Price mul 2 as Twice)/aggregate(Twice with sum as Total)", `[{"Total":31}]`},
		{"successive_compute", "filter(Name eq 'Coffee Mug')/compute(Price mul 2 as Twice)/compute(Twice add 1 as Next)/aggregate(Next with sum as Total)", `[{"Total":32}]`},
		{"compute_filter", "filter(Name eq 'Coffee Mug')/compute(Price mul 2 as Twice)/filter(Twice gt 30)/aggregate($count as N)", `[{"N":1}]`},
		{"top_before_filter", "orderby(Price asc)/top(1)/filter(Price gt 100)/aggregate($count as N)", `[{"N":0}]`},
		{"top_before_aggregate", "orderby(Price asc)/top(1)/aggregate(Price with sum as Total)", `[{"Total":15.5}]`},
		{"branch_alias_scope", "filter(Name eq 'Coffee Mug')/compute(Price mul 2 as Twice)/concat(aggregate(Twice with sum as Total),aggregate(Twice with max as Total))", `[{"Total":31},{"Total":31}]`},
	} {
		s.AddTest("test_apply_"+tc.name, tc.name, func(ctx *framework.TestContext) error {
			rows, err := applyRows(ctx, tc.expr)
			if err != nil {
				return err
			}
			for _, row := range rows {
				for key := range row {
					if strings.Contains(key, "@") {
						delete(row, key)
					}
				}
			}
			var want []map[string]interface{}
			if err = json.Unmarshal([]byte(tc.want), &want); err != nil {
				return err
			}
			if !reflect.DeepEqual(rows, want) {
				return fmt.Errorf("%s: got %#v, want %#v", tc.expr, rows, want)
			}
			return nil
		})
	}
	for _, expr := range []string{"concat(identity)", "concat()", "compute(Price as Name)", "compute(Price as X,Price as X)", "filter(Twice gt 1)/compute(Price mul 2 as Twice)"} {
		s.AddTest("invalid_"+expr, expr, func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$apply=" + url.QueryEscape(expr))
			if err != nil {
				return err
			}
			if err = ctx.AssertStatusCode(resp, 400); err != nil {
				return err
			}
			var body map[string]interface{}
			if err = json.Unmarshal(resp.Body, &body); err != nil {
				return err
			}
			e, ok := body["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("missing OData error object")
			}
			if _, ok = e["code"].(string); !ok {
				return fmt.Errorf("missing string error code")
			}
			if _, ok = e["message"].(string); !ok {
				return fmt.Errorf("missing string error message")
			}
			return nil
		})
	}
	return s
}
