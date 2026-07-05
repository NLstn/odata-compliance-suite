package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// UpdateEntity creates the 11.4.3 Update an Entity test suite
func UpdateEntity() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.3 Update an Entity",
		"Tests PATCH and PUT operations for updating entities according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_UpdateanEntity",
	)

	// Test 1: PATCH updates specified properties only
	suite.AddTest(
		"test_patch_update",
		"PATCH updates specified properties and preserves unspecified ones (partial update)",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "UpdateEntityPatch", 199.99)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)

			resp, err := ctx.PATCH(productPath, map[string]interface{}{
				"Price": 149.99,
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 204 && resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200 or 204, got %d", resp.StatusCode)
			}

			verifyResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(verifyResp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse verification JSON: %w", err)
			}

			price, ok := result["Price"].(float64)
			if !ok || price != 149.99 {
				return fmt.Errorf("PATCH: Price not updated correctly (got %v)", result["Price"])
			}

			// Verify the unpatched field (Name) was not destroyed — this is the partial-update guarantee.
			name, _ := result["Name"].(string)
			if name != "UpdateEntityPatch" {
				return fmt.Errorf("PATCH partial update: Name=%q should still be %q (unpatched field must be preserved)", name, "UpdateEntityPatch")
			}

			return nil
		},
	)

	// Test 1b: PATCH round-trip — Name field
	suite.AddTest(
		"test_patch_update_name_roundtrip",
		"PATCH Name field and verify via follow-up GET",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "PatchRoundTripName", 55.00)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)

			resp, err := ctx.PATCH(productPath, map[string]interface{}{
				"Name": "PatchRoundTripName-Updated",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 204 && resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200 or 204, got %d", resp.StatusCode)
			}

			// Round-trip: verify the patched field was persisted
			verifyResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(verifyResp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(verifyResp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse verification JSON: %w", err)
			}

			name, ok := result["Name"].(string)
			if !ok || name != "PatchRoundTripName-Updated" {
				return fmt.Errorf("expected Name %q after PATCH, got %v", "PatchRoundTripName-Updated", result["Name"])
			}

			ctx.Log("PATCH round-trip verified: Name updated correctly")
			return nil
		},
	)

	// Test 2: PATCH with invalid property returns error
	suite.AddTest(
		"test_patch_invalid_property",
		"PATCH with invalid property returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "UpdateEntityInvalidProp", 199.99)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)

			resp, err := ctx.PATCH(productPath, map[string]interface{}{
				"NonExistentProperty": "value",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 3: PATCH to non-existent entity returns 404
	suite.AddTest(
		"test_patch_not_found",
		"PATCH to non-existent entity returns 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.PATCH("/Products(00000000-0000-0000-0000-000000000000)", map[string]interface{}{
				"Price": 100,
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 4: Content-Type header validation
	suite.AddTest(
		"test_patch_no_content_type",
		"PATCH without Content-Type returns 400 or 415",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "UpdateEntityNoContentType", 250.00)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)
			payload, err := json.Marshal(map[string]interface{}{
				"Price": 99.99,
			})
			if err != nil {
				return fmt.Errorf("failed to marshal payload: %w", err)
			}
			resp, err := ctx.PATCHRawNoContentType(productPath, payload)
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 && resp.StatusCode != 415 {
				return fmt.Errorf("expected status 400 or 415, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// PUT tests - full entity replacement
	suite.AddTest(
		"test_put_full_replacement",
		"PUT replaces entire entity (full update)",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "UpdateEntityPut", 299.99)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)

			// Get the current entity to get required fields
			getResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(getResp, 200); err != nil {
				return err
			}

			var current map[string]interface{}
			if err := json.Unmarshal(getResp.Body, &current); err != nil {
				return fmt.Errorf("failed to parse current entity: %w", err)
			}

			// Create complete replacement entity
			replacement := map[string]interface{}{
				"Name":       "Completely Replaced Product",
				"Price":      399.99,
				"CategoryID": current["CategoryID"], // Keep required foreign key
				"Status":     current["Status"],     // Keep required status
			}

			resp, err := ctx.PUT(productPath, replacement,
				framework.Header{Key: "Content-Type", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			// Should return 200 or 204
			if resp.StatusCode != 200 && resp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204, got %d", resp.StatusCode)
			}

			// Verify the replacement
			verifyResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(verifyResp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse verification JSON: %w", err)
			}

			// Check that all fields were replaced
			if name, ok := result["Name"].(string); !ok || name != "Completely Replaced Product" {
				return fmt.Errorf("name not replaced correctly")
			}

			if price, ok := result["Price"].(float64); !ok || price != 399.99 {
				return fmt.Errorf("price not replaced correctly")
			}

			ctx.Log("Entity fully replaced with PUT")
			return nil
		},
	)

	suite.AddTest(
		"test_put_response_body_validation",
		"PUT with 200 response includes updated entity body",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "PutResponseBody", 199.99)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)

			// Get current entity for required fields
			getResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			var current map[string]interface{}
			if err := json.Unmarshal(getResp.Body, &current); err != nil {
				return err
			}

			replacement := map[string]interface{}{
				"Name":       "Updated Name",
				"Price":      249.99,
				"CategoryID": current["CategoryID"],
				"Status":     current["Status"],
			}

			resp, err := ctx.PUT(productPath, replacement,
				framework.Header{Key: "Content-Type", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			if resp.StatusCode == 204 {
				// 204 No Content is fine, body validation not needed
				ctx.Log("PUT returned 204 No Content (valid)")
				return nil
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200 or 204, got %d", resp.StatusCode)
			}

			// With 200, body should contain updated entity
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("200 response should have valid JSON body: %w", err)
			}

			// Should have ID field
			if _, ok := result["ID"]; !ok {
				return fmt.Errorf("200 response body should include entity with ID")
			}

			// Should have updated values
			if name, ok := result["Name"].(string); !ok || name != "Updated Name" {
				return fmt.Errorf("200 response body should reflect updated values")
			}

			ctx.Log("PUT with 200 response correctly includes updated entity")
			return nil
		},
	)

	suite.AddTest(
		"test_put_nonexistent_entity",
		"PUT on non-existent entity returns 404",
		func(ctx *framework.TestContext) error {
			nonExistentPath := nonExistingEntityPath("Products")

			// Get a valid CategoryID first
			categoryID, err := firstEntityID(ctx, "Categories")
			if err != nil {
				return err
			}

			replacement := map[string]interface{}{
				"Name":       "Should Not Be Created",
				"Price":      99.99,
				"CategoryID": categoryID,
				"Status":     1,
			}

			resp, err := ctx.PUT(nonExistentPath, replacement,
				framework.Header{Key: "Content-Type", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			// PUT should not create new entities - use POST for that
			// Should return 404
			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404 for PUT on non-existent entity, got %d", resp.StatusCode)
			}

			ctx.Log("Correctly returned 404 for PUT on non-existent entity")
			return nil
		},
	)

	// Deep update via PATCH: include an inline related entity in the body (§11.4.3).
	// The server MAY support deep update. If it does (200/204), the related entity
	// must reflect the supplied inline values. If it returns 400/501, the test skips.
	suite.AddTest(
		"test_patch_deep_update_inline_collection",
		"PATCH with inline collection-valued navigation updates related entities (§11.4.3, MAY)",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "DeepUpdateInlineTest", 77.77)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)

			// Create a Description for the product first so we have one to update.
			_, err = ctx.POST("/ProductDescriptions", map[string]interface{}{
				"ProductID":   productID,
				"LanguageKey": "EN",
				"Description": "Before deep update",
			})
			if err != nil {
				return err
			}

			// Now PATCH the Product with an inline updated Description.
			patchResp, err := ctx.PATCH(productPath, map[string]interface{}{
				"Name": "DeepUpdateInlineTest-Updated",
				"Descriptions": []map[string]interface{}{
					{
						"LanguageKey": "EN",
						"Description": "After deep update",
					},
				},
			})
			if err != nil {
				return err
			}
			if patchResp.StatusCode == 400 || patchResp.StatusCode == 501 {
				return ctx.Skip("server does not support deep update of collection-valued navigation (400/501) — MAY support per §11.4.3")
			}
			if patchResp.StatusCode != 200 && patchResp.StatusCode != 204 {
				return fmt.Errorf("deep update PATCH returned unexpected status %d", patchResp.StatusCode)
			}

			// Verify the top-level entity was updated.
			getResp, err := ctx.GET(productPath + "?$expand=Descriptions")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(getResp, 200); err != nil {
				return err
			}
			var body map[string]interface{}
			if err := json.Unmarshal(getResp.Body, &body); err != nil {
				return fmt.Errorf("failed to parse response after deep update: %w", err)
			}
			name, _ := body["Name"].(string)
			if name != "DeepUpdateInlineTest-Updated" {
				return fmt.Errorf("deep update: product Name not updated (got %q)", name)
			}

			// Verify the related Description was also updated.
			descs, _ := body["Descriptions"].([]interface{})
			for _, d := range descs {
				desc, ok := d.(map[string]interface{})
				if !ok {
					continue
				}
				if desc["LanguageKey"] == "EN" {
					descText, _ := desc["Description"].(string)
					if descText != "After deep update" {
						return fmt.Errorf("deep update: EN description not updated (got %q)", descText)
					}
					return nil
				}
			}
			return fmt.Errorf("deep update: EN description not found in expanded Descriptions after PATCH")
		},
	)

	return suite
}
