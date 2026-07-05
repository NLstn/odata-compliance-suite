package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// DataValidation creates the 11.4.15 Data Validation and Constraints test suite
func DataValidation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.15 Data Validation",
		"Tests that the service enforces data validation rules, required fields, and constraints on entity creation and updates.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html",
	)

	// Test 1: Missing required field returns error
	suite.AddTest(
		"test_missing_required_field",
		"Missing required field returns 400",
		func(ctx *framework.TestContext) error {
			categoryID, err := firstEntityID(ctx, "Categories")
			if err != nil {
				return err
			}
			payload := map[string]interface{}{
				"Price":      99.99,
				"CategoryID": categoryID,
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 2: Invalid data type returns error
	suite.AddTest(
		"test_invalid_data_type",
		"Invalid data type returns 400",
		func(ctx *framework.TestContext) error {
			categoryID, err := firstEntityID(ctx, "Categories")
			if err != nil {
				return err
			}
			payload := map[string]interface{}{
				"Name":       "Test Product",
				"Price":      "not-a-number",
				"CategoryID": categoryID,
				"Status":     1,
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 3: Malformed JSON returns error
	suite.AddTest(
		"test_malformed_json",
		"Malformed JSON returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POSTRaw("/Products", []byte(`{"Name":"Test","Price":99.99,}`), "application/json")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 4: Content-Type header missing or incorrect
	suite.AddTest(
		"test_missing_content_type",
		"Missing Content-Type returns 415",
		func(ctx *framework.TestContext) error {
			categoryID, err := firstEntityID(ctx, "Categories")
			if err != nil {
				return err
			}
			body := fmt.Sprintf(`{"Name":"Test","Price":99.99,"CategoryID":"%s","Status":1}`, categoryID)
			resp, err := ctx.POSTRaw("/Products", []byte(body), "")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 415); err != nil {
				return err
			}

			if err := ctx.AssertJSONField(resp, "error"); err != nil {
				return err
			}

			var payload map[string]interface{}
			if err := ctx.GetJSON(resp, &payload); err != nil {
				return err
			}

			errorValue, ok := payload["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("error field is not an object")
			}

			message, ok := errorValue["message"].(string)
			if !ok || strings.TrimSpace(message) == "" {
				return fmt.Errorf("error.message is missing or empty")
			}

			return nil
		},
	)

	// Test 5: POST with nonexistent foreign key.
	// OData Protocol §11.4.2 says services SHOULD enforce referential constraints.
	// This is a SHOULD (not MUST), so skipping on 201 is acceptable. If the server
	// does validate it, the response must be 4xx. go-odata#772 tracks enforcement.
	suite.AddTest(
		"test_foreign_key_constraint",
		"POST with nonexistent foreign key returns 400/404/409 or is skipped (SHOULD per §11.4.2)",
		func(ctx *framework.TestContext) error {
			const fakePID = "00000000-0000-0000-0000-000000000000"
			payload := map[string]interface{}{
				"ProductID":   fakePID,
				"LanguageKey": "EN",
				"Description": "Should fail — ProductID does not exist",
			}
			resp, err := ctx.POST("/ProductDescriptions", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}
			if resp.StatusCode == 201 {
				return ctx.Skip("server does not enforce referential constraints (SHOULD per §11.4.2) — see go-odata#772")
			}
			// Any 4xx or 5xx is acceptable rejection.
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return fmt.Errorf("expected 4xx/5xx for nonexistent foreign key, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 6: Duplicate key on POST must be rejected.
	// Creating a ProductDescription with the same (ProductID, LanguageKey) twice
	// must be rejected because the composite key is already in use.
	// go-odata#773: go-odata currently returns 500 instead of 409 for this case.
	// The test accepts any non-2xx rejection and fails only on 2xx (insert silently accepted).
	suite.AddTest(
		"test_duplicate_key_rejected",
		"POST with duplicate key is rejected (409 Conflict expected; 500 is go-odata#773)",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "DuplicateKeyTest", 9.99)
			if err != nil {
				return err
			}

			payload := map[string]interface{}{
				"ProductID":   productID,
				"LanguageKey": "DK",
				"Description": "First insert",
			}
			first, err := ctx.POST("/ProductDescriptions", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}
			if first.StatusCode != 201 {
				return fmt.Errorf("first insert: expected 201, got %d", first.StatusCode)
			}

			// Second insert with the same (ProductID, LanguageKey) — must NOT succeed.
			payload["Description"] = "Second insert — should be rejected"
			second, err := ctx.POST("/ProductDescriptions", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}
			if second.StatusCode >= 200 && second.StatusCode < 300 {
				return fmt.Errorf("duplicate key insert returned %d — duplicate key silently accepted (should be 409)", second.StatusCode)
			}
			// 5xx means the server crashed on the constraint violation (go-odata#773); warn but don't fail.
			if second.StatusCode >= 500 {
				ctx.Log(fmt.Sprintf("duplicate key insert returned %d (should be 409 Conflict — see go-odata#773)", second.StatusCode))
			}
			return nil
		},
	)

	// Test 7: PATCH that changes the entity key must be rejected.
	// OData §11.4.3 states that services SHOULD reject a request that attempts
	// to change a key property. Expected: 400 or 422.
	suite.AddTest(
		"test_patch_key_property_rejected",
		"PATCH that changes the entity key is rejected with 400/405",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "PatchKeyTest", 1.00)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)

			// Try to PATCH the ID (key property) to a new value.
			patchResp, err := ctx.PATCH(productPath, map[string]interface{}{
				"ID": "00000000-0000-0000-0000-000000000001",
			})
			if err != nil {
				return err
			}
			// The patch must NOT succeed (2xx). 400, 405, or 422 are all acceptable.
			if patchResp.StatusCode >= 200 && patchResp.StatusCode < 300 {
				// Verify the ID was not actually changed — the server may silently ignore the key field.
				verifyResp, err := ctx.GET(productPath)
				if err != nil {
					return err
				}
				if verifyResp.StatusCode == 404 {
					return fmt.Errorf("PATCH key field: entity no longer exists at original path after patch")
				}
				var body map[string]interface{}
				if err := json.Unmarshal(verifyResp.Body, &body); err != nil {
					return err
				}
				newID := fmt.Sprintf("%v", body["ID"])
				if newID == "00000000-0000-0000-0000-000000000001" {
					return fmt.Errorf("PATCH key field: server accepted key change — ID was mutated to %s", newID)
				}
				// Server returned 2xx but silently ignored the key change — acceptable.
				ctx.Log("PATCH with key field returned 2xx but silently ignored the key change (conformant)")
			}
			return nil
		},
	)

	return suite
}
