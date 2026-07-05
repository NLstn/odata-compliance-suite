package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// AsynchronousProcessing creates the 13.1 Asynchronous Request Processing test suite
func AsynchronousProcessing() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"13.1 Asynchronous Processing",
		"Tests asynchronous request processing features including the Prefer: respond-async header, status monitor URLs, and proper async response patterns.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_AsynchronousRequests",
	)

	// Test 1: Service accepts Prefer: respond-async header
	suite.AddTest(
		"test_prefer_async_header_accepted",
		"Service accepts Prefer: respond-async header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			// Should return 200 (synchronous) or 202 (asynchronous)
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return ctx.AssertStatusCode(resp, 202)
			}

			return nil
		},
	)

	// Test 2: Service can respond synchronously even with Prefer: respond-async
	suite.AddTest(
		"test_synchronous_response_allowed",
		"Service can respond synchronously with async preference",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			// Both 200 and 202 are valid
			if resp.StatusCode == 200 || resp.StatusCode == 202 {
				return nil
			}

			return framework.NewError("Service should respond with 200 or 202")
		},
	)

	// Test 3: Async response returns 202 Accepted
	suite.AddTest(
		"test_async_returns_202",
		"Async 202 response includes Location header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			if resp.StatusCode == 202 {
				// Per OData Protocol §13.1, a 202 response MUST include a Location header.
				location := resp.Headers.Get("Location")
				if location == "" {
					return framework.NewError("202 Accepted response missing required Location header (OData Protocol §13.1)")
				}
				return nil
			} else if resp.StatusCode == 200 {
				return framework.NewError("Service does not support asynchronous processing")
			}

			return fmt.Errorf("expected 200 or 202, got %d", resp.StatusCode)
		},
	)

	// Test 4: Async response includes Location header
	suite.AddTest(
		"test_async_location_header",
		"Async 202 response includes Location header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			if resp.StatusCode == 202 {
				location := resp.Headers.Get("Location")
				if location == "" {
					return framework.NewError("202 Accepted response must include Location header")
				}
				return nil
			} else if resp.StatusCode == 200 {
				return framework.NewError("Service does not support asynchronous processing")
			}

			return framework.NewError("Unexpected status code")
		},
	)

	// Test 5: POST with async preference
	suite.AddTest(
		"test_async_post_request",
		"POST request with async preference returns 201 (sync) or 202 (async)",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Async Test", 99.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Prefer", Value: "respond-async"},
				framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// 201: processed synchronously; 202: async accepted.
			// 400 is not acceptable — it means the server rejected a valid payload.
			if resp.StatusCode == 201 || resp.StatusCode == 202 {
				return nil
			}

			return framework.NewError(
				"POST with respond-async must return 201 (synchronous) or 202 (asynchronous), not " +
					fmt.Sprintf("%d", resp.StatusCode))
		},
	)

	// Test 6: Service handles multiple concurrent async requests
	suite.AddTest(
		"test_multiple_async_requests",
		"Service handles multiple async requests",
		func(ctx *framework.TestContext) error {
			resp1, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			resp2, err := ctx.GET("/Categories", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			// Both should succeed (200 or 202)
			if (resp1.StatusCode == 200 || resp1.StatusCode == 202) &&
				(resp2.StatusCode == 200 || resp2.StatusCode == 202) {
				return nil
			}

			return framework.NewError("Service should handle multiple async requests")
		},
	)

	// Test 7: Status-monitor lifecycle — poll Location URL after 202 (OData §13.1)
	// If the server returns 202, the Location URL is a status monitor that must:
	//   • return 200 with the completed response body (done), OR
	//   • return 202 with a new Location header (still processing), OR
	//   • return 301/302 redirecting to the result resource.
	suite.AddTest(
		"test_async_status_monitor_lifecycle",
		"Status monitor URL after 202 returns 200 (done), 202 (pending), or 301/302 (redirect) (§13.1)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}
			if resp.StatusCode != 202 {
				return ctx.Skip("server does not support asynchronous processing (responded synchronously)")
			}

			location := resp.Headers.Get("Location")
			if location == "" {
				return framework.NewError("202 response missing required Location header (§13.1)")
			}

			// Extract relative path from the Location header.
			path := location
			if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
				parts := strings.SplitN(location, "/", 4)
				if len(parts) >= 4 {
					path = "/" + parts[3]
				}
			}

			// Poll the status monitor URL once (no retry loop — we don't want to spin).
			monitorResp, err := ctx.GET(path)
			if err != nil {
				return fmt.Errorf("failed to GET status monitor URL %q: %w", path, err)
			}

			switch monitorResp.StatusCode {
			case 200:
				// Completed: response body MUST be a valid JSON collection or entity.
				var result map[string]interface{}
				if err := json.Unmarshal(monitorResp.Body, &result); err != nil {
					return fmt.Errorf("status monitor 200 response is not valid JSON: %w", err)
				}
				return nil
			case 202:
				// Still processing: per §13.1, response MUST include a new Location (or same).
				newLoc := monitorResp.Headers.Get("Location")
				if newLoc == "" {
					return framework.NewError("status monitor 202 (still processing) missing Location header (§13.1)")
				}
				return nil
			case 301, 302:
				// Redirect to completed result: Location must be present.
				redirectLoc := monitorResp.Headers.Get("Location")
				if redirectLoc == "" {
					return fmt.Errorf("status monitor %d redirect missing Location header", monitorResp.StatusCode)
				}
				return nil
			default:
				return fmt.Errorf("status monitor URL returned unexpected status %d (expected 200/202/301/302)", monitorResp.StatusCode)
			}
		},
	)

	// Test 8: DELETE on status monitor URL cancels the async request (OData §13.1.2)
	suite.AddTest(
		"test_async_cancel_via_delete",
		"DELETE on status monitor URL cancels the async request (§13.1.2)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}
			if resp.StatusCode != 202 {
				return ctx.Skip("server does not support asynchronous processing (responded synchronously)")
			}

			location := resp.Headers.Get("Location")
			if location == "" {
				return framework.NewError("202 response missing required Location header (§13.1)")
			}

			path := location
			if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
				parts := strings.SplitN(location, "/", 4)
				if len(parts) >= 4 {
					path = "/" + parts[3]
				}
			}

			// Per §13.1.2: DELETE on the status monitor URL cancels the request.
			// The server MUST return 204 No Content (or 501 if not supported).
			cancelResp, err := ctx.DELETE(path)
			if err != nil {
				return fmt.Errorf("failed to DELETE status monitor URL: %w", err)
			}
			if cancelResp.StatusCode == 501 {
				return ctx.Skip("async cancellation via DELETE not supported (501)")
			}
			if cancelResp.StatusCode != 204 {
				return fmt.Errorf("DELETE on status monitor expected 204, got %d", cancelResp.StatusCode)
			}
			return nil
		},
	)

	return suite
}
