package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// ComplexTypes creates the 5.2 Complex Types test suite
func ComplexTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.2 Complex Types",
		"Validates handling of complex (structured) types including nested properties, filtering, selecting, and operations.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ComplexType",
	)

	suite.AddTest(
		"test_complex_type_retrieval",
		"Retrieve entity with complex type property",
		func(ctx *framework.TestContext) error {
			// Find a product that has a ShippingAddress (not all products have one)
			prodPath, err := entityPathByFilter(ctx, "Products", "ShippingAddress/City ne null")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, "ShippingAddress") {
				// Verify it's a JSON object
				if !strings.Contains(body, `"ShippingAddress"`) {
					return framework.NewError("ShippingAddress is not a complex object")
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_complex_nested_property",
		"Access nested property of complex type",
		func(ctx *framework.TestContext) error {
			// Find a product that has a ShippingAddress (not all products have one)
			prodPath, err := entityPathByFilter(ctx, "Products", "ShippingAddress/City ne null")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/ShippingAddress/City")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			// Check that response contains a value field (actual city may vary by product)
			if !strings.Contains(body, `"value":`) {
				return framework.NewError("Nested property response missing expected value")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_filter_complex_property",
		"Filter by nested complex type property: every returned product has ShippingAddress/City='Seattle'",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingAddress/City eq 'Seattle'")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			for i, p := range items {
				addr, ok := p["ShippingAddress"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("entity %d: ShippingAddress missing or not an object (%T)", i, p["ShippingAddress"])
				}
				city, _ := addr["City"].(string)
				if city != "Seattle" {
					return fmt.Errorf("entity %d: ShippingAddress/City=%q, expected 'Seattle'", i, city)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_select_complex_property",
		"Select complex type property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=Name,ShippingAddress")
			if err != nil {
				return err
			}
			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("complex type selection returned %d; declared complex properties must be addressable", resp.StatusCode)
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_complex_null_value",
		"Retrieve entity with null complex type: returned entity has null ShippingAddress or null City",
		func(ctx *framework.TestContext) error {
			// Get product with null ShippingAddress using nested property filter
			resp, err := ctx.GET("/Products?$filter=ShippingAddress/City eq null&$top=1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return nil // no matching entities is acceptable
			}
			p := items[0]
			addr := p["ShippingAddress"]
			if addr != nil {
				// ShippingAddress is present — verify City is null
				addrMap, ok := addr.(map[string]interface{})
				if !ok {
					return fmt.Errorf("entity 0: ShippingAddress present but not an object (%T)", addr)
				}
				if city, exists := addrMap["City"]; exists && city != nil {
					return fmt.Errorf("entity 0: ShippingAddress/City=%v, expected null (filter was ShippingAddress/City eq null)", city)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_filter_null_complex",
		"Filter for null complex type value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingAddress eq null")
			if err != nil {
				return err
			}
			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("null filtering on complex properties returned %d; declared complex properties must be queryable", resp.StatusCode)
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_orderby_complex_property",
		"Order by complex type property: non-null City values appear in ascending order",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=ShippingAddress/City")
			if err != nil {
				return err
			}
			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("orderby on complex properties returned %d; declared complex properties must be queryable", resp.StatusCode)
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			// Collect non-null City values in result order and verify ascending.
			var prevCity string
			prevSet := false
			for i, p := range items {
				addr, ok := p["ShippingAddress"].(map[string]interface{})
				if !ok {
					continue // null ShippingAddress — skip (null ordering is implementation-defined)
				}
				city, ok := addr["City"].(string)
				if !ok {
					continue // null City — skip
				}
				if prevSet && city < prevCity {
					return fmt.Errorf("ShippingAddress/City ordering violated at index %d: %q < %q", i, city, prevCity)
				}
				prevCity = city
				prevSet = true
			}
			return nil
		},
	)

	suite.AddTest(
		"test_access_complex_type",
		"Access complex type property directly",
		func(ctx *framework.TestContext) error {
			// Find a product that has a ShippingAddress (not all products have one)
			prodPath, err := entityPathByFilter(ctx, "Products", "ShippingAddress/City ne null")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/ShippingAddress")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			// Check that response contains City field (actual value may vary by product)
			if !strings.Contains(body, `"City":`) {
				return framework.NewError("Complex property response missing expected City value")
			}

			return nil
		},
	)

	return suite
}
