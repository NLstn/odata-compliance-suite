package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// InOperator creates the 11.2.5.1 $filter in-operator test suite.
func InOperator() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.1 $filter In Operator",
		"Validates the OData 4.01 in operator in $filter expressions, including successful membership filtering and required 400 responses for invalid expressions.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_in",
	)

	suite.AddTest(
		"test_filter_in_string_membership",
		"String membership: Name in ('Laptop','Wireless Mouse') returns only matching entities",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name in ('Laptop','Wireless Mouse')")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := decodeCollectionAllowEmpty(resp)
			if err != nil {
				return err
			}

			allowed := map[string]struct{}{"Laptop": {}, "Wireless Mouse": {}}
			for i, entity := range entities {
				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing string Name field", i))
				}
				if _, exists := allowed[name]; !exists {
					return framework.NewError(fmt.Sprintf("entity %d has Name=%q not in expected set", i, name))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_filter_in_numeric_membership",
		"Numeric membership: Price in (29.99, 15.50) returns only matching entities",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price in (29.99,15.50)")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := decodeCollectionAllowEmpty(resp)
			if err != nil {
				return err
			}

			allowedPrices := map[float64]struct{}{29.99: {}, 15.50: {}}
			for i, entity := range entities {
				price, err := floatField(entity, "Price")
				if err != nil {
					return framework.NewError(fmt.Sprintf("entity %d: %v", i, err))
				}
				if _, exists := allowedPrices[price]; !exists {
					return framework.NewError(fmt.Sprintf("entity %d has Price=%f not in expected set", i, price))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_filter_in_with_combined_expression",
		"Combined expression: Name in ('Laptop','Wireless Mouse') and Price gt 20 enforces both predicates",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name in ('Laptop','Wireless Mouse') and Price gt 20")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := decodeCollectionAllowEmpty(resp)
			if err != nil {
				return err
			}

			allowedNames := map[string]struct{}{"Laptop": {}, "Wireless Mouse": {}}
			for i, entity := range entities {
				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing string Name field", i))
				}
				if _, exists := allowedNames[name]; !exists {
					return framework.NewError(fmt.Sprintf("entity %d has Name=%q not in expected set", i, name))
				}

				price, err := floatField(entity, "Price")
				if err != nil {
					return framework.NewError(fmt.Sprintf("entity %d: %v", i, err))
				}
				if price <= 20 {
					return framework.NewError(fmt.Sprintf("entity %d has Price=%v, expected > 20", i, price))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_filter_in_type_mismatch_string_in_numeric",
		"Type mismatch: string property compared to numeric list returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name in (1,2)")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	suite.AddTest(
		"test_filter_in_type_mismatch_numeric_in_string",
		"Type mismatch: numeric property compared to string list returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price in ('expensive','cheap')")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	suite.AddTest(
		"test_filter_in_malformed_missing_parentheses",
		"Malformed list syntax (missing parentheses) returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name in 'Laptop','Wireless Mouse'")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	suite.AddTest(
		"test_filter_in_malformed_missing_comma",
		"Malformed list syntax (missing comma) returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name in ('Laptop' 'Wireless Mouse')")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	suite.AddTest(
		"test_filter_in_empty_expression",
		"Empty expression errors return 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	suite.AddTest(
		"test_filter_in_empty_list_rejected",
		"Empty in-list syntax returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name in ()")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	suite.AddTest(
		"test_filter_in_duplicate_members",
		"Duplicate members in an in-list do not duplicate result entities",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name in ('Laptop','Laptop')")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := decodeCollectionAllowEmpty(resp)
			if err != nil {
				return err
			}
			if len(entities) != 1 {
				return framework.NewError(fmt.Sprintf("expected duplicate in-list to return Laptop once, got %d entities", len(entities)))
			}
			name, ok := entities[0]["Name"].(string)
			if !ok || name != "Laptop" {
				return framework.NewError(fmt.Sprintf("expected only Laptop, got Name=%v", entities[0]["Name"]))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_filter_not_in_expression",
		"not applied to an in-expression excludes listed members",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=not (Name in ('Laptop','Wireless Mouse'))")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := decodeCollectionAllowEmpty(resp)
			if err != nil {
				return err
			}
			for i, entity := range entities {
				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing string Name field", i))
				}
				if name == "Laptop" || name == "Wireless Mouse" {
					return framework.NewError(fmt.Sprintf("entity %d has excluded Name=%q", i, name))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_filter_in_null_member",
		"Null members in an in-list match null property values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Description in (null)")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := decodeCollectionAllowEmpty(resp)
			if err != nil {
				return err
			}
			for i, entity := range entities {
				if value, ok := entity["Description"]; ok && value != nil {
					return framework.NewError(fmt.Sprintf("entity %d has non-null Description=%v", i, value))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_filter_in_version_negotiation_4_01_vs_4_0",
		"in-operator is accepted with OData-MaxVersion 4.01 and rejected when negotiated to 4.0",
		func(ctx *framework.TestContext) error {
			v401Headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			v401Resp, err := ctx.GET("/Products?$filter=Name in ('Laptop','Wireless Mouse')", v401Headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(v401Resp, http.StatusOK); err != nil {
				return framework.NewError(fmt.Sprintf("4.01 negotiated in-operator request should succeed: %v", err))
			}

			v40Headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.0"}}
			v40Resp, err := ctx.GET("/Products?$filter=Name in ('Laptop','Wireless Mouse')", v40Headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(v40Resp, http.StatusBadRequest); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 negotiated request must reject 4.01 in-operator syntax: %v", err))
			}
			if err := ctx.AssertODataError(v40Resp, http.StatusBadRequest, "not supported in OData 4.0"); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 negotiated in-operator rejection must include strict OData error payload: %v", err))
			}

			return nil
		},
	)

	return suite
}

func decodeCollectionAllowEmpty(resp *framework.HTTPResponse) ([]map[string]interface{}, error) {
	var payload struct {
		Value []map[string]interface{} `json:"value"`
	}

	if err := decodeJSON(resp, &payload); err != nil {
		return nil, err
	}

	if payload.Value == nil {
		return nil, framework.NewError("response body does not contain 'value' collection")
	}

	return payload.Value, nil
}

func decodeJSON(resp *framework.HTTPResponse, target interface{}) error {
	if err := json.Unmarshal(resp.Body, target); err != nil {
		return framework.NewError(fmt.Sprintf("failed to parse JSON response: %v", err))
	}
	return nil
}

func floatField(entity map[string]interface{}, key string) (float64, error) {
	v, ok := entity[key]
	if !ok {
		return 0, fmt.Errorf("missing %q field", key)
	}

	n, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("field %q is %T, expected number", key, v)
	}

	return n, nil
}
