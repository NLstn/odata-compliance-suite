package v4_0

import (
	"fmt"
	"math"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// QuerySkiptoken tests server-generated continuation links. Clients must not
// invent $skiptoken values, and services need not use $skiptoken for paging.
func QuerySkiptoken() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.7 $skiptoken",
		"Tests server-driven paging through opaque @odata.nextLink URLs.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ServerDrivenPaging",
	)

	suite.AddTest(
		"test_nextlink_continuation",
		"A server-provided nextLink covers the complete collection without duplicates",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true", framework.Header{Key: "Prefer", Value: "odata.maxpagesize=2"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			var envelope struct {
				Count    float64                  `json:"@odata.count"`
				Value    []map[string]interface{} `json:"value"`
				NextLink string                   `json:"@odata.nextLink"`
			}
			if err := ctx.GetJSON(resp, &envelope); err != nil {
				return err
			}
			if envelope.Count < 0 || math.Trunc(envelope.Count) != envelope.Count {
				return fmt.Errorf("invalid @odata.count: %v", envelope.Count)
			}
			if envelope.Count > float64(len(envelope.Value)) && envelope.NextLink == "" {
				return fmt.Errorf("partial collection has no @odata.nextLink: count=%v, page=%d", envelope.Count, len(envelope.Value))
			}
			all, err := collectEntityCollection(ctx, "/Products?$count=true", framework.Header{Key: "Prefer", Value: "odata.maxpagesize=2"})
			if err != nil {
				return err
			}
			if len(all) != int(envelope.Count) {
				return fmt.Errorf("continuation returned %d unique entities; @odata.count=%v", len(all), envelope.Count)
			}
			return nil
		},
	)

	suite.AddTest(
		"test_top_bounds_total",
		"$top bounds the total across all server-driven pages",
		func(ctx *framework.TestContext) error {
			items, err := collectEntityCollection(ctx, "/Products?$top=3", framework.Header{Key: "Prefer", Value: "odata.maxpagesize=2"})
			if err != nil {
				return err
			}
			if len(items) > 3 {
				return fmt.Errorf("$top=3 returned %d entities across pages", len(items))
			}
			return nil
		},
	)
	return suite
}
