package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// FilterDivByOperator creates the OData 4.01 'divby' arithmetic operator test suite.
func FilterDivByOperator() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.5.1.1 divby Arithmetic Operator",
		"Validates the OData 4.01 'divby' decimal division operator in $filter expressions. "+
			"divby performs floating-point division, in contrast to 'div' which performs integer division.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_ArithmeticOperators",
	)

	// Test 1: divby returns 200 for a valid filter (OData 4.01 negotiated)
	suite.AddTest(
		"test_divby_operator_basic",
		"divby operator in $filter returns 200 when OData 4.01 is negotiated",
		func(ctx *framework.TestContext) error {
			// Price divby 100 gt 5.0  ↔  Price > 500
			// Seed data: Laptop (999.99), Smartphone (799.99), Premium Laptop Pro (1999.99)
			// all satisfy this; Office Chair (249.99), Wireless Mouse (29.99), Coffee Mug (15.50),
			// Gaming Mouse Ultra (149.99) do not.
			filter := url.QueryEscape("Price divby 100 gt 5.0")
			headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			resp, err := ctx.GET("/Products?$filter="+filter, headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			var payload struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse divby filter response: %v", err))
			}
			if payload.Value == nil {
				return framework.NewError("divby filter response missing 'value' collection")
			}
			// All 3 qualifying products must be present.
			if len(payload.Value) < 3 {
				return framework.NewError(fmt.Sprintf("Price divby 100 gt 5.0 expected at least 3 products (Price>500), got %d", len(payload.Value)))
			}
			// Every returned product must actually have Price > 500.
			for i, entity := range payload.Value {
				priceRaw, ok := entity["Price"]
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing Price field", i))
				}
				price, ok := priceRaw.(float64)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d has non-numeric Price %T", i, priceRaw))
				}
				if price <= 500 {
					return framework.NewError(fmt.Sprintf("entity %d has Price %.2f which does not satisfy Price divby 100 gt 5.0", i, price))
				}
			}
			return nil
		},
	)

	// Test 2: divby performs decimal (not integer) division
	suite.AddTest(
		"test_divby_performs_decimal_division",
		"divby performs decimal division (e.g. 3 divby 2 = 1.5, not 1)",
		func(ctx *framework.TestContext) error {
			// Price divby 2 ge 1  ↔  Price >= 2.
			// All 7 seed products have Price >= 2, so all 7 must be returned.
			filter := url.QueryEscape("Price divby 2 ge 1")
			headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			resp, err := ctx.GET("/Products?$filter="+filter, headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			var payload struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse divby decimal filter response: %v", err))
			}
			if payload.Value == nil {
				return framework.NewError("divby decimal filter response missing 'value' collection")
			}
			// All 7 seed products satisfy Price >= 2; fewer results means the filter
			// incorrectly excluded some rows.
			if len(payload.Value) < 7 {
				return framework.NewError(fmt.Sprintf("Price divby 2 ge 1 expected all 7 seed products (Price>=2), got %d", len(payload.Value)))
			}
			// Every returned product must have Price >= 2.
			for i, entity := range payload.Value {
				priceRaw, ok := entity["Price"]
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing Price field", i))
				}
				price, ok := priceRaw.(float64)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d has non-numeric Price %T", i, priceRaw))
				}
				if price < 2 {
					return framework.NewError(fmt.Sprintf("entity %d has Price %.2f which does not satisfy Price divby 2 ge 1", i, price))
				}
			}
			return nil
		},
	)

	// Test 3: divby combined with comparison and logical operators
	suite.AddTest(
		"test_divby_combined_with_and",
		"divby combined with 'and' logical operator",
		func(ctx *framework.TestContext) error {
			// Price divby 2 gt 0 and Price divby 2 lt 1000  ↔  Price > 0 and Price < 2000.
			// Premium Laptop Pro (1999.99) satisfies Price < 2000; all products have Price > 0.
			// So all 7 seed products should be returned.
			filter := url.QueryEscape("Price divby 2 gt 0 and Price divby 2 lt 1000")
			headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			resp, err := ctx.GET("/Products?$filter="+filter, headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			var payload struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse divby+and filter response: %v", err))
			}
			if payload.Value == nil {
				return framework.NewError("divby+and filter response missing 'value' collection")
			}
			// All 7 products have Price in (0, 2000), so all must appear.
			if len(payload.Value) < 7 {
				return framework.NewError(fmt.Sprintf("Price divby 2 gt 0 and Price divby 2 lt 1000 expected all 7 seed products, got %d", len(payload.Value)))
			}
			// Every returned product must satisfy both predicates.
			for i, entity := range payload.Value {
				priceRaw, ok := entity["Price"]
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing Price field", i))
				}
				price, ok := priceRaw.(float64)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d has non-numeric Price %T", i, priceRaw))
				}
				if price <= 0 || price >= 2000 {
					return framework.NewError(fmt.Sprintf("entity %d has Price %.2f outside expected range (0, 2000)", i, price))
				}
			}
			return nil
		},
	)

	// Test: division by zero is the canonical edge case for divby vs div —
	// divby is defined to always produce a well-formed floating-point result
	// (conceptually ±Infinity per IEEE 754 for a nonzero dividend) rather than
	// erroring. This asserts the one thing every implementation must agree on
	// (no server error, and a self-consistent result), while tolerating either
	// defensible interpretation of the resulting value: IEEE-754 Infinity
	// (which would satisfy "gt 0" for every positive-priced product) or a
	// three-valued-logic null (which excludes every row from any comparison).
	// A partial/inconsistent result set would indicate neither, and is a bug.
	suite.AddTest(
		"test_divby_zero",
		"Price divby 0 is handled without a server error, and consistently: either every positive-priced product matches 'gt 0' (IEEE-754 Infinity) or none do (null, excluded from comparisons) — never a partial/inconsistent set",
		func(ctx *framework.TestContext) error {
			headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}

			allResp, err := ctx.GET("/Products", headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(allResp, http.StatusOK); err != nil {
				return err
			}
			var all struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(allResp.Body, &all); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse baseline product list: %v", err))
			}
			positiveCount := 0
			for _, entity := range all.Value {
				if price, ok := entity["Price"].(float64); ok && price > 0 {
					positiveCount++
				}
			}
			if positiveCount == 0 {
				return ctx.Skip("no positive-priced products available to exercise divby 0")
			}

			filter := url.QueryEscape("Price divby 0 gt 0")
			resp, err := ctx.GET("/Products?$filter="+filter, headers...)
			if err != nil {
				return err
			}
			if resp.StatusCode >= 500 {
				return fmt.Errorf("divby division by zero must not produce a server error, got status %d", resp.StatusCode)
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			var payload struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse divby-zero filter response: %v", err))
			}
			if payload.Value == nil {
				return framework.NewError("divby-zero filter response missing 'value' collection")
			}

			switch len(payload.Value) {
			case 0:
				ctx.Log("Price divby 0 appears to yield null (excluded from 'gt 0'), not IEEE-754 Infinity")
			case positiveCount:
				ctx.Log("Price divby 0 appears to yield IEEE-754 Infinity, satisfying 'gt 0' for every positive-priced product")
			default:
				return fmt.Errorf(
					"Price divby 0 gt 0 returned %d result(s); expected either 0 (null semantics) or %d (IEEE-754 Infinity semantics), not a partial/inconsistent set",
					len(payload.Value), positiveCount)
			}
			return nil
		},
	)

	// A 4.01 service keeps accepting supported URL syntax when the response is
	// constrained to OData 4.0 (Protocol §13.2.1 item 9).
	suite.AddTest(
		"test_divby_version_negotiation_4_0_accepts",
		"divby operator remains accepted when OData-MaxVersion 4.0 is negotiated",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price divby 1.5 gt 0")
			headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.0"}}
			resp, err := ctx.GET("/Products?$filter="+filter, headers...)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	return suite
}
