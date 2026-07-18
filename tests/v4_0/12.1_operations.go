package v4_0

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

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

	// topNProductNamesByPrice returns the top n product Names sorted by Price
	// descending, computed independently from a full fetch, as the oracle for
	// GetTopProducts.
	topNProductNamesByPrice := func(ctx *framework.TestContext, n int) ([]string, error) {
		all, err := fetchAllProducts(ctx)
		if err != nil {
			return nil, err
		}
		sort.Slice(all, func(i, j int) bool {
			pi, _ := productFloat(all[i], "Price")
			pj, _ := productFloat(all[j], "Price")
			return pi > pj
		})
		if n > len(all) {
			n = len(all)
		}
		names := make([]string, 0, n)
		for _, p := range all[:n] {
			names = append(names, productString(p, "Name"))
		}
		return names, nil
	}

	// Test 1: Unbound function invocation
	suite.AddTest(
		"test_unbound_function",
		"Unbound function GetTopProducts() returns all products sorted by Price descending",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/GetTopProducts()")
			if err != nil {
				return err
			}
			if resp.StatusCode == 404 {
				return framework.NewError("Unbound function not defined in service")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return fmt.Errorf("unexpected status for unbound function: %w", err)
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			all, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}
			if len(items) != len(all) {
				return fmt.Errorf("GetTopProducts() with no count returned %d products, expected all %d", len(items), len(all))
			}
			var prev float64
			for i, item := range items {
				price, ok := productFloat(item, "Price")
				if !ok {
					return fmt.Errorf("item %d missing numeric Price", i)
				}
				if i > 0 && price > prev {
					return fmt.Errorf("GetTopProducts() not sorted descending by Price: item %d (Price=%v) follows item %d (Price=%v)", i, price, i-1, prev)
				}
				prev = price
			}
			return nil
		},
	)

	// Test 2: Unbound function with parameters
	suite.AddTest(
		"test_unbound_function_parameters",
		"GetTopProducts(count=3) returns exactly the 3 highest-priced products",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/GetTopProducts(count=3)")
			if err != nil {
				return err
			}
			if resp.StatusCode == 404 {
				return framework.NewError("Unbound function with parameters not defined in service")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return fmt.Errorf("unexpected status for function with parameters: %w", err)
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			expectedNames, err := topNProductNamesByPrice(ctx, 3)
			if err != nil {
				return err
			}
			if len(items) != len(expectedNames) {
				return fmt.Errorf("GetTopProducts(count=3) returned %d products, expected %d", len(items), len(expectedNames))
			}
			for i, item := range items {
				name := productString(item, "Name")
				if name != expectedNames[i] {
					return fmt.Errorf("GetTopProducts(count=3) item %d = %q, expected %q (by Price descending)", i, name, expectedNames[i])
				}
			}
			return nil
		},
	)

	// Test 3: Bound function on entity
	suite.AddTest(
		"test_bound_function",
		"GetTotalPrice(taxRate=0.08) returns Price * 1.08 for the target entity",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			getResp, err := ctx.GET(path)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(getResp, 200); err != nil {
				return err
			}
			var entity map[string]interface{}
			if err := ctx.GetJSON(getResp, &entity); err != nil {
				return err
			}
			price, ok := productFloat(entity, "Price")
			if !ok {
				return framework.NewError("entity missing numeric Price")
			}

			resp, err := ctx.GET(path + "/GetTotalPrice(taxRate=0.08)")
			if err != nil {
				return err
			}
			if resp.StatusCode == 404 {
				return framework.NewError("Bound function not defined for this entity type")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return fmt.Errorf("unexpected status for bound function: %w", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("bound function response is not valid JSON: %w", err)
			}
			value, ok := result["value"].(float64)
			if !ok {
				return framework.NewError("Bound function response missing numeric 'value' field")
			}
			expected := price * 1.08
			if math.Abs(value-expected) > 0.001 {
				return fmt.Errorf("GetTotalPrice(taxRate=0.08) = %v, expected Price(%v) * 1.08 = %v", value, price, expected)
			}
			return nil
		},
	)

	// Test 4: Bound function on collection
	suite.AddTest(
		"test_bound_function_collection",
		"GetAveragePrice() returns the actual average Price across all products",
		func(ctx *framework.TestContext) error {
			all, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}
			var sum float64
			count := 0
			for _, p := range all {
				price, ok := productFloat(p, "Price")
				if !ok {
					continue
				}
				sum += price
				count++
			}
			if count == 0 {
				return ctx.Skip("no products with a numeric Price available to compute an average")
			}
			expected := sum / float64(count)

			resp, err := ctx.GET("/Products/GetAveragePrice()")
			if err != nil {
				return err
			}
			if resp.StatusCode == 404 {
				return framework.NewError("Collection-bound function not defined for Products")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return fmt.Errorf("unexpected status for collection-bound function: %w", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("collection-bound function response is not valid JSON: %w", err)
			}
			value, ok := result["value"].(float64)
			if !ok {
				return framework.NewError("Collection-bound function response missing numeric 'value' field")
			}
			if math.Abs(value-expected) > 0.001 {
				return fmt.Errorf("GetAveragePrice() = %v, expected average of %d product(s) = %v", value, count, expected)
			}
			return nil
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

			if resp.StatusCode == 404 {
				return framework.NewError("Unbound action not defined in service")
			}

			return fmt.Errorf("Unexpected status code %d for unbound action", resp.StatusCode)
		},
	)

	// Test 6: Bound action on entity
	suite.AddTest(
		"test_bound_action",
		"ApplyDiscount(percentage=10) reduces the entity's Price by exactly 10%, persisted",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}

			getResp, err := ctx.GET(path)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(getResp, 200); err != nil {
				return err
			}
			var before map[string]interface{}
			if err := ctx.GetJSON(getResp, &before); err != nil {
				return err
			}
			originalPrice, ok := productFloat(before, "Price")
			if !ok {
				return framework.NewError("entity missing numeric Price")
			}

			payload := map[string]interface{}{
				"percentage": 10,
			}
			resp, err := ctx.POST(path+"/ApplyDiscount", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return framework.NewError("Bound action not defined for this entity type")
			}
			if resp.StatusCode != 200 && resp.StatusCode != 204 {
				return fmt.Errorf("Unexpected status code %d for bound action", resp.StatusCode)
			}
			if resp.StatusCode == 200 {
				var body interface{}
				if err := json.Unmarshal(resp.Body, &body); err != nil {
					return fmt.Errorf("bound action response is not valid JSON: %w", err)
				}
			}

			// Verify the discount actually took effect, not just that the
			// action's own response status was successful.
			verifyResp, err := ctx.GET(path)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(verifyResp, 200); err != nil {
				return err
			}
			var after map[string]interface{}
			if err := ctx.GetJSON(verifyResp, &after); err != nil {
				return err
			}
			newPrice, ok := productFloat(after, "Price")
			if !ok {
				return framework.NewError("entity missing numeric Price after ApplyDiscount")
			}
			expected := originalPrice * 0.9
			if math.Abs(newPrice-expected) > 0.001 {
				return fmt.Errorf("after ApplyDiscount(percentage=10), Price = %v, expected %v * 0.9 = %v", newPrice, originalPrice, expected)
			}
			return nil
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

			// 404 indicates operation not defined
			if resp.StatusCode == 404 {
				return framework.NewError("Operation not defined in service")
			}

			return fmt.Errorf("Unexpected status code %d", resp.StatusCode)
		},
	)

	// Test 8: invoking a function via POST is rejected
	suite.AddTest(
		"test_function_invoked_via_wrong_verb_rejected",
		"Invoking a function (read-only) via POST is rejected",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/GetTopProducts()", map[string]interface{}{}, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}
			if resp.StatusCode < 400 || resp.StatusCode >= 500 {
				return fmt.Errorf("expected a 4xx rejection for POST to a function, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 9: invoking an action via GET is rejected
	suite.AddTest(
		"test_action_invoked_via_wrong_verb_rejected",
		"Invoking an action (has side effects) via GET is rejected",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/ResetProducts")
			if err != nil {
				return err
			}
			if resp.StatusCode < 400 || resp.StatusCode >= 500 {
				return fmt.Errorf("expected a 4xx rejection for GET on an action, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 10: bound function invoked with a missing required parameter
	suite.AddTest(
		"test_bound_function_missing_parameter_rejected",
		"GetTotalPrice() invoked without its required taxRate parameter is rejected",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path + "/GetTotalPrice()")
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	// Test 11: bound function invoked with a wrong-typed parameter
	suite.AddTest(
		"test_bound_function_wrong_type_parameter_rejected",
		"GetTotalPrice(taxRate='not-a-number') is rejected",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path + "/GetTotalPrice(taxRate='not-a-number')")
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	return suite
}
