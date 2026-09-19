package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"

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

	suite.AddTest(
		"test_query_request_body_orderby_top",
		"POST /Products/$query applies $orderby and $top supplied in the request body (§4.17)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POSTRaw(
				"/Products/$query",
				[]byte("$orderby=Price%20desc&$top=2"),
				"text/plain",
				queryBodyHeaders()...,
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return fmt.Errorf("orderby/top query failed: %w", err)
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if len(items) == 0 || len(items) > 2 {
				return fmt.Errorf("expected 1-2 products from $top=2, got %d", len(items))
			}
			for i := 1; i < len(items); i++ {
				previous, ok := items[i-1]["Price"].(float64)
				if !ok {
					return fmt.Errorf("result %d has non-numeric Price %T", i-1, items[i-1]["Price"])
				}
				current, ok := items[i]["Price"].(float64)
				if !ok {
					return fmt.Errorf("result %d has non-numeric Price %T", i, items[i]["Price"])
				}
				if previous < current {
					return fmt.Errorf("prices are not ordered descending: %v before %v", previous, current)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_query_request_body_count",
		"POST /Products/$query returns an inline count for a body-supplied filter (§4.17, §11.2.5.5)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POSTRaw(
				"/Products/$query",
				[]byte("$filter=Price%20gt%20900.0&$count=true"),
				"text/plain",
				queryBodyHeaders()...,
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return fmt.Errorf("count query failed: %w", err)
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			var payload map[string]interface{}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return fmt.Errorf("count query response is not valid JSON: %w", err)
			}
			count, ok := payload["@odata.count"].(float64)
			if !ok {
				return fmt.Errorf("count query response is missing numeric @odata.count")
			}
			if int(count) != len(items) {
				return fmt.Errorf("@odata.count=%v does not match returned item count %d", count, len(items))
			}
			return nil
		},
	)

	suite.AddTest(
		"test_query_request_body_malformed",
		"POST /Products/$query rejects malformed percent-encoded query options (§4.17)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POSTRaw(
				"/Products/$query",
				[]byte("$filter=Price%20gt"),
				"text/plain",
				queryBodyHeaders()...,
			)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	suite.AddTest(
		"test_query_request_body_content_type",
		"POST /Products/$query requires a text/plain request body (§4.17)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POSTRaw(
				"/Products/$query",
				[]byte("$top=1"),
				"application/json",
				queryBodyHeaders()...,
			)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, http.StatusUnsupportedMediaType)
		},
	)

	suite.AddTest(
		"test_query_request_body_method",
		"GET /Products/$query is rejected because query-option request bodies use POST (§4.17)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/$query", queryBodyHeaders()...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusMethodNotAllowed); err != nil {
				return err
			}
			return ctx.AssertHeaderContains(resp, "Allow", "POST")
		},
	)

	suite.AddTest(
		"test_query_request_body_whitespace",
		"POST /Products/$query rejects unencoded whitespace in the request body (§4.17)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POSTRaw(
				"/Products/$query",
				[]byte("$top=1\n"),
				"text/plain",
				queryBodyHeaders()...,
			)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	return suite
}

func queryBodyHeaders() []framework.Header {
	return []framework.Header{
		{Key: "Accept", Value: "application/json"},
		{Key: "OData-MaxVersion", Value: "4.01"},
	}
}
