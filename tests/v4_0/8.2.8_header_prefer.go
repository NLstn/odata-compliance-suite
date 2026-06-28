package v4_0

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderPrefer creates the 8.2.8 Prefer Header test suite
func HeaderPrefer() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.8 Prefer Header",
		"Tests Prefer header handling for client preferences like return=minimal and return=representation.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderPrefer",
	)

	suite.AddTest(
		"test_prefer_return_minimal",
		"Prefer: return=minimal",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/Products", map[string]interface{}{
				"Name":   "Prefer Test",
				"Price":  99.99,
				"Status": 1, // ProductStatusInStock
			}, framework.Header{
				Key:   "Prefer",
				Value: "return=minimal",
			})
			if err != nil {
				return err
			}
			// Per OData spec 11.4.2: POST MUST return 201 Created.
			// With Prefer: return=minimal the server MAY return 204 No Content instead.
			if resp.StatusCode != 201 && resp.StatusCode != 204 {
				return fmt.Errorf("expected 201 Created or 204 No Content for return=minimal, got %d", resp.StatusCode)
			}

			if resp.StatusCode == 204 {
				if len(resp.Body) > 0 {
					return framework.NewError("return=minimal honored but body is not empty")
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_prefer_return_representation",
		"Prefer: return=representation",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/Products", map[string]interface{}{
				"Name":   "Prefer Test 2",
				"Price":  99.99,
				"Status": 1, // ProductStatusInStock
			}, framework.Header{
				Key:   "Prefer",
				Value: "return=representation",
			})
			if err != nil {
				return err
			}
			// Per OData spec 11.4.2: POST MUST return 201 Created with entity representation.
			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}

			if len(resp.Body) == 0 {
				return framework.NewError("expected entity representation in response body")
			}
			if err := ctx.AssertJSONField(resp, "Name"); err != nil {
				return fmt.Errorf("response body should contain created entity: %v", err)
			}

			return nil
		},
	)

	return suite
}
