package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// ParameterAliases creates the 11.2.5.8 Parameter Aliases test suite
func ParameterAliases() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.8 Parameter Aliases",
		"Tests parameter alias support in system query options.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ParameterAliases",
	)

	suite.AddTest(
		"test_filter_parameter_alias",
		"$filter supports parameter aliases",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt @p&@p=10")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %v", err)
			}

			value, ok := result["value"].([]interface{})
			if !ok {
				return framework.NewError("response missing value array")
			}

			// Success: parameter alias was accepted by server (status 200)
			// The actual filter results depend on data, so we don't assert on count
			_ = value // Acknowledge we have the value
			return nil
		},
	)

	suite.AddTest(
		"test_top_parameter_alias",
		"$top supports parameter aliases",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=@t&@t=1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %v", err)
			}

			value, ok := result["value"].([]interface{})
			if !ok {
				return framework.NewError("response missing value array")
			}

			// Verify that $top=@t&@t=1 respected the limit
			if len(value) > 1 {
				return fmt.Errorf("expected at most 1 product with $top=1, got %d", len(value))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_string_filter_parameter_alias",
		"$filter supports string-valued parameter aliases",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name eq @name&@name=" + url.QueryEscape("'Laptop'"))
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
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Name eq @name", func(entity map[string]interface{}) (bool, string) {
				name, ok := entity["Name"].(string)
				if !ok {
					return false, "Name field is missing or not a string"
				}
				if name != "Laptop" {
					return false, fmt.Sprintf("expected Name='Laptop', got %q", name)
				}
				return true, ""
			})
		},
	)

	suite.AddTest(
		"test_repeated_parameter_alias_consistent",
		"same parameter alias can be referenced multiple times in an expression",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt @p and Price lt @p&@p=100")
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
			if len(items) != 0 {
				return fmt.Errorf("expected contradictory repeated alias filter to return no items, got %d", len(items))
			}
			return nil
		},
	)

	suite.AddTest(
		"test_missing_parameter_alias_rejected",
		"unbound parameter alias returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt @missing")
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "")
		},
	)

	return suite
}
