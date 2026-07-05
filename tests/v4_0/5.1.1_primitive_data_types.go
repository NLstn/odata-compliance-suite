package v4_0

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"time"

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
		"Edm.String type handles text values: every returned product has Name='Laptop'",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name eq 'Laptop'")
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

			if _, hasValue := result["value"]; !hasValue {
				return framework.NewError("Response missing value field")
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return framework.NewError("filter Name eq 'Laptop' returned no products")
			}
			for i, p := range items {
				name, _ := p["Name"].(string)
				if name != "Laptop" {
					return fmt.Errorf("entity %d has Name=%q but filter was Name eq 'Laptop'", i, name)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_int32_type",
		"Edm.Int32 type handles integer values: Status gt 0 returns products with non-None status",
		func(ctx *framework.TestContext) error {
			// Status is a flags enum serialized as "InStock", "InStock,Featured", etc.
			// Use gt 0 to test integer comparison without depending on the specific enum string "InStock".
			return assertProductFilter(ctx, "Status gt 0",
				func(p map[string]interface{}) bool {
					status, _ := p["Status"].(string)
					// Empty string or "None" means Status == 0; anything else means Status > 0.
					return status != "" && status != "None"
				})
		},
	)

	suite.AddTest(
		"test_decimal_type",
		"Edm.Decimal type handles decimal values: every returned product has Price=999.99",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price eq 999.99",
				func(p map[string]interface{}) bool {
					price, ok := productFloat(p, "Price")
					return ok && math.Abs(price-999.99) <= 0.001
				})
		},
	)

	suite.AddTest(
		"test_datetime_type",
		"Edm.DateTimeOffset type handles datetime values: returned products have parseable CreatedAt before 2099",
		func(ctx *framework.TestContext) error {
			// Edm.DateTimeOffset literals in a URL are written bare, NOT single-quoted.
			resp, err := ctx.GET("/Products?$filter=CreatedAt lt 2099-12-31T23:59:59Z")
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
			threshold, _ := time.Parse(time.RFC3339, "2099-12-31T23:59:59Z")
			for i, p := range items {
				rawCreated, ok := p["CreatedAt"].(string)
				if !ok {
					return fmt.Errorf("entity %d CreatedAt is not a string (%T)", i, p["CreatedAt"])
				}
				t, err := time.Parse(time.RFC3339, rawCreated)
				if err != nil {
					// Try without nanoseconds
					t, err = time.Parse("2006-01-02T15:04:05.999999999Z07:00", rawCreated)
					if err != nil {
						return fmt.Errorf("entity %d CreatedAt=%q not parseable as RFC3339: %v", i, rawCreated, err)
					}
				}
				if !t.Before(threshold) {
					return fmt.Errorf("entity %d CreatedAt=%q violates filter CreatedAt lt 2099", i, rawCreated)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_null_value_handling",
		"Null value handling in filters: every returned product has non-null CategoryID",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "CategoryID ne null",
				func(p map[string]interface{}) bool {
					cat, hasKey := p["CategoryID"]
					return hasKey && cat != nil
				})
		},
	)

	suite.AddTest(
		"test_number_precision",
		"Decimal precision is maintained: Laptop's Price is returned with at least 2 decimal places of precision",
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
				return framework.NewError("filter Name eq 'Laptop' returned no products")
			}
			price, ok := productFloat(items[0], "Price")
			if !ok {
				return framework.NewError("Laptop product missing Price field")
			}
			// Verify price retains sub-penny precision: 999.99 != 1000.
			if math.Abs(price-999.99) > 0.001 {
				return fmt.Errorf("Laptop Price=%v, expected ~999.99 (decimal precision not maintained)", price)
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

			return nil
		},
	)

	suite.AddTest(
		"test_empty_string",
		"Empty string handling in filters: every returned product has a non-empty Name",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Name ne ''",
				func(p map[string]interface{}) bool {
					name, _ := p["Name"].(string)
					return name != ""
				})
		},
	)

	return suite
}
