package v4_0

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// NullValueHandling creates the 11.4.14 Null Value Handling test suite
func NullValueHandling() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.14 Null Value Handling",
		"Tests that the service properly handles null values in entity creation, updates, and filtering.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html",
	)

	var createdID string

	// Test 1: Create entity with null property
	suite.AddTest(
		"test_create_with_null",
		"Create entity with explicit null property",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{
				"Name":        "Null Test Product",
				"Price":       99.99,
				"Description": nil,
				"Status":      1,
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}

			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			if id, ok := data["ID"].(string); ok {
				createdID = id
			}

			return nil
		},
	)

	// Test 2: Retrieve entity with null property
	suite.AddTest(
		"test_retrieve_null_property",
		"Retrieve entity returns null property correctly",
		func(ctx *framework.TestContext) error {
			if createdID == "" {
				return framework.NewError("No test entity available")
			}

			resp, err := ctx.GET(fmt.Sprintf("/Products(%s)", createdID))
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			description, ok := data["Description"]
			if ok && description != nil {
				return framework.NewError("Expected Description to be null when present")
			}

			return nil
		},
	)

	// Test 3: Update property to null using PATCH
	suite.AddTest(
		"test_patch_to_null",
		"Update property to null using PATCH",
		func(ctx *framework.TestContext) error {
			if createdID == "" {
				return framework.NewError("No test entity available")
			}

			payload := map[string]interface{}{
				"Description": nil,
			}

			resp, err := ctx.PATCH(fmt.Sprintf("/Products(%s)", createdID), payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// Should return 204 or 200
			if err := ctx.AssertStatusCode(resp, 204); err != nil {
				if err := ctx.AssertStatusCode(resp, 200); err != nil {
					return err
				}
			}

			// Verify the property actually became null, not just that the
			// PATCH request itself returned a success status.
			verifyResp, err := ctx.GET(fmt.Sprintf("/Products(%s)", createdID))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(verifyResp, 200); err != nil {
				return err
			}
			var verified map[string]interface{}
			if err := ctx.GetJSON(verifyResp, &verified); err != nil {
				return err
			}
			if desc, hasKey := verified["Description"]; hasKey && desc != nil {
				return fmt.Errorf("expected Description to be null after PATCH, got %v", desc)
			}

			return nil
		},
	)

	// Test 3b: PATCH omitting a property leaves its existing value unchanged
	// (the partial-update contrast to explicit-null above).
	suite.AddTest(
		"test_patch_omit_is_noop",
		"PATCH omitting a property preserves its existing value (contrast with explicit null)",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{
				"Name":        "Omit Test Product",
				"Price":       55.00,
				"Description": "Description that must survive an unrelated PATCH",
				"Status":      1,
			}
			createResp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(createResp, 201); err != nil {
				return err
			}
			var created map[string]interface{}
			if err := ctx.GetJSON(createResp, &created); err != nil {
				return err
			}
			id, err := parseEntityID(created["ID"])
			if err != nil {
				return err
			}
			entityPath := fmt.Sprintf("/Products(%s)", id)

			// PATCH a different property; Description is not mentioned at all.
			patchResp, err := ctx.PATCH(entityPath, map[string]interface{}{
				"Name": "Omit Test Product - Renamed",
			}, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}
			if patchResp.StatusCode != 200 && patchResp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204, got %d", patchResp.StatusCode)
			}

			verifyResp, err := ctx.GET(entityPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(verifyResp, 200); err != nil {
				return err
			}
			var verified map[string]interface{}
			if err := ctx.GetJSON(verifyResp, &verified); err != nil {
				return err
			}
			desc, _ := verified["Description"].(string)
			if desc != "Description that must survive an unrelated PATCH" {
				return fmt.Errorf("PATCH omitting Description must not change it; got %q", desc)
			}
			return nil
		},
	)

	// Test 3c: PUT (full replacement) resets an omitted nullable property to
	// null — the actual subject of §11.4.14, distinct from PATCH's partial
	// semantics above.
	suite.AddTest(
		"test_put_omitted_nullable_resets_to_null",
		"PUT replacement resets an omitted nullable property to null",
		func(ctx *framework.TestContext) error {
			categoryID, err := firstEntityID(ctx, "Categories")
			if err != nil {
				return err
			}
			payload := map[string]interface{}{
				"Name":        "PUT Reset Test Product",
				"Price":       65.00,
				"Description": "Description that must be reset by PUT",
				"CategoryID":  categoryID,
				"Status":      1,
			}
			createResp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(createResp, 201); err != nil {
				return err
			}
			var created map[string]interface{}
			if err := ctx.GetJSON(createResp, &created); err != nil {
				return err
			}
			id, err := parseEntityID(created["ID"])
			if err != nil {
				return err
			}
			entityPath := fmt.Sprintf("/Products(%s)", id)

			// Full replacement omitting Description entirely.
			putResp, err := ctx.PUT(entityPath, map[string]interface{}{
				"Name":       "PUT Reset Test Product - Replaced",
				"Price":      75.00,
				"CategoryID": categoryID,
				"Status":     1,
			}, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}
			if putResp.StatusCode != 200 && putResp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204, got %d", putResp.StatusCode)
			}

			verifyResp, err := ctx.GET(entityPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(verifyResp, 200); err != nil {
				return err
			}
			var verified map[string]interface{}
			if err := ctx.GetJSON(verifyResp, &verified); err != nil {
				return err
			}
			if desc, hasKey := verified["Description"]; hasKey && desc != nil {
				return fmt.Errorf("expected Description to be reset to null by PUT (omitted from the replacement body), got %v", desc)
			}
			return nil
		},
	)

	// Test 4: Filter for null values — every returned product must have Description == null.
	suite.AddTest(
		"test_filter_eq_null",
		"Filter for null values: every returned entity has the filtered property as null",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Description eq null")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			for i, p := range items {
				desc, hasKey := p["Description"]
				// Description must be absent or explicitly null.
				if hasKey && desc != nil {
					return fmt.Errorf("Products[%d] has non-null Description=%v but filter was Description eq null", i, desc)
				}
			}
			return nil
		},
	)

	// Test 5: Filter for non-null values — every returned product must have Description != null.
	suite.AddTest(
		"test_filter_ne_null",
		"Filter for non-null values: every returned entity has the filtered property as non-null",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Description ne null")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			for i, p := range items {
				desc, hasKey := p["Description"]
				if !hasKey || desc == nil {
					return fmt.Errorf("Products[%d] has null/absent Description but filter was Description ne null", i)
				}
			}
			return nil
		},
	)

	return suite
}
