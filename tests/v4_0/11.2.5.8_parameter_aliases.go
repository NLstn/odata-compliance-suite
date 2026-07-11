package v4_0

import (
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

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return fmt.Errorf("failed to parse response: %v", err)
			}

			// All 7 seed products have Price > 10 (cheapest is Coffee Mug at 15.50),
			// so at least 1 result is expected.
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}

			// Every returned product must satisfy the alias-expanded filter (Price > 10).
			return ctx.AssertAllEntitiesSatisfy(items, "Price gt @p (@p=10)", func(entity map[string]interface{}) (bool, string) {
				price, ok := entity["Price"].(float64)
				if !ok {
					// Price may have been excluded by a $select; skip the check.
					return true, ""
				}
				if price <= 10 {
					return false, fmt.Sprintf("Price=%.2f is not > 10; parameter alias @p=10 was not applied", price)
				}
				return true, ""
			})
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
