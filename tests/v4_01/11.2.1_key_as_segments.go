package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// KeyAsSegments creates the optional key-as-segments URL convention test suite.
// It validates that /EntitySet/{key} resolves equivalently to /EntitySet({key})
// and remains available when a 4.01 service constrains its response to 4.0.
func KeyAsSegments() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Key-As-Segments URL Convention",
		"Validates optional key-as-segment URL addressing for entity and property access under both response-version constraints.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_KeyasSegmentConvention",
	)
	suite.RequiredCapabilities = []framework.RequiredCapability{
		framework.Require(framework.CapKeyAsSegment, ""),
	}

	// Helper: fetch the ID of the first product.
	getFirstProductID := func(ctx *framework.TestContext) (string, error) {
		return firstEntityID(ctx, "Products")
	}

	// Test 1: /Products/{id} with OData-MaxVersion: 4.01 returns the same entity as /Products({id}).
	suite.AddTest(
		"test_key_as_segment_entity_401",
		"GET /Products/{id} with OData-MaxVersion: 4.01 resolves same entity as /Products({id})",
		func(ctx *framework.TestContext) error {
			id, err := getFirstProductID(ctx)
			if err != nil {
				return err
			}

			headers401 := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}

			// Parenthetical form (baseline)
			parenResp, err := ctx.GET(fmt.Sprintf("/Products(%s)", id), headers401...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(parenResp, http.StatusOK); err != nil {
				return fmt.Errorf("parenthetical key form failed: %w", err)
			}

			// Key-as-segment form
			segResp, err := ctx.GET(fmt.Sprintf("/Products/%s", id), headers401...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(segResp, http.StatusOK); err != nil {
				return fmt.Errorf("key-as-segment form failed (expected 200): %w", err)
			}

			// Both responses must return the same entity ID
			var parenBody, segBody map[string]interface{}
			if err := json.Unmarshal(parenResp.Body, &parenBody); err != nil {
				return fmt.Errorf("failed to parse parenthetical response: %w", err)
			}
			if err := json.Unmarshal(segResp.Body, &segBody); err != nil {
				return fmt.Errorf("failed to parse key-as-segment response: %w", err)
			}

			if fmt.Sprintf("%v", parenBody["ID"]) != fmt.Sprintf("%v", segBody["ID"]) {
				return fmt.Errorf("entity ID mismatch: parenthetical=%v, key-as-segment=%v",
					parenBody["ID"], segBody["ID"])
			}
			return nil
		},
	)

	// OData 4.01 Protocol §13.2.1(9) requires supported URL conventions
	// regardless of the requested OData-MaxVersion.
	suite.AddTest(
		"test_key_as_segment_active_40",
		"GET /Products/{id} with OData-MaxVersion: 4.0 still applies supported key-as-segments",
		func(ctx *framework.TestContext) error {
			id, err := getFirstProductID(ctx)
			if err != nil {
				return err
			}

			headers40 := []framework.Header{{Key: "OData-MaxVersion", Value: "4.0"}}
			resp, err := ctx.GET(fmt.Sprintf("/Products/%s", id), headers40...)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return fmt.Errorf("supported key-as-segment syntax must work regardless of OData-MaxVersion: %w", err)
			}
			return nil
		},
	)

	// Test 3: /Products/{id}/Name (structural property via key-as-segment) under 4.01.
	suite.AddTest(
		"test_key_as_segment_property_access_401",
		"GET /Products/{id}/Name with OData-MaxVersion: 4.01 returns property value",
		func(ctx *framework.TestContext) error {
			id, err := getFirstProductID(ctx)
			if err != nil {
				return err
			}

			headers401 := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			resp, err := ctx.GET(fmt.Sprintf("/Products/%s/Name", id), headers401...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return fmt.Errorf("property access via key-as-segment failed: %w", err)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("failed to parse property response: %w", err)
			}
			if _, ok := body["value"]; !ok {
				return fmt.Errorf("property response missing 'value' field, got: %v", body)
			}
			return nil
		},
	)

	// Test 4: Non-existent key via key-as-segment returns 404 under 4.01.
	suite.AddTest(
		"test_key_as_segment_nonexistent_401",
		"GET /Products/00000000-0000-0000-0000-000000000000 with OData-MaxVersion: 4.01 returns 404",
		func(ctx *framework.TestContext) error {
			headers401 := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			resp, err := ctx.GET("/Products/00000000-0000-0000-0000-000000000000", headers401...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusNotFound); err != nil {
				return fmt.Errorf("non-existent entity via key-as-segment should return 404: %w", err)
			}
			return nil
		},
	)

	return suite
}
