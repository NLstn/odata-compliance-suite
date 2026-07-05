package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// CanonicalURL creates the 11.2.2 Canonical URL test suite
func CanonicalURL() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.2 Canonical URL",
		"Tests canonical URL representation according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_CanonicalURL",
	)

	// Test 1: Entity should have @odata.id with canonical URL
	suite.AddTest(
		"test_entity_odata_id",
		"Entity has @odata.id with canonical URL",
		func(ctx *framework.TestContext) error {
			// First get a product ID
			allResp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if allResp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", allResp.StatusCode)
			}

			var allResult map[string]interface{}
			if err := json.Unmarshal(allResp.Body, &allResult); err != nil {
				return fmt.Errorf("failed to parse products JSON: %w", err)
			}

			value, ok := allResult["value"].([]interface{})
			if !ok || len(value) == 0 {
				return fmt.Errorf("no products available")
			}

			firstItem, ok := value[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("first item is not an object")
			}

			productID := firstItem["ID"]

			// Get single entity
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)", productID))
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

			odataID, ok := result["@odata.id"].(string)
			if !ok {
				return fmt.Errorf("@odata.id field is missing or not a string")
			}

			if odataID == "" {
				return fmt.Errorf("@odata.id is empty")
			}

			return nil
		},
	)

	// Test 2: Canonical URL should be dereferenceable
	suite.AddTest(
		"test_odata_id_dereferenceable",
		"@odata.id URL is dereferenceable",
		func(ctx *framework.TestContext) error {
			// First get a product ID
			allResp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if allResp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", allResp.StatusCode)
			}

			var allResult map[string]interface{}
			if err := json.Unmarshal(allResp.Body, &allResult); err != nil {
				return fmt.Errorf("failed to parse products JSON: %w", err)
			}

			value, ok := allResult["value"].([]interface{})
			if !ok || len(value) == 0 {
				return fmt.Errorf("no products available")
			}

			firstItem, ok := value[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("first item is not an object")
			}

			productID := firstItem["ID"]

			// Get single entity
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)", productID))
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

			odataID, ok := result["@odata.id"].(string)
			if !ok || odataID == "" {
				return fmt.Errorf("@odata.id field is missing")
			}

			// Extract the path from the URL (remove base URL if present)
			path := odataID
			if strings.HasPrefix(odataID, "http://") || strings.HasPrefix(odataID, "https://") {
				// It's an absolute URL, extract just the path
				parts := strings.SplitN(odataID, "/", 4)
				if len(parts) >= 4 {
					path = "/" + parts[3]
				}
			}

			// Try to dereference it
			derefResp, err := ctx.GET(path)
			if err != nil {
				return fmt.Errorf("failed to dereference @odata.id: %w", err)
			}
			if derefResp.StatusCode != 200 {
				return fmt.Errorf("@odata.id is not dereferenceable, got status %d", derefResp.StatusCode)
			}

			return nil
		},
	)

	// Test 3: Collection should have @odata.id for each entity
	suite.AddTest(
		"test_collection_odata_ids",
		"Each entity in collection has @odata.id",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3")
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

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			entityCount := len(value)
			if entityCount == 0 {
				return fmt.Errorf("no entities in collection")
			}

			odataIDCount := 0
			for i, v := range value {
				item, ok := v.(map[string]interface{})
				if !ok {
					return fmt.Errorf("item %d is not an object", i)
				}
				if _, ok := item["@odata.id"]; ok {
					odataIDCount++
				}
			}

			if odataIDCount != entityCount {
				return fmt.Errorf("found %d entities but %d @odata.id fields", entityCount, odataIDCount)
			}

			return nil
		},
	)

	// Test 4: Canonical URL format should match entity set and key
	suite.AddTest(
		"test_canonical_url_format",
		"Canonical URL format matches entity set and key pattern",
		func(ctx *framework.TestContext) error {
			// First get a product ID
			allResp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if allResp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", allResp.StatusCode)
			}

			var allResult map[string]interface{}
			if err := json.Unmarshal(allResp.Body, &allResult); err != nil {
				return fmt.Errorf("failed to parse products JSON: %w", err)
			}

			value, ok := allResult["value"].([]interface{})
			if !ok || len(value) == 0 {
				return fmt.Errorf("no products available")
			}

			firstItem, ok := value[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("first item is not an object")
			}

			productID := firstItem["ID"]

			// Get single entity
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)", productID))
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

			odataID, ok := result["@odata.id"].(string)
			if !ok || odataID == "" {
				return fmt.Errorf("@odata.id field is missing")
			}

			// Verify it contains "Products(" pattern
			if !strings.Contains(odataID, "Products(") {
				return fmt.Errorf("@odata.id format does not match expected pattern (missing 'Products('): %s", odataID)
			}

			return nil
		},
	)

	// Test 5: /$entity?@odata.id=<url> — entity dereference via canonical URL
	suite.AddTest(
		"test_entity_dereference_via_odata_id",
		"/$entity?@odata.id=<url> returns the referenced entity (OData §11.5.4.1)",
		func(ctx *framework.TestContext) error {
			// Step 1: fetch a product and extract its @odata.id.
			resp, err := ctx.GET("/Products?$top=1&$select=ID")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("GET /Products?$top=1 failed: status %d", resp.StatusCode)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("failed to parse products: %w", err)
			}
			value, ok := body["value"].([]interface{})
			if !ok || len(value) == 0 {
				return framework.NewError("no products available for entity-dereference test")
			}
			product, ok := value[0].(map[string]interface{})
			if !ok {
				return framework.NewError("first product is not a JSON object")
			}
			odataID, ok := product["@odata.id"].(string)
			if !ok || odataID == "" {
				return fmt.Errorf("product missing @odata.id (needed for entity-dereference test)")
			}

			// Step 2: dereference via /$entity?@odata.id=<url>.
			// The @odata.id may be absolute or relative; percent-encode it so colons and
			// slashes in an absolute URL don't corrupt the query string.
			derefResp, err := ctx.GET("/$entity?@odata.id=" + url.QueryEscape(odataID))
			if err != nil {
				return err
			}
			switch derefResp.StatusCode {
			case 200:
				// Verify the returned entity has an ID field.
				var entity map[string]interface{}
				if err := json.Unmarshal(derefResp.Body, &entity); err != nil {
					return fmt.Errorf("/$entity response is not valid JSON: %w", err)
				}
				if _, ok := entity["ID"]; !ok {
					return framework.NewError("/$entity response missing 'ID' field")
				}
				return nil
			case 404, 501:
				// Optional feature — server may not support /$entity endpoint.
				return ctx.Skip("/$entity endpoint not implemented (404/501)")
			default:
				return fmt.Errorf("/$entity?@odata.id=... returned unexpected status %d", derefResp.StatusCode)
			}
		},
	)

	return suite
}
