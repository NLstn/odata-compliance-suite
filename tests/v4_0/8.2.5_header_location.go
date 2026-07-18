package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderLocation creates the 8.2.5 Location Header test suite
func HeaderLocation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.5 Location Header",
		"Tests Location header in responses for created entities.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderLocation",
	)

	suite.AddTest(
		"test_location_header",
		"Location header for created entity resolves to that same entity",
		func(ctx *framework.TestContext) error {
			// Fetch a valid Category ID first (Products require a CategoryID that is a UUID)
			categoryID, err := firstEntityID(ctx, "Categories")
			if err != nil {
				return fmt.Errorf("failed to fetch Category ID: %w", err)
			}

			createResp, err := ctx.POST("/Products", map[string]interface{}{
				"Name":       "Location Test",
				"Price":      99.99,
				"CategoryID": categoryID,
			})
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 && createResp.StatusCode != 200 {
				// Skip if creation fails due to validation
				if createResp.StatusCode == 400 {
					return framework.NewError("Entity creation validation error (likely schema mismatch)")
				}
				return framework.NewError("Entity creation not supported or failed")
			}

			// OData v4 requires Location header in 201 responses
			location := createResp.Headers.Get("Location")
			if location == "" {
				return fmt.Errorf("Location header is required for 201 responses per OData v4 spec")
			}
			if !strings.Contains(location, "Products") {
				return fmt.Errorf("Location header should contain entity URL, got: %s", location)
			}

			var created map[string]interface{}
			if err := ctx.GetJSON(createResp, &created); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}
			createdID, _ := created["ID"].(string)

			// Dereference the Location header — a URL that merely contains the
			// substring "Products" (even garbage text) would otherwise pass;
			// it must actually resolve back to the entity that was created.
			path := strings.TrimPrefix(location, ctx.ServerURL())
			if path == location {
				return fmt.Errorf("Location header %q is not rooted at this service's base URL %q", location, ctx.ServerURL())
			}

			getResp, err := ctx.GET(path)
			if err != nil {
				return fmt.Errorf("failed to dereference Location header: %w", err)
			}
			if err := ctx.AssertStatusCode(getResp, 200); err != nil {
				return fmt.Errorf("Location header does not resolve to the created entity: %w", err)
			}

			var fetched map[string]interface{}
			if err := ctx.GetJSON(getResp, &fetched); err != nil {
				return fmt.Errorf("failed to parse entity fetched via Location header: %w", err)
			}
			fetchedID, _ := fetched["ID"].(string)
			if createdID != "" && fetchedID != createdID {
				return fmt.Errorf("Location header resolved to a different entity: created ID=%q, fetched ID=%q", createdID, fetchedID)
			}

			return nil
		},
	)

	return suite
}
