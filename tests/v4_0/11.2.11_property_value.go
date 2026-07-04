package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// PropertyValue creates the 11.2.11 Property $value test suite
func PropertyValue() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.11 Property $value",
		"Tests accessing raw property values using the $value path segment.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_AddressingIndividualPropertiesofanEnt",
	)

	// Test 1: Access primitive property $value and verify it matches the property
	suite.AddTest(
		"test_property_value",
		"Access primitive property raw value matches the property value",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			// Fetch the entity to learn the actual Name value.
			entResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(entResp, 200); err != nil {
				return err
			}
			var entity map[string]interface{}
			if err := ctx.GetJSON(entResp, &entity); err != nil {
				return err
			}
			name, ok := entity["Name"].(string)
			if !ok {
				return fmt.Errorf("entity Name is missing or not a string")
			}

			resp, err := ctx.GET(productPath + "/Name/$value")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// $value returns the raw value (no JSON wrapper, no quotes) and must
			// equal the property's actual value.
			got := strings.TrimSpace(string(resp.Body))
			if got != name {
				return fmt.Errorf("$value returned %q but the Name property is %q", got, name)
			}

			return nil
		},
	)

	// Test 2: $value returns text/plain Content-Type for a string property
	suite.AddTest(
		"test_value_content_type",
		"$value returns a text/plain Content-Type for a string property",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/Name/$value")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// The raw value of an Edm.String property is served as text/plain
			// (OData Part 1 §11.2.3.1: $value uses the media type of the property).
			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "text/plain") {
				return fmt.Errorf("expected Content-Type text/plain for a string $value, got %q", contentType)
			}

			return nil
		},
	)

	// Test 3: $value on numeric property
	suite.AddTest(
		"test_numeric_value",
		"$value works on numeric properties",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/Price/$value")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 4: $value on non-existent property returns 404
	suite.AddTest(
		"test_value_nonexistent_property",
		"$value on non-existent property returns 404",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/NonExistent/$value")
			if err != nil {
				return err
			}

			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 5: $value on null property
	suite.AddTest(
		"test_value_null_property",
		"$value on nullable property returns 204 or 200",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/CategoryID/$value")
			if err != nil {
				return err
			}

			// Should return 204 No Content for null, or 200 with value
			if resp.StatusCode != 200 && resp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 6: $value on collection property returns error
	suite.AddTest(
		"test_value_collection_error",
		"$value on collection returns error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/$value")
			if err != nil {
				return err
			}

			// Should return 400 or 404
			if resp.StatusCode != 400 && resp.StatusCode != 404 {
				return fmt.Errorf("expected status 400 or 404, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 7: $value returns raw value without quotes
	suite.AddTest(
		"test_value_raw_format",
		"$value returns raw value without JSON wrapper",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/Status/$value")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// Should return raw value like "1" not JSON like {"value": 1}
			bodyStr := string(resp.Body)
			if strings.Contains(bodyStr, `"value"`) {
				return fmt.Errorf("response appears to be JSON-wrapped")
			}

			return nil
		},
	)

	// Test 8: $value with Accept header
	suite.AddTest(
		"test_value_accept_header",
		"$value respects Accept header",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GETWithHeaders(productPath+"/Name/$value", map[string]string{
				"Accept": "text/plain",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 9: $value on complex type property
	suite.AddTest(
		"test_value_complex_type",
		"$value on complex type must return 400",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			// OData Protocol §11.2.11: $value is only defined for primitive properties.
			// Appending /$value to a complex-type structured property is a client error
			// and the server MUST return 400 Bad Request.
			resp, err := ctx.GET(productPath + "/Dimensions/$value")
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400 for $value on complex type, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 10: $value without trailing slash
	suite.AddTest(
		"test_value_no_trailing_slash",
		"$value works without trailing slash",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/Name/$value")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	return suite
}
