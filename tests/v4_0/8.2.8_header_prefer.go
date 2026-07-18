package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderPrefer creates the 8.2.8 Prefer Header test suite
func HeaderPrefer() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.8 Prefer Header",
		"Tests Prefer header handling for client preferences like return=minimal and return=representation.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderPrefer",
	)

	suite.AddTest(
		"test_prefer_return_minimal",
		"Prefer: return=minimal — 204 body must be empty; Preference-Applied must include return=minimal when honored",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/Products", map[string]interface{}{
				"Name":   "Prefer Test",
				"Price":  99.99,
				"Status": 1, // ProductStatusInStock
			}, framework.Header{
				Key:   "Prefer",
				Value: "return=minimal",
			})
			if err != nil {
				return err
			}
			if resp.StatusCode != 201 && resp.StatusCode != 204 && resp.StatusCode != 200 {
				return fmt.Errorf("expected successful creation (200/201/204), got %d", resp.StatusCode)
			}

			// A service may signal "honored" either via 204 No Content or via
			// 201 Created with an empty body plus Preference-Applied — both
			// are used by real implementations, so the header (or 204) is the
			// actual signal, not the status code alone.
			prefApplied := resp.Headers.Get("Preference-Applied")
			honored := resp.StatusCode == 204 || prefApplied == "return=minimal"

			if honored {
				if len(resp.Body) > 0 {
					return framework.NewError("return=minimal honored but response body is not empty")
				}
				if prefApplied != "" && prefApplied != "return=minimal" {
					return fmt.Errorf("Preference-Applied=%q; expected 'return=minimal' when preference was honored", prefApplied)
				}
				return nil
			}

			// Not honored: the server chose to return the full representation
			// instead, matching return=representation behavior. A server that
			// simply ignores the preference must not be indistinguishable from
			// one that correctly implements it — verify a real representation
			// actually came back, not just "some" 2xx status.
			if len(resp.Body) == 0 {
				return fmt.Errorf("return=minimal not honored (status %d, Preference-Applied=%q) but response body is empty; expected the full entity representation", resp.StatusCode, prefApplied)
			}
			if err := ctx.AssertJSONField(resp, "Name"); err != nil {
				return fmt.Errorf("return=minimal not honored: response body should contain the full entity representation: %w", err)
			}
			return nil
		},
	)

	suite.AddTest(
		"test_prefer_return_representation",
		"Prefer: return=representation — response body must contain entity; Preference-Applied must include return=representation when honored",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/Products", map[string]interface{}{
				"Name":   "Prefer Test 2",
				"Price":  99.99,
				"Status": 1, // ProductStatusInStock
			}, framework.Header{
				Key:   "Prefer",
				Value: "return=representation",
			})
			if err != nil {
				return err
			}
			if resp.StatusCode != 201 && resp.StatusCode != 200 {
				return fmt.Errorf("expected successful creation (200/201), got %d", resp.StatusCode)
			}

			if len(resp.Body) == 0 {
				return framework.NewError("expected entity representation in response body")
			}
			if err := ctx.AssertJSONField(resp, "Name"); err != nil {
				return fmt.Errorf("response body should contain created entity: %v", err)
			}

			// If Preference-Applied is present it must acknowledge the honored preference.
			prefApplied := resp.Headers.Get("Preference-Applied")
			if prefApplied != "" && prefApplied != "return=representation" {
				return fmt.Errorf("Preference-Applied=%q; expected 'return=representation' when preference was honored", prefApplied)
			}
			return nil
		},
	)

	// Test 3: Prefer: odata.maxpagesize=N — server SHOULD limit page size per §8.2.8.3.
	// If honored: response has ≤N items AND @odata.nextLink (since seed has 7 products > 2).
	// If not honored: server returns all items in one page — skip gracefully.
	suite.AddTest(
		"test_prefer_maxpagesize",
		"Prefer: odata.maxpagesize=2 — if honored, page must have ≤2 items with @odata.nextLink",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{
				Key:   "Prefer",
				Value: "odata.maxpagesize=2",
			})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			var body map[string]interface{}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("maxpagesize response is not valid JSON: %w", err)
			}
			items, ok := body["value"].([]interface{})
			if !ok {
				return fmt.Errorf("maxpagesize response missing 'value' array")
			}
			_, hasNextLink := body["@odata.nextLink"]

			if len(items) <= 2 {
				// Preference was honored — verify nextLink is present (seed has 7 products).
				if !hasNextLink {
					if len(items) < 7 {
						return fmt.Errorf("odata.maxpagesize=2 honored (%d items returned) but @odata.nextLink is missing (expected more pages)", len(items))
					}
					// If all 7 (or fewer) items fit, nextLink absence is fine.
				}
				ctx.Log(fmt.Sprintf("odata.maxpagesize=2 honored: %d items returned", len(items)))
				// Verify Preference-Applied header if present.
				pa := resp.Headers.Get("Preference-Applied")
				if pa != "" && pa != "odata.maxpagesize=2" {
					return fmt.Errorf("Preference-Applied=%q does not reflect honored maxpagesize preference", pa)
				}
			} else {
				// Server returned more than 2 items — preference ignored.
				return ctx.Skip("server does not honor Prefer: odata.maxpagesize (SHOULD per §8.2.8.3)")
			}
			return nil
		},
	)

	return suite
}
