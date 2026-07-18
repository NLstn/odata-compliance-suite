package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderIsolation validates the OData 4.0 spelling and required fallback
// behavior of the snapshot-isolation request header.
func HeaderIsolation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.6 Header OData-Isolation",
		"Tests OData-Isolation:snapshot acceptance or the required 412 response when snapshot isolation is unsupported.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderODataIsolation",
	)

	suite.AddTest(
		"test_odata_isolation_snapshot",
		"OData-Isolation:snapshot is honored or rejected with 412 Precondition Failed; if accepted, a paginated read is exercised against a concurrent delete to observe whether the isolation is actually snapshot-consistent",
		func(ctx *framework.TestContext) error {
			isolationHeaders := []framework.Header{
				{Key: "OData-Isolation", Value: "snapshot"},
				{Key: "OData-MaxVersion", Value: "4.0"},
			}

			resp, err := ctx.GET("/Products?$top=3", isolationHeaders...)
			if err != nil {
				return err
			}
			if resp.StatusCode == 412 {
				ctx.Log("OData-Isolation:snapshot rejected with 412 (isolation unsupported) — conformant")
				return nil
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return fmt.Errorf("OData-Isolation:snapshot status = %w, want 200 if supported or 412 if unsupported", err)
			}

			var page1 struct {
				Value    []map[string]interface{} `json:"value"`
				NextLink string                   `json:"@odata.nextLink"`
			}
			if err := json.Unmarshal(resp.Body, &page1); err != nil {
				return fmt.Errorf("failed to parse first page: %w", err)
			}
			if page1.NextLink == "" {
				return ctx.Skip("fewer products than $top=3 available; cannot exercise multi-page isolation behavior")
			}

			seenIDs := map[string]bool{}
			for _, p := range page1.Value {
				seenIDs[productID(p)] = true
			}

			// Find a product not yet fetched, to delete between page 1 and
			// page 2 — if isolation is truly snapshot-consistent, page 2
			// should still reflect the pre-delete state (the deleted ID may
			// still appear); if not, it must be absent.
			all, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}
			var deleteTarget string
			for _, p := range all {
				id := productID(p)
				if !seenIDs[id] {
					deleteTarget = id
					break
				}
			}
			if deleteTarget == "" {
				return ctx.Skip("no unfetched product available to delete between pages")
			}

			delResp, err := ctx.DELETE(fmt.Sprintf("/Products(%s)", deleteTarget))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(delResp, 204); err != nil {
				return fmt.Errorf("setup: failed to delete product between pages: %w", err)
			}

			nextPath := strings.TrimPrefix(page1.NextLink, ctx.ServerURL())
			page2Resp, err := ctx.GET(nextPath, isolationHeaders...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(page2Resp, 200); err != nil {
				return fmt.Errorf("failed to follow nextLink after a concurrent delete: %w", err)
			}
			var page2 struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(page2Resp.Body, &page2); err != nil {
				return fmt.Errorf("failed to parse second page: %w", err)
			}

			foundDeletedOnPage2 := false
			for _, p := range page2.Value {
				if productID(p) == deleteTarget {
					foundDeletedOnPage2 = true
					break
				}
			}
			if foundDeletedOnPage2 {
				ctx.Log("OData-Isolation:snapshot held a consistent view across pages: a product deleted between page fetches still appeared on the next page")
			} else {
				ctx.Log("OData-Isolation:snapshot did not prevent a concurrent delete from affecting a later page — isolation may be accepted but not behaviorally enforced across paginated reads")
			}
			return nil
		},
	)

	return suite
}
