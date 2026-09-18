package v4_01

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// AdditionalResourcePaths creates tests for OData 4.01 resource paths that are
// distinct from ordinary entity-set addressing.
func AdditionalResourcePaths() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.1 Additional Resource Paths",
		"Tests $all, $crossjoin, and passing query options in the request body.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_AddressingAllEntitiesInaService",
	)

	suite.AddTest(
		"test_all_resource",
		"/$all returns the union of entity sets and supports collection query options (§4.16)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$all?$top=1",
				framework.Header{Key: "Accept", Value: "application/json"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return fmt.Errorf("$all request failed: %w", err)
			}

			var body struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("$all response is not valid JSON: %w", err)
			}
			if len(body.Value) == 0 {
				return fmt.Errorf("$all returned an empty collection; the reference model contains entities")
			}
			return nil
		},
	)

	suite.AddTest(
		"test_query_options_in_request_body",
		"POST /Products/$query combines URL and text/plain body query options (§4.17)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POSTRaw(
				"/Products/$query?$select=Name,Price",
				[]byte("$filter=Price%20gt%20900.0"),
				"text/plain",
				framework.Header{Key: "Accept", Value: "application/json"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return fmt.Errorf("request-body query failed: %w", err)
			}

			var body struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("request-body query response is not valid JSON: %w", err)
			}
			if len(body.Value) == 0 {
				return fmt.Errorf("request-body query returned no products matching Price gt 900")
			}
			for i, product := range body.Value {
				if product["Name"] == nil || product["Price"] == nil {
					return fmt.Errorf("result %d does not contain both selected properties Name and Price", i)
				}
				if _, present := product["Description"]; present {
					return fmt.Errorf("result %d contains Description despite URL $select=Name,Price", i)
				}
			}
			return nil
		},
	)

	return suite
}
