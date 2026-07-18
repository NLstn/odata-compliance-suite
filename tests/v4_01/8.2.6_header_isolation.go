package v4_01

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderIsolation validates both the 4.01 Isolation spelling and the legacy
// OData-Isolation spelling that remains necessary for 4.0 clients.
func HeaderIsolation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.6 Header Isolation",
		"Tests snapshot-isolation header compatibility for OData 4.01 and 4.0 clients.",
		"https://docs.oasis-open.org/odata/odata/v4.01/os/part1-protocol/odata-v4.01-os-part1-protocol.html#sec_HeaderIsolationODataIsolation",
	)

	for _, tc := range []struct {
		name       string
		headerName string
		maxVersion string
	}{
		{"test_isolation_snapshot_401", "Isolation", "4.01"},
		{"test_odata_isolation_snapshot_40_compatibility", "OData-Isolation", "4.0"},
	} {
		tc := tc
		suite.AddTest(
			tc.name,
			tc.headerName+":snapshot is honored or rejected with 412 Precondition Failed; if accepted, a paginated read is exercised against a concurrent delete to observe whether the isolation is actually snapshot-consistent",
			func(ctx *framework.TestContext) error {
				isolationHeaders := []framework.Header{
					{Key: tc.headerName, Value: "snapshot"},
					{Key: "OData-MaxVersion", Value: tc.maxVersion},
				}

				resp, err := ctx.GET("/Products?$top=3", isolationHeaders...)
				if err != nil {
					return err
				}
				if resp.StatusCode == 412 {
					ctx.Log(fmt.Sprintf("%s:snapshot rejected with 412 (isolation unsupported) — conformant", tc.headerName))
					return nil
				}
				if err := ctx.AssertStatusCode(resp, 200); err != nil {
					return fmt.Errorf("%s:snapshot status = %w, want 200 if supported or 412 if unsupported", tc.headerName, err)
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
					seenIDs[entityID(p)] = true
				}

				// Find a product not yet fetched, to delete between page 1 and
				// page 2 — if isolation is truly snapshot-consistent, page 2
				// should still reflect the pre-delete state (the deleted ID
				// may still appear); if not, it must be absent.
				allResp, err := ctx.GET("/Products?$top=1000")
				if err != nil {
					return err
				}
				if err := ctx.AssertStatusCode(allResp, 200); err != nil {
					return err
				}
				all, err := decodeCollection(allResp)
				if err != nil {
					return err
				}
				var deleteTarget string
				for _, p := range all {
					id := entityID(p)
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
					if entityID(p) == deleteTarget {
						foundDeletedOnPage2 = true
						break
					}
				}
				if foundDeletedOnPage2 {
					ctx.Log(fmt.Sprintf("%s:snapshot held a consistent view across pages: a product deleted between page fetches still appeared on the next page", tc.headerName))
				} else {
					ctx.Log(fmt.Sprintf("%s:snapshot did not prevent a concurrent delete from affecting a later page — isolation may be accepted but not behaviorally enforced across paginated reads", tc.headerName))
				}
				return nil
			},
		)
	}

	return suite
}

// entityID returns the "ID" field of a decoded entity as a string.
func entityID(entity map[string]interface{}) string {
	return fmt.Sprintf("%v", entity["ID"])
}
