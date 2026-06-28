package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// PrimitiveDataTypes creates the 5.1.1 Primitive Data Types test suite
func PrimitiveDataTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.1 Primitive Data Types",
		"Validates handling of OData primitive data types in requests and responses including Edm.String, Edm.Int32, Edm.Decimal, etc.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_string_type",
		"Edm.String type handles text values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name eq 'Laptop'")
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

			return ctx.AssertAllEntitiesSatisfy(items, "Name eq 'Laptop' filter", func(entity map[string]interface{}) (bool, string) {
				name, _ := entity["Name"].(string)
				if name != "Laptop" {
					return false, fmt.Sprintf("entity Name=%q does not match filter 'Laptop'", name)
				}
				return true, ""
			})
		},
	)

	suite.AddTest(
		"test_int32_type",
		"Edm.Int32 type handles integer values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Status eq 1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if _, ok := result["value"]; !ok {
				return framework.NewError("Response missing value array")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_decimal_type",
		"Edm.Decimal type handles decimal values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 0")
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

			return ctx.AssertAllEntitiesSatisfy(items, "Price gt 0 filter", func(entity map[string]interface{}) (bool, string) {
				price, ok := entity["Price"].(float64)
				if !ok {
					return false, "entity missing numeric Price field"
				}
				if price <= 0 {
					return false, fmt.Sprintf("entity Price=%v does not satisfy gt 0", price)
				}
				return true, ""
			})
		},
	)

	suite.AddTest(
		"test_datetime_type",
		"Edm.DateTimeOffset type handles datetime values",
		func(ctx *framework.TestContext) error {
			// Edm.DateTimeOffset literals in a URL are written bare, NOT single-quoted
			// (single quotes would make it an Edm.String literal). See OData Part 2
			// URL Conventions §5.1.1.6 / ABNF dateTimeOffsetValue.
			resp, err := ctx.GET("/Products?$filter=CreatedAt lt 2099-12-31T23:59:59Z")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if _, err := ctx.ParseEntityCollection(resp); err != nil {
				return fmt.Errorf("DateTimeOffset filter response is not a valid collection: %w", err)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_null_value_handling",
		"Null value handling in filters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=CategoryID ne null")
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

			return ctx.AssertAllEntitiesSatisfy(items, "CategoryID ne null filter", func(entity map[string]interface{}) (bool, string) {
				if entity["CategoryID"] == nil {
					return false, "entity CategoryID is null but filter requires ne null"
				}
				return true, ""
			})
		},
	)

	suite.AddTest(
		"test_number_precision",
		"Decimal precision is maintained",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name eq 'Laptop'")
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
				return nil
			}

			if _, ok := items[0]["Price"]; !ok {
				return framework.NewError("entity missing Price field")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_special_characters",
		"Special characters in strings are handled",
		func(ctx *framework.TestContext) error {
			escapedFilter := url.QueryEscape("contains(Name,'&') or contains(Name,'/')")
			resp, err := ctx.GET("/Products?$filter=" + escapedFilter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if _, err := ctx.ParseEntityCollection(resp); err != nil {
				return fmt.Errorf("special character filter response is not a valid collection: %w", err)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_empty_string",
		"Empty string handling in filters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name ne ''")
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

			return ctx.AssertAllEntitiesSatisfy(items, "Name ne '' filter", func(entity map[string]interface{}) (bool, string) {
				name, _ := entity["Name"].(string)
				if name == "" {
					return false, "entity Name is empty but filter requires ne ''"
				}
				return true, ""
			})
		},
	)

	return suite
}
