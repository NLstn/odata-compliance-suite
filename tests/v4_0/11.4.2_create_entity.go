package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// CreateEntity creates the 11.4.2 Create an Entity (POST) test suite
func CreateEntity() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.2 Create an Entity (POST)",
		"Tests entity creation according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_CreateanEntity",
	)

	// Track created IDs for cleanup
	var createdIDs []string

	// Test 1: POST should return 201 Created
	suite.AddTest(
		"test_post_returns_201",
		"POST entity returns 201 Created",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ComplianceTestProduct1", 99.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if resp.StatusCode != 201 {
				return fmt.Errorf("expected status 201, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if id, ok := result["ID"]; ok {
				parsedID, err := parseEntityID(id)
				if err != nil {
					return err
				}
				createdIDs = append(createdIDs, parsedID)
			}

			return nil
		},
	)

	// Test 2: Location header should be present in 201 response
	suite.AddTest(
		"test_location_header",
		"Location header present in 201 response",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ComplianceTestProduct2", 199.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if resp.StatusCode != 201 {
				return fmt.Errorf("expected status 201, got %d", resp.StatusCode)
			}

			location := resp.Headers.Get("Location")
			if location == "" {
				return fmt.Errorf("Location header missing")
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err == nil {
				if id, ok := result["ID"]; ok {
					parsedID, err := parseEntityID(id)
					if err != nil {
						return err
					}
					createdIDs = append(createdIDs, parsedID)
				}
			}

			return nil
		},
	)

	// Test 3: Created entity should be returned in response body
	suite.AddTest(
		"test_entity_in_response",
		"Created entity returned in response body",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ComplianceTestProduct3", 299.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if resp.StatusCode != 201 {
				return fmt.Errorf("expected status 201, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["ID"]; !ok {
				return fmt.Errorf("response missing 'ID' field")
			}

			name, ok := result["Name"].(string)
			if !ok || name != "ComplianceTestProduct3" {
				return fmt.Errorf("response missing or incorrect 'Name' field")
			}

			if id, ok := result["ID"]; ok {
				parsedID, err := parseEntityID(id)
				if err != nil {
					return err
				}
				createdIDs = append(createdIDs, parsedID)
			}

			return nil
		},
	)

	// Test 4: POST with Prefer: return=minimal — server SHOULD return 204 + OData-EntityId,
	// or MAY return 201 with the full representation if preference is not honoured.
	suite.AddTest(
		"test_prefer_minimal",
		"POST with Prefer: return=minimal returns 201 or 204",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ComplianceTestProduct4", 399.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Prefer", Value: "return=minimal"})
			if err != nil {
				return err
			}

			// Per §8.2.8.7: when return=minimal is honoured, respond with 204 No Content
			// and OData-EntityId.  When not honoured, respond with 201 + full entity.
			if resp.StatusCode != 201 && resp.StatusCode != 204 {
				return fmt.Errorf("expected 201 or 204 for return=minimal, got %d", resp.StatusCode)
			}

			if resp.StatusCode == 204 {
				// MUST carry OData-EntityId so the client can locate the created entity.
				entityId := resp.Headers.Get("OData-EntityId")
				if entityId == "" {
					return framework.NewError("204 response with return=minimal MUST include OData-EntityId (§8.2.8.7)")
				}
			}

			location := resp.Headers.Get("Location")
			if location == "" {
				return fmt.Errorf("Location header missing")
			}

			// Extract ID from location for cleanup
			if strings.Contains(location, "Products(") {
				start := strings.Index(location, "Products(")
				if start != -1 {
					idPart := location[start+len("Products("):]
					end := strings.Index(idPart, ")")
					if end != -1 {
						createdIDs = append(createdIDs, idPart[:end])
					}
				}
			}

			return nil
		},
	)

	// Test 5: OData-EntityId header MUST be present when server honours return=minimal.
	suite.AddTest(
		"test_odata_entityid_header",
		"OData-EntityId header present when return=minimal is honoured",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ComplianceTestProduct5", 499.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Prefer", Value: "return=minimal"})
			if err != nil {
				return err
			}

			// 204 = preference honoured: OData-EntityId is MANDATORY.
			// 201 = preference not honoured: OData-EntityId is optional (prefer to check).
			if resp.StatusCode == 204 {
				entityID := resp.Headers.Get("OData-EntityId")
				if entityID == "" {
					return framework.NewError("204 with return=minimal MUST include OData-EntityId header (§8.2.8.7)")
				}
				return nil
			}
			if resp.StatusCode == 201 {
				entityID := resp.Headers.Get("OData-EntityId")
				if entityID == "" {
					return fmt.Errorf("OData-EntityId header missing from 201 response")
				}
				return nil
			}
			return fmt.Errorf("expected 201 or 204 for return=minimal, got %d", resp.StatusCode)
		},
	)

	// Test 6: POST with a required field missing returns 400
	suite.AddTest(
		"test_post_missing_required_field",
		"POST with a missing required property returns 400",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ComplianceTestMissingName", 10.00)
			if err != nil {
				return err
			}
			delete(payload, "Name")

			resp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	// Test 7: POST with a wrong-typed property returns 400
	suite.AddTest(
		"test_post_wrong_type_rejected",
		"POST with a wrong-typed property (Price as a string) returns 400",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ComplianceTestWrongType", 10.00)
			if err != nil {
				return err
			}
			payload["Price"] = "not-a-number"

			resp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	// Test 8: a client-supplied value for the server-generated key is ignored
	suite.AddTest(
		"test_post_client_supplied_key_ignored",
		"POST with a client-supplied key value is ignored in favor of a server-generated key",
		func(ctx *framework.TestContext) error {
			const clientSuppliedID = "11111111-1111-1111-1111-111111111111"
			payload, err := buildProductPayload(ctx, "ComplianceTestClientKey", 10.00)
			if err != nil {
				return err
			}
			payload["ID"] = clientSuppliedID

			resp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}

			var created map[string]interface{}
			if err := ctx.GetJSON(resp, &created); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}
			createdID, err := parseEntityID(created["ID"])
			if err != nil {
				return err
			}
			if createdID == clientSuppliedID {
				return fmt.Errorf("server accepted the client-supplied key %q instead of generating its own", clientSuppliedID)
			}
			createdIDs = append(createdIDs, createdID)

			// The client-supplied ID must not exist as a distinct entity.
			verifyResp, err := ctx.GET(fmt.Sprintf("/Products(%s)", clientSuppliedID))
			if err != nil {
				return err
			}
			if verifyResp.StatusCode != 404 {
				return fmt.Errorf("expected the client-supplied key %q to not exist as an entity, got status %d", clientSuppliedID, verifyResp.StatusCode)
			}
			return nil
		},
	)

	return suite
}
