package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// EntityReferences creates the 11.2.15 Entity References ($ref) test suite
func EntityReferences() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.15 Entity References",
		"Tests $ref for retrieving and manipulating entity references instead of the full entity representation.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_AddressingReferences",
	)

	invalidProductPath := nonExistingEntityPath("Products")

	// Test 1: Get reference to single entity
	suite.AddTest(
		"test_entity_ref_single",
		"Get reference to single entity",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/$ref")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// Should return @odata.id with the entity reference
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["@odata.id"]; !ok {
				return fmt.Errorf("response missing @odata.id")
			}

			return nil
		},
	)

	// Test 2: Get reference to collection
	suite.AddTest(
		"test_entity_ref_collection",
		"Get references to entity collection: each element must include @odata.id",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/$ref")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			refs, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing value array")
			}

			// Per OData v4 §11.4.6.1, each reference element MUST include @odata.id.
			for i, r := range refs {
				ref, ok := r.(map[string]interface{})
				if !ok {
					return fmt.Errorf("reference element %d is not an object", i)
				}
				id, ok := ref["@odata.id"].(string)
				if !ok || id == "" {
					return fmt.Errorf("reference element %d missing or empty @odata.id", i)
				}
			}
			return nil
		},
	)

	// Test 3: Reference should contain @odata.context
	suite.AddTest(
		"test_ref_has_context",
		"Reference contains @odata.context",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/$ref")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			bodyStr := string(resp.Body)
			if !strings.Contains(bodyStr, "@odata.context") {
				return fmt.Errorf("reference missing @odata.context")
			}

			return nil
		},
	)

	// Test 4: Reference should NOT contain entity properties
	suite.AddTest(
		"test_ref_no_properties",
		"Reference does not contain entity properties",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/$ref")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			bodyStr := string(resp.Body)
			// Should not contain entity properties like Name, Price
			if strings.Contains(bodyStr, `"Name"`) || strings.Contains(bodyStr, `"Price"`) {
				return fmt.Errorf("reference contains entity properties")
			}

			return nil
		},
	)

	// Test 5: $ref with $filter on collection
	suite.AddTest(
		"test_ref_with_filter",
		"$ref with $filter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/$ref?$filter=Price gt 50")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				var result map[string]interface{}
				if err := json.Unmarshal(resp.Body, &result); err != nil {
					return fmt.Errorf("$ref?$filter returned non-JSON: %w", err)
				}
				if _, ok := result["value"]; !ok {
					return fmt.Errorf("$ref?$filter response missing 'value' array")
				}
				return nil
			}
			if resp.StatusCode == 400 {
				return framework.NewError("$ref with $filter not supported by service")
			}

			return fmt.Errorf("expected status 200, got %d", resp.StatusCode)

		},
	)

	// Test 6: $ref with $top on collection
	suite.AddTest(
		"test_ref_with_top",
		"$ref with $top",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/$ref?$top=3")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("$ref?$top returned non-JSON: %w", err)
			}
			refs, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("$ref?$top response missing 'value' array")
			}
			if len(refs) > 3 {
				return fmt.Errorf("$top=3 returned %d references (expected ≤ 3)", len(refs))
			}
			return nil
		},
	)

	// Test 7: $ref with $skip on collection
	suite.AddTest(
		"test_ref_with_skip",
		"$ref with $skip",
		func(ctx *framework.TestContext) error {
			allResp, err := ctx.GET("/Products/$ref")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(allResp, 200); err != nil {
				return err
			}
			var allResult map[string]interface{}
			if err := json.Unmarshal(allResp.Body, &allResult); err != nil {
				return fmt.Errorf("$ref returned non-JSON: %w", err)
			}
			allRefs, _ := allResult["value"].([]interface{})

			resp, err := ctx.GET("/Products/$ref?$skip=2")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("$ref?$skip returned non-JSON: %w", err)
			}
			skippedRefs, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("$ref?$skip response missing 'value' array")
			}
			expectedCount := len(allRefs) - 2
			if expectedCount < 0 {
				expectedCount = 0
			}
			if len(skippedRefs) != expectedCount {
				return fmt.Errorf("$skip=2 from %d total: expected %d references, got %d", len(allRefs), expectedCount, len(skippedRefs))
			}
			return nil
		},
	)

	// Test 8: $ref with $orderby on collection
	suite.AddTest(
		"test_ref_with_orderby",
		"$ref with $orderby",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/$ref?$orderby=ID")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("$ref?$orderby returned non-JSON: %w", err)
			}
			if _, ok := result["value"]; !ok {
				return fmt.Errorf("$ref?$orderby response missing 'value' array")
			}
			return nil
		},
	)

	// Test 9: $ref should not support $expand
	suite.AddTest(
		"test_ref_no_expand",
		"$ref rejects $expand (should return 400)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/$ref?$expand=Category")
			if err != nil {
				return err
			}

			// Should return 400 Bad Request as $expand is not valid with $ref
			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400 for invalid query option, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 10: $ref should not support $select
	suite.AddTest(
		"test_ref_no_select",
		"$ref rejects $select (should return 400)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/$ref?$select=Name")
			if err != nil {
				return err
			}

			// Should return 400 Bad Request as $select is not valid with $ref
			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400 for invalid query option, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 11: $ref on non-existent entity returns 404
	suite.AddTest(
		"test_ref_not_found",
		"$ref on non-existent entity returns 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath + "/$ref")
			if err != nil {
				return err
			}

			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 12: $ref with $count
	suite.AddTest(
		"test_ref_with_count",
		"$ref with $count",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/$ref?$count=true")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			bodyStr := string(resp.Body)
			if !strings.Contains(bodyStr, "@odata.count") {
				return fmt.Errorf("response missing @odata.count")
			}

			return nil
		},
	)

	return suite
}
