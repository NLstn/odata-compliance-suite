package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// Operations creates the 12.1 Operations (Actions and Functions) test suite
func Operations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"12.1 Operations",
		"Tests OData operations (actions and functions) including bound and unbound operations, parameter passing, and proper invocation syntax.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Operations",
	)

	// Helper function to get product path for each test
	// Note: Must refetch on each call because database is reseeded between tests
	getProductPath := func(ctx *framework.TestContext) (string, error) {
		return firstEntityPath(ctx, "Products")
	}

	// Test 1: Unbound function invocation
	suite.AddTest(
		"test_unbound_function",
		"Unbound function invocation",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/GetTopProducts()")
			if err != nil {
				return err
			}

			// If operation exists in metadata, must work properly
			if resp.StatusCode == 200 {
				// Validate response is valid OData collection or entity
				return ctx.AssertJSONField(resp, "value")
			}

			// Custom operations are optional OData features
			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("GetTopProducts() not defined in this service (optional feature)")
			}

			return fmt.Errorf("unexpected status code %d for unbound function", resp.StatusCode)
		},
	)

	// Test 2: Unbound function with parameters
	suite.AddTest(
		"test_unbound_function_parameters",
		"Unbound function with parameters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/GetTopProducts(count=3)")
			if err != nil {
				return err
			}

			// If function exists, parameters must be handled correctly
			if resp.StatusCode == 200 {
				return ctx.AssertJSONField(resp, "value")
			}

			// Custom operations are optional OData features
			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("GetTopProducts(count=3) not defined in this service (optional feature)")
			}

			return fmt.Errorf("unexpected status code %d for function with parameters", resp.StatusCode)
		},
	)

	// Test 3: Bound function on entity
	suite.AddTest(
		"test_bound_function",
		"Bound function on entity",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path + "/GetTotalPrice(taxRate=0.08)")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				var result map[string]interface{}
				if err := json.Unmarshal(resp.Body, &result); err != nil {
					return fmt.Errorf("bound function response is not valid JSON: %w", err)
				}
				if _, ok := result["value"]; !ok {
					return framework.NewError("Bound function response missing 'value' field")
				}
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("GetTotalPrice() not defined for Products (optional feature)")
			}

			return fmt.Errorf("unexpected status code %d for bound function", resp.StatusCode)
		},
	)

	// Test 4: Bound function on collection
	suite.AddTest(
		"test_bound_function_collection",
		"Bound function on collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/GetAveragePrice()")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				var result map[string]interface{}
				if err := json.Unmarshal(resp.Body, &result); err != nil {
					return fmt.Errorf("collection-bound function response is not valid JSON: %w", err)
				}
				if _, ok := result["value"]; !ok {
					return framework.NewError("Collection-bound function response missing 'value' field")
				}
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("GetAveragePrice() not defined for Products (optional feature)")
			}

			return fmt.Errorf("unexpected status code %d for collection-bound function", resp.StatusCode)
		},
	)

	// Test 5: Unbound action invocation
	suite.AddTest(
		"test_unbound_action",
		"Unbound action invocation",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{}

			resp, err := ctx.POST("/ResetProducts", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if resp.StatusCode == 204 {
				return nil
			}
			if resp.StatusCode == 200 {
				var body interface{}
				if err := json.Unmarshal(resp.Body, &body); err != nil {
					return fmt.Errorf("unbound action response is not valid JSON: %w", err)
				}
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("ResetProducts action not defined in this service (optional feature)")
			}

			return fmt.Errorf("unexpected status code %d for unbound action", resp.StatusCode)
		},
	)

	// Test 6: Bound action on entity
	suite.AddTest(
		"test_bound_action",
		"Bound action on entity",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			payload := map[string]interface{}{
				"percentage": 10,
			}

			resp, err := ctx.POST(path+"/ApplyDiscount", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if resp.StatusCode == 204 {
				return nil
			}
			if resp.StatusCode == 200 {
				var body interface{}
				if err := json.Unmarshal(resp.Body, &body); err != nil {
					return fmt.Errorf("bound action response is not valid JSON: %w", err)
				}
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("ApplyDiscount action not defined for Products (optional feature)")
			}

			return fmt.Errorf("unexpected status code %d for bound action", resp.StatusCode)
		},
	)

	// Test 7: Operation returns collection
	suite.AddTest(
		"test_operation_returns_collection",
		"Operation returns collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/GetTopProducts()")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				// Must return valid OData collection format
				if err := ctx.AssertJSONField(resp, "value"); err != nil {
					return fmt.Errorf("Operation returning collection must include 'value' array: %v", err)
				}
				return nil
			}

			// Custom operations are optional OData features
			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("GetTopProducts() not defined in this service (optional feature)")
			}

			return fmt.Errorf("unexpected status code %d", resp.StatusCode)
		},
	)

	return suite
}
