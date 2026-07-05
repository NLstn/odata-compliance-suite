package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderODataEntityId creates the 8.2.3 OData-EntityId Header test suite
func HeaderODataEntityId() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.3 OData-EntityId Header",
		"Tests OData-EntityId header in responses for created entities.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderODataEntityId",
	)

	suite.AddTest(
		"test_odata_entityid_header",
		"OData-EntityId header contains entity URL on 201",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "EntityId Test", 99.99)
			if err != nil {
				return err
			}
			resp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if resp.StatusCode != 201 && resp.StatusCode != 200 {
				return nil
			}

			entityId := resp.Headers.Get("OData-EntityId")
			if entityId != "" && !strings.Contains(entityId, "Products") {
				return fmt.Errorf("OData-EntityId=%q should contain entity set name 'Products'", entityId)
			}
			return nil
		},
	)

	suite.AddTest(
		"test_odata_entityid_header_return_minimal",
		"OData-EntityId header MUST be present when Prefer: return=minimal is honored with 204",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "EntityId Minimal Test", 55.00)
			if err != nil {
				return err
			}
			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Prefer", Value: "return=minimal"})
			if err != nil {
				return err
			}

			// Per OData §8.2.8.7: when return=minimal is applied and server returns 204,
			// OData-EntityId MUST be included so the client can identify the created entity.
			if resp.StatusCode == 204 {
				entityId := resp.Headers.Get("OData-EntityId")
				if entityId == "" {
					return framework.NewError("204 response with return=minimal must include OData-EntityId header (§8.2.8.7)")
				}
				if !strings.Contains(entityId, "Products") {
					return fmt.Errorf("OData-EntityId=%q should reference the Products entity set", entityId)
				}
			}
			return nil
		},
	)

	return suite
}
