package v4_01

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// JSONBatch creates the OData JSON Format v4.01 §19 JSON Batch Requests compliance test suite.
//
// JSON batch is a 4.01-only feature.  The suite verifies:
//   - Basic JSON batch acceptance and response format (§19.2 / §19.5)
//   - Per-request id echo
//   - dependsOn failure propagation (424 Failed Dependency)
//   - atomicityGroup rollback-on-failure semantics
//   - Prefer: continue-on-error support
//   - Rejection of JSON batch when the client negotiates OData 4.0 (version gating)
func JSONBatch() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"19 JSON Batch Requests (OData 4.01)",
		"Tests OData 4.01 JSON batch request and response semantics.",
		"https://docs.oasis-open.org/odata/odata-json-format/v4.01/odata-json-format-v4.01.html#sec_BatchRequestsandResponses",
	)

	// ── helpers ──────────────────────────────────────────────────────────────

	// postJSONBatch sends a JSON batch request and returns the decoded response envelope.
	postJSONBatch := func(ctx *framework.TestContext, body map[string]interface{}, headers ...framework.Header) (*framework.HTTPResponse, error) {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal batch body: %w", err)
		}
		hdrs := append([]framework.Header{{Key: "Content-Type", Value: "application/json"}}, headers...)
		return ctx.POSTRaw("/$batch", b, "application/json", hdrs...)
	}

	makeRequests := func(reqs ...map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{"requests": reqs}
	}

	parseResponses := func(body []byte) (map[string]map[string]interface{}, error) {
		var envelope struct {
			Responses []map[string]interface{} `json:"responses"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			return nil, fmt.Errorf("failed to parse JSON batch response: %w", err)
		}
		responses := make(map[string]map[string]interface{}, len(envelope.Responses))
		for i, response := range envelope.Responses {
			id, ok := response["id"].(string)
			if !ok || id == "" {
				return nil, fmt.Errorf("response item %d is missing its request id", i)
			}
			if _, duplicate := responses[id]; duplicate {
				return nil, fmt.Errorf("response id %q occurs more than once", id)
			}
			if _, ok := response["status"].(float64); !ok {
				return nil, fmt.Errorf("response %q is missing its numeric status", id)
			}
			responses[id] = response
		}
		return responses, nil
	}

	expectRejected := func(ctx *framework.TestContext, body map[string]interface{}) error {
		resp, err := postJSONBatch(ctx, body)
		if err != nil {
			return err
		}
		if resp.StatusCode != 400 {
			return fmt.Errorf("invalid JSON batch status = %d, want 400: %s", resp.StatusCode, string(resp.Body))
		}
		return nil
	}

	// ── §19.2  JSON batch envelope accepted ──────────────────────────────────

	suite.AddTest(
		"test_json_batch_accepted",
		"JSON batch (application/json) is accepted and returns 200",
		func(ctx *framework.TestContext) error {
			resp, err := postJSONBatch(ctx, makeRequests(map[string]interface{}{
				"id":     "r1",
				"method": "GET",
				"url":    "Products",
			}))
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// ── §19.5  JSON batch response format ────────────────────────────────────

	suite.AddTest(
		"test_json_batch_response_format",
		"JSON batch response has application/json Content-Type and 'responses' array",
		func(ctx *framework.TestContext) error {
			resp, err := postJSONBatch(ctx, makeRequests(map[string]interface{}{
				"id":     "r1",
				"method": "GET",
				"url":    "Products",
			}))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			ct := resp.Headers.Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				return framework.NewError(fmt.Sprintf("expected application/json Content-Type, got %q", ct))
			}

			var envelope map[string]interface{}
			if err := json.Unmarshal(resp.Body, &envelope); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse JSON batch response: %v", err))
			}
			if _, ok := envelope["responses"]; !ok {
				return framework.NewError("JSON batch response must contain a 'responses' member")
			}
			return nil
		},
	)

	// ── §19.5  Per-request id is echoed ──────────────────────────────────────

	suite.AddTest(
		"test_json_batch_id_echoed",
		"Each response object echoes the id of the corresponding request",
		func(ctx *framework.TestContext) error {
			resp, err := postJSONBatch(ctx, makeRequests(map[string]interface{}{
				"id":     "my-unique-request-id",
				"method": "GET",
				"url":    "Products",
			}))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			responses, err := parseResponses(resp.Body)
			if err != nil {
				return err
			}
			if len(responses) == 0 {
				return framework.NewError("expected at least one response")
			}
			if _, ok := responses["my-unique-request-id"]; !ok {
				return framework.NewError("response did not echo request id \"my-unique-request-id\"")
			}
			return nil
		},
	)

	// ── §19.2  Multiple independent requests ─────────────────────────────────

	suite.AddTest(
		"test_json_batch_multiple_requests",
		"Multiple independent requests in a JSON batch all receive responses",
		func(ctx *framework.TestContext) error {
			resp, err := postJSONBatch(ctx, makeRequests(
				map[string]interface{}{"id": "r1", "method": "GET", "url": "Products"},
				map[string]interface{}{"id": "r2", "method": "GET", "url": "Products"},
			))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			responses, err := parseResponses(resp.Body)
			if err != nil {
				return err
			}
			if len(responses) != 2 {
				return framework.NewError(fmt.Sprintf("expected 2 responses, got %d", len(responses)))
			}
			for _, id := range []string{"r1", "r2"} {
				if _, ok := responses[id]; !ok {
					return fmt.Errorf("batch response is missing request id %q", id)
				}
			}
			return nil
		},
	)

	// ── §19.2  dependsOn failure propagation ─────────────────────────────────

	suite.AddTest(
		"test_json_batch_dependson_failure_propagation",
		"A request whose dependsOn dependency failed receives 424 Failed Dependency",
		func(ctx *framework.TestContext) error {
			// r1 will fail (non-existent entity); r2 depends on r1.
			resp, err := postJSONBatch(ctx, makeRequests(
				map[string]interface{}{
					"id":     "r1",
					"method": "GET",
					"url":    "Products(2147483647)", // highly unlikely to exist
				},
				map[string]interface{}{
					"id":        "r2",
					"method":    "GET",
					"url":       "Products",
					"dependsOn": []string{"r1"},
				},
			))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			responses, err := parseResponses(resp.Body)
			if err != nil {
				return err
			}
			if len(responses) != 2 {
				return framework.NewError(fmt.Sprintf("expected 2 responses, got %d", len(responses)))
			}

			r1, ok1 := responses["r1"]
			r2, ok2 := responses["r2"]
			if !ok1 || !ok2 {
				return framework.NewError("batch response is missing r1 or r2")
			}

			if r1["status"] == float64(200) {
				return framework.NewError("r1 should have failed (entity not found)")
			}
			if r2["status"] != float64(424) {
				return framework.NewError(fmt.Sprintf("expected r2 status 424, got %v", r2["status"]))
			}
			return nil
		},
	)

	// ── §19.2  atomicityGroup rollback semantics ──────────────────────────────

	suite.AddTest(
		"test_json_batch_atomicitygroup_rollback",
		"A failed atomicityGroup rolls back all changes and echoes the group id",
		func(ctx *framework.TestContext) error {
			// r1 succeeds (creates entity); r2 fails (non-existent delete).
			// Both are in the same atomicityGroup, so both should end up as 4xx.
			resp, err := postJSONBatch(ctx, makeRequests(
				map[string]interface{}{
					"id":             "r1",
					"method":         "POST",
					"url":            "Products",
					"headers":        map[string]interface{}{"Content-Type": "application/json"},
					"body":           map[string]interface{}{"Name": "AtomicRollbackTest", "Price": 1.0},
					"atomicityGroup": "g1",
				},
				map[string]interface{}{
					"id":             "r2",
					"method":         "DELETE",
					"url":            "Products(2147483647)", // non-existent
					"atomicityGroup": "g1",
				},
			))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			responses, err := parseResponses(resp.Body)
			if err != nil {
				return err
			}
			if len(responses) != 2 {
				return framework.NewError(fmt.Sprintf("expected 2 responses, got %d", len(responses)))
			}

			r1, ok1 := responses["r1"]
			r2, ok2 := responses["r2"]
			if !ok1 || !ok2 {
				return framework.NewError("batch response is missing r1 or r2")
			}
			for id, response := range map[string]map[string]interface{}{"r1": r1, "r2": r2} {
				if response["atomicityGroup"] != "g1" {
					return fmt.Errorf("response %s atomicityGroup = %v, want g1", id, response["atomicityGroup"])
				}
			}
			if r2["status"] == float64(200) || r2["status"] == float64(201) {
				return framework.NewError(fmt.Sprintf("expected r2 to fail, got status %v", r2["status"]))
			}
			// r1 itself succeeded when submitted, but the group failed and was
			// rolled back — its own response status must reflect that too, not
			// just the downstream side effect (checked below). A server that
			// returns 201 for r1 while silently discarding the row underneath
			// would otherwise only be caught indirectly.
			if r1["status"] == float64(200) || r1["status"] == float64(201) {
				return fmt.Errorf("expected r1's response status to reflect the atomicityGroup rollback (not 2xx), got %v", r1["status"])
			}

			// Verify the entity was NOT persisted.
			listResp, err := ctx.GET("/Products?$filter=Name eq 'AtomicRollbackTest'")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(listResp, 200); err != nil {
				return err
			}
			var listEnvelope map[string]interface{}
			if err := json.Unmarshal(listResp.Body, &listEnvelope); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse product list: %v", err))
			}
			items, _ := listEnvelope["value"].([]interface{})
			if len(items) > 0 {
				return framework.NewError("entity should have been rolled back but still exists")
			}
			return nil
		},
	)

	// ── §19.2  Prefer: continue-on-error ─────────────────────────────────────

	suite.AddTest(
		"test_json_batch_continue_on_error",
		"With Prefer: continue-on-error, independent requests run even after a failure",
		func(ctx *framework.TestContext) error {
			// r1 fails; r2 is independent (no dependsOn) and should execute.
			resp, err := postJSONBatch(ctx,
				makeRequests(
					map[string]interface{}{
						"id":     "r1",
						"method": "GET",
						"url":    "Products(2147483647)", // will fail
					},
					map[string]interface{}{
						"id":     "r2",
						"method": "GET",
						"url":    "Products",
					},
				),
				framework.Header{Key: "Prefer", Value: "continue-on-error"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			responses, err := parseResponses(resp.Body)
			if err != nil {
				return err
			}
			if len(responses) != 2 {
				return framework.NewError(fmt.Sprintf("expected 2 responses with continue-on-error, got %d", len(responses)))
			}

			r2, ok := responses["r2"]
			if !ok {
				return framework.NewError("batch response is missing r2")
			}
			if r2["status"] != float64(200) {
				return framework.NewError(fmt.Sprintf("expected r2 status 200 (continued after error), got %v", r2["status"]))
			}
			return nil
		},
	)

	suite.AddTest(
		"test_json_batch_default_continues_on_error",
		"Without continue-on-error=false, independent requests continue after a failure",
		func(ctx *framework.TestContext) error {
			resp, err := postJSONBatch(ctx, makeRequests(
				map[string]interface{}{"id": "r1", "method": "GET", "url": "Products(2147483647)"},
				map[string]interface{}{"id": "r2", "method": "GET", "url": "Products?$top=1"},
			))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			responses, err := parseResponses(resp.Body)
			if err != nil {
				return err
			}
			r2, ok := responses["r2"]
			if !ok {
				return framework.NewError("batch response is missing independent request r2")
			}
			if r2["status"] != float64(200) {
				return fmt.Errorf("independent r2 status = %v, want 200; processing stops only for continue-on-error=false", r2["status"])
			}
			return nil
		},
	)

	suite.AddTest(
		"test_json_batch_response_header_names_lowercase",
		"Header names inside JSON batch response objects are lowercase",
		func(ctx *framework.TestContext) error {
			resp, err := postJSONBatch(ctx, makeRequests(
				map[string]interface{}{"id": "r1", "method": "GET", "url": "Products?$top=1"},
			))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			responses, err := parseResponses(resp.Body)
			if err != nil {
				return err
			}
			r1, ok := responses["r1"]
			if !ok {
				return framework.NewError("batch response is missing r1")
			}
			headers, ok := r1["headers"].(map[string]interface{})
			if !ok || len(headers) == 0 {
				return framework.NewError("successful JSON batch response item is missing headers")
			}
			for name := range headers {
				if name != strings.ToLower(name) {
					return fmt.Errorf("JSON batch response header name %q is not lowercase", name)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_json_batch_requires_requests_member",
		"JSON batch envelope without the required requests member is rejected",
		func(ctx *framework.TestContext) error {
			return expectRejected(ctx, map[string]interface{}{})
		},
	)

	suite.AddTest(
		"test_json_batch_rejects_duplicate_ids",
		"JSON batch request identifiers must be unique",
		func(ctx *framework.TestContext) error {
			return expectRejected(ctx, makeRequests(
				map[string]interface{}{"id": "same", "method": "GET", "url": "/"},
				map[string]interface{}{"id": "same", "method": "GET", "url": "/"},
			))
		},
	)

	suite.AddTest(
		"test_json_batch_rejects_forward_dependency",
		"dependsOn cannot reference a later request",
		func(ctx *framework.TestContext) error {
			return expectRejected(ctx, makeRequests(
				map[string]interface{}{"id": "r1", "method": "GET", "url": "/", "dependsOn": []string{"r2"}},
				map[string]interface{}{"id": "r2", "method": "GET", "url": "/"},
			))
		},
	)

	suite.AddTest(
		"test_json_batch_requires_adjacent_atomicity_group",
		"Requests in the same atomicityGroup must be adjacent",
		func(ctx *framework.TestContext) error {
			return expectRejected(ctx, makeRequests(
				map[string]interface{}{"id": "r1", "method": "GET", "url": "/", "atomicityGroup": "g1"},
				map[string]interface{}{"id": "r2", "method": "GET", "url": "/"},
				map[string]interface{}{"id": "r3", "method": "GET", "url": "/", "atomicityGroup": "g1"},
			))
		},
	)

	suite.AddTest(
		"test_json_batch_separates_request_and_group_ids",
		"Request identifiers and atomicityGroup identifiers must be distinct",
		func(ctx *framework.TestContext) error {
			return expectRejected(ctx, makeRequests(
				map[string]interface{}{"id": "g1", "method": "GET", "url": "/"},
				map[string]interface{}{"id": "r2", "method": "GET", "url": "/", "atomicityGroup": "g1"},
			))
		},
	)

	// ── Version gating: JSON batch MUST NOT apply under OData-MaxVersion: 4.0 ─
	// Per the custom-agent guidelines for v4_01/ tests, this test MUST include:
	//   1. A negative assertion (4.0 client is rejected).
	//   2. A positive assertion (4.01 client is accepted).

	suite.AddTest(
		"test_json_batch_rejected_for_odata_40",
		"JSON batch is rejected when client negotiates OData-MaxVersion: 4.0 but accepted for 4.01",
		func(ctx *framework.TestContext) error {
			batchBody, _ := json.Marshal(makeRequests(map[string]interface{}{
				"id":     "r1",
				"method": "GET",
				"url":    "Products",
			}))

			// Negative assertion: JSON batch with OData-MaxVersion: 4.0 must be rejected.
			resp40, err := ctx.POSTRaw("/$batch", batchBody, "application/json",
				framework.Header{Key: "OData-MaxVersion", Value: "4.0"},
			)
			if err != nil {
				return err
			}
			if resp40.StatusCode < 400 {
				return framework.NewError(fmt.Sprintf(
					"expected a 4xx error when using JSON batch with OData-MaxVersion: 4.0, got %d",
					resp40.StatusCode,
				))
			}

			// Positive assertion: JSON batch with OData-MaxVersion: 4.01 must be accepted.
			resp401, err := ctx.POSTRaw("/$batch", batchBody, "application/json",
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}
			if resp401.StatusCode != 200 {
				return framework.NewError(fmt.Sprintf(
					"expected 200 when using JSON batch with OData-MaxVersion: 4.01, got %d",
					resp401.StatusCode,
				))
			}
			return nil
		},
	)

	// ── §19.5  Empty batch returns empty responses array ──────────────────────

	suite.AddTest(
		"test_json_batch_empty_requests",
		"An empty JSON batch returns an empty responses array",
		func(ctx *framework.TestContext) error {
			resp, err := postJSONBatch(ctx, map[string]interface{}{"requests": []interface{}{}})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var envelope map[string]interface{}
			if err := json.Unmarshal(resp.Body, &envelope); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse response: %v", err))
			}

			responses, ok := envelope["responses"].([]interface{})
			if !ok {
				// An empty array may be marshalled as nil by some decoders; treat nil as empty.
				if envelope["responses"] == nil {
					return nil
				}
				return framework.NewError(fmt.Sprintf("expected 'responses' array, got %T", envelope["responses"]))
			}
			if len(responses) != 0 {
				return framework.NewError(fmt.Sprintf("expected 0 responses, got %d", len(responses)))
			}
			return nil
		},
	)

	return suite
}
