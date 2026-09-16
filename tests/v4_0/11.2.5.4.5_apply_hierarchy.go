package v4_0

import (
	"fmt"
	"github.com/nlstn/odata-compliance-suite/framework"
	"net/url"
	"reflect"
	"sort"
)

func QueryApplyHierarchy() *framework.TestSuite {
	s := framework.NewTestSuite("11.2.5.4.5 $apply hierarchy", "RecursiveHierarchy traversal and input-set restriction.", "https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/cs04/odata-data-aggregation-ext-v4.0-cs04.html#Transformationsancestorsanddescendants")
	for _, tc := range []struct {
		name, expr string
		ids        []int
		ordered    bool
	}{
		{"ancestors", "ancestors($root/HierarchyNodes,Tree,ID,filter(ID eq 4))", []int{1, 2}, false},
		{"ancestor_distance", "ancestors($root/HierarchyNodes,Tree,ID,filter(ID eq 4),1)", []int{2}, false},
		{"keep_start", "ancestors($root/HierarchyNodes,Tree,ID,filter(ID eq 4),keep start)", []int{1, 2, 4}, false},
		{"descendants", "descendants($root/HierarchyNodes,Tree,ID,filter(ID eq 1))", []int{2, 3, 4}, false},
		{"descendant_distance", "descendants($root/HierarchyNodes,Tree,ID,filter(ID eq 1),1,keep start)", []int{1, 2, 3}, false},
		{"overlapping_starts", "ancestors($root/HierarchyNodes,Tree,ID,filter(ID eq 4 or ID eq 3),keep start)", []int{1, 2, 3, 4}, false},
		{"input_subset", "filter(ID ne 2)/ancestors($root/HierarchyNodes,Tree,ID,filter(ID eq 4),keep start)", []int{1, 4}, false},
		{"no_matches", "descendants($root/HierarchyNodes,Tree,ID,filter(ID eq 99))", []int{}, false},
		{"preorder", "traverse($root/HierarchyNodes,Tree,ID,preorder,ID asc)", []int{1, 2, 4, 3, 5}, true},
		{"postorder", "traverse($root/HierarchyNodes,Tree,ID,postorder,ID asc)", []int{4, 2, 3, 1, 5}, true},
	} {
		s.AddTest("hierarchy_"+tc.name, tc.name, func(ctx *framework.TestContext) error {
			r, e := ctx.GET("/HierarchyNodes?$apply=" + url.QueryEscape(tc.expr))
			if e != nil {
				return e
			}
			if e = ctx.AssertStatusCode(r, 200); e != nil {
				return e
			}
			rows, e := ctx.ParseEntityCollection(r)
			if e != nil {
				return e
			}
			ids := make([]int, 0, len(rows))
			for _, row := range rows {
				id, ok := row["ID"].(float64)
				if !ok {
					return fmt.Errorf("missing numeric ID: %v", row)
				}
				ids = append(ids, int(id))
				if row["Name"] != fmt.Sprintf("Node %d", int(id)) {
					return fmt.Errorf("incorrect Name: %v", row)
				}
			}
			if !tc.ordered {
				sort.Ints(ids)
			}
			if !reflect.DeepEqual(ids, tc.ids) {
				return fmt.Errorf("IDs %v, want %v", ids, tc.ids)
			}
			return nil
		})
	}
	for _, expr := range []string{"ancestors($root/HierarchyNodes,Missing,ID,filter(ID eq 4))", "descendants($root/HierarchyNodes,Tree,ID,filter(ID eq 1),0)", "ancestors($root/HierarchyNodes,Tree,ID,aggregate($count as N))", "traverse($root/HierarchyNodes,Tree,ID,sideways)"} {
		s.AddTest("invalid_"+expr, expr, func(ctx *framework.TestContext) error {
			r, e := ctx.GET("/HierarchyNodes?$apply=" + url.QueryEscape(expr))
			if e != nil {
				return e
			}
			return ctx.AssertStatusCode(r, 400)
		})
	}
	return s
}
