package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// Upsert creates the 11.4.5 Upsert Operations test suite
func Upsert() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.5 Upsert Operations",
		"Tests upsert (PUT) operations according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_UpsertanEntity",
	)

	// Test 1: PUT to existing entity updates it
	suite.AddTest(
		"test_put_update_existing",
		"PUT updates existing entity",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Original Product", 99.99)
			if err != nil {
				return err
			}
			// First create an entity
			createResp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				return fmt.Errorf("failed to create entity (status: %d)", createResp.StatusCode)
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			id, err := parseEntityID(createResult["ID"])
			if err != nil {
				return err
			}

			// Now PUT to update it
			replacement, err := buildProductPayload(ctx, "Updated Product", 199.99)
			if err != nil {
				return err
			}
			replacement["Description"] = "Updated via PUT"
			putResp, err := ctx.PUT(fmt.Sprintf("/Products(%s)", id), replacement)
			if err != nil {
				return err
			}

			if putResp.StatusCode != 200 && putResp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204, got %d", putResp.StatusCode)
			}

			getResp, err := ctx.GET(fmt.Sprintf("/Products(%s)", id))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(getResp, 200); err != nil {
				return err
			}

			var updated map[string]interface{}
			if err := json.Unmarshal(getResp.Body, &updated); err != nil {
				return fmt.Errorf("failed to parse updated entity: %w", err)
			}
			if updated["Name"] != "Updated Product" {
				return fmt.Errorf("expected updated Name, got %v", updated["Name"])
			}
			if updated["Price"] != 199.99 {
				return fmt.Errorf("expected updated Price 199.99, got %v", updated["Price"])
			}
			if updated["Description"] != "Updated via PUT" {
				return fmt.Errorf("expected updated Description, got %v", updated["Description"])
			}

			return nil
		},
	)

	// Product keys are server-generated; §11.4.4 forbids upserting one at a
	// client-selected URL even if upsert is supported for other entity sets.
	suite.AddTest(
		"test_put_create_nonexistent",
		"PUT cannot upsert a Product with a server-generated key",
		func(ctx *framework.TestContext) error {
			const nonexistentID = "00000000-0000-0000-0000-000000000000"
			path := fmt.Sprintf("/Products(%s)", nonexistentID)
			resp, err := ctx.PUT(path, map[string]interface{}{
				"ID":          nonexistentID,
				"Name":        "Upserted Product",
				"Price":       299.99,
				"Description": "Created via PUT",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode < 400 || resp.StatusCode >= 500 {
				return fmt.Errorf("expected 4xx for generated-key upsert, got %d", resp.StatusCode)
			}
			getResp, err := ctx.GET(path)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(getResp, 404)
		},
	)

	// Test 3: PUT replaces the entity, resetting omitted non-key properties to
	// their default/null values (full-replacement semantics, §11.4.3).
	suite.AddTest(
		"test_put_incomplete_entity",
		"PUT missing a required property without a default is rejected",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Test Product", 50.00)
			if err != nil {
				return err
			}
			// First create an entity
			createResp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				return fmt.Errorf("failed to create entity")
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			id, err := parseEntityID(createResult["ID"])
			if err != nil {
				return err
			}

			// Try PUT with incomplete data
			putResp, err := ctx.PUT(fmt.Sprintf("/Products(%s)", id), map[string]interface{}{
				"Name": "Incomplete",
			})
			if err != nil {
				return err
			}

			if putResp.StatusCode != 400 {
				return fmt.Errorf("expected 400 for omitted non-nullable Price without a default, got %d", putResp.StatusCode)
			}

			getResp, err := ctx.GET(fmt.Sprintf("/Products(%s)", id))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(getResp, 200); err != nil {
				return err
			}

			var updated map[string]interface{}
			if err := json.Unmarshal(getResp.Body, &updated); err != nil {
				return fmt.Errorf("failed to parse replaced entity: %w", err)
			}
			if updated["Name"] != "Test Product" {
				return fmt.Errorf("rejected PUT changed Name to %v", updated["Name"])
			}
			if updated["Price"] != 50.0 {
				return fmt.Errorf("rejected PUT changed Price to %v", updated["Price"])
			}

			return nil
		},
	)

	// Test 4: PUT should return proper headers
	suite.AddTest(
		"test_put_response_headers",
		"PUT response includes proper headers",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Header Test", 75.00)
			if err != nil {
				return err
			}
			// First create an entity
			createResp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				return fmt.Errorf("failed to create entity")
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			id, err := parseEntityID(createResult["ID"])
			if err != nil {
				return err
			}

			// PUT to update
			replacement, err := buildProductPayload(ctx, "Header Test Product", 99.99)
			if err != nil {
				return err
			}
			replacement["Description"] = "Testing headers"
			putResp, err := ctx.PUT(fmt.Sprintf("/Products(%s)", id), replacement)
			if err != nil {
				return err
			}

			if putResp.StatusCode != 200 && putResp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204, got %d", putResp.StatusCode)
			}

			// Check for OData-Version header
			version := putResp.Headers.Get("OData-Version")
			if version == "" {
				return fmt.Errorf("OData-Version header missing")
			}

			return nil
		},
	)

	// Test 5: PUT with If-Match header
	suite.AddTest(
		"test_put_if_match",
		"PUT with If-Match for optimistic concurrency",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ETag Test", 85.00)
			if err != nil {
				return err
			}
			// First create an entity
			createResp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				return fmt.Errorf("failed to create entity")
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			id, err := parseEntityID(createResult["ID"])
			if err != nil {
				return err
			}

			// Get the entity to retrieve ETag
			getResp, err := ctx.GET(fmt.Sprintf("/Products(%s)", id))
			if err != nil {
				return err
			}

			etag := getResp.Headers.Get("ETag")
			if etag != "" {
				// If ETag is supported, try PUT with If-Match
				replacement, err := buildProductPayload(ctx, "Conditional Update", 149.99)
				if err != nil {
					return err
				}
				replacement["Description"] = "With ETag"
				putResp, err := ctx.PUT(fmt.Sprintf("/Products(%s)", id), replacement, framework.Header{Key: "If-Match", Value: etag})
				if err != nil {
					return err
				}

				if putResp.StatusCode != 200 && putResp.StatusCode != 204 {
					return fmt.Errorf("expected status 200 or 204, got %d", putResp.StatusCode)
				}
			}
			// ETags are optional, so pass if not supported

			return nil
		},
	)

	return suite
}
