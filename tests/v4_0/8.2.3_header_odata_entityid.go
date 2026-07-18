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

	// dereferenceEntityID resolves an OData-EntityId (or Location) header value
	// and confirms it identifies wantID, rather than merely containing the
	// entity-set name as a substring.
	dereferenceEntityID := func(ctx *framework.TestContext, headerName, headerValue, wantID string) error {
		if headerValue == "" {
			return fmt.Errorf("%s header missing", headerName)
		}
		if !strings.Contains(headerValue, "Products") {
			return fmt.Errorf("%s=%q should contain entity set name 'Products'", headerName, headerValue)
		}
		path := strings.TrimPrefix(headerValue, ctx.ServerURL())
		if path == headerValue {
			return fmt.Errorf("%s %q is not rooted at this service's base URL %q", headerName, headerValue, ctx.ServerURL())
		}
		getResp, err := ctx.GET(path)
		if err != nil {
			return err
		}
		if err := ctx.AssertStatusCode(getResp, 200); err != nil {
			return fmt.Errorf("%s does not resolve to the created entity: %w", headerName, err)
		}
		var fetched map[string]interface{}
		if err := ctx.GetJSON(getResp, &fetched); err != nil {
			return fmt.Errorf("failed to parse entity fetched via %s: %w", headerName, err)
		}
		fetchedID, _ := fetched["ID"].(string)
		if fetchedID != wantID {
			return fmt.Errorf("%s resolved to a different entity: created ID=%q, fetched ID=%q", headerName, wantID, fetchedID)
		}
		return nil
	}

	suite.AddTest(
		"test_odata_entityid_header",
		"OData-EntityId header is present on 201 and resolves to the created entity",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "EntityId Test", 99.99)
			if err != nil {
				return err
			}
			resp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}
			// A plain create (no Prefer header) must succeed with 201; a server
			// returning anything else here is a regression, not something to
			// silently skip past.
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

			entityId := resp.Headers.Get("OData-EntityId")
			return dereferenceEntityID(ctx, "OData-EntityId", entityId, createdID)
		},
	)

	suite.AddTest(
		"test_odata_entityid_header_return_minimal",
		"OData-EntityId header is present and resolves to the created entity when Prefer: return=minimal is honored",
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
			if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 204 {
				return fmt.Errorf("expected a successful creation status (200/201/204), got %d", resp.StatusCode)
			}

			// Regardless of whether return=minimal was honored (a service may
			// signal that either via 204 No Content or via 201 Created with an
			// empty body plus Preference-Applied — see 8.2.8_header_prefer.go
			// for the same distinction confirmed against this reference
			// server), OData-EntityId must be present per §8.2.8.7.
			entityId := resp.Headers.Get("OData-EntityId")
			if entityId == "" {
				return framework.NewError("OData-EntityId header must be present on a successful create response (§8.2.8.7), regardless of whether return=minimal was honored")
			}

			// When the preference was honored, the body is empty, so the only
			// way to know which entity was created is the header itself —
			// dereference it and require it to actually resolve.
			if !strings.Contains(entityId, "Products") {
				return fmt.Errorf("OData-EntityId=%q should reference the Products entity set", entityId)
			}
			path := strings.TrimPrefix(entityId, ctx.ServerURL())
			if path == entityId {
				return fmt.Errorf("OData-EntityId %q is not rooted at this service's base URL %q", entityId, ctx.ServerURL())
			}
			getResp, err := ctx.GET(path)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(getResp, 200)
		},
	)

	return suite
}
