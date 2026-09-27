package v4_01

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"sort"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// AdditionalResourcePaths creates tests for OData 4.01 resource paths that are
// distinct from ordinary entity-set addressing.
func AdditionalResourcePaths() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.1 Additional Resource Paths",
		"Tests $all and passing query options in the request body.",
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
			if len(body.Value) != 1 {
				return fmt.Errorf("$all?$top=1 returned %d entities, want exactly one", len(body.Value))
			}
			var envelope struct {
				NextLink string `json:"@odata.nextLink"`
			}
			if err := json.Unmarshal(resp.Body, &envelope); err != nil {
				return err
			}
			if envelope.NextLink != "" {
				return fmt.Errorf("$all?$top=1 returned an unexpected nextLink after the requested entity")
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

			items, err := collectEntityPages(ctx, resp, queryBodyHeaders()...)
			if err != nil {
				return fmt.Errorf("request-body query response: %w", err)
			}
			if len(items) == 0 {
				return fmt.Errorf("request-body query returned no products matching Price gt 900")
			}
			for i, product := range items {
				if product["Name"] == nil || product["Price"] == nil {
					return fmt.Errorf("result %d does not contain both selected properties Name and Price", i)
				}
				price, ok := product["Price"].(float64)
				if !ok || price <= 900 {
					return fmt.Errorf("result %d has Price=%v, expected > 900", i, product["Price"])
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_query_request_body_orderby_top",
		"POST /Products/$query returns the two highest-priced products when $orderby and $top are in the body (§4.17)",
		func(ctx *framework.TestContext) error {
			baseline, err := collectEntityCollection(ctx, "/Products?$select=ID,Price", queryBodyHeaders()...)
			if err != nil {
				return fmt.Errorf("baseline products: %w", err)
			}
			if len(baseline) < 3 {
				return fmt.Errorf("need at least three baseline products to test $top=2, got %d", len(baseline))
			}
			for i, product := range baseline {
				if _, ok := product["Price"].(float64); !ok || product["ID"] == nil {
					return fmt.Errorf("baseline product %d is missing ID or numeric Price", i)
				}
			}
			sort.Slice(baseline, func(i, j int) bool {
				return baseline[i]["Price"].(float64) > baseline[j]["Price"].(float64)
			})
			if baseline[1]["Price"].(float64) == baseline[2]["Price"].(float64) {
				return fmt.Errorf("baseline has a price tie at the $top=2 boundary")
			}
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
			items, err := collectEntityPages(ctx, resp, queryBodyHeaders()...)
			if err != nil {
				return err
			}
			if len(items) != 2 {
				return fmt.Errorf("expected 2 products from $top=2, got %d", len(items))
			}
			for i, item := range items {
				if item["ID"] == nil || item["Price"] != baseline[i]["Price"] || item["ID"] != baseline[i]["ID"] {
					return fmt.Errorf("result %d is ID=%v Price=%v, want ID=%v Price=%v", i, item["ID"], item["Price"], baseline[i]["ID"], baseline[i]["Price"])
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_query_request_body_count",
		"POST /Products/$query returns the filtered items and inline count supplied in the body (§4.17, §11.2.5.5)",
		func(ctx *framework.TestContext) error {
			baseline, err := collectEntityCollection(ctx, "/Products?$select=ID,Price", queryBodyHeaders()...)
			if err != nil {
				return fmt.Errorf("baseline products: %w", err)
			}
			expected := map[string]bool{}
			for i, product := range baseline {
				price, ok := product["Price"].(float64)
				if !ok || product["ID"] == nil {
					return fmt.Errorf("baseline product %d is missing ID or numeric Price", i)
				}
				if price > 900 {
					expected[fmt.Sprint(product["ID"])] = true
				}
			}
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
			items, err := collectEntityPages(ctx, resp, queryBodyHeaders()...)
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
			if count != float64(len(items)) {
				return fmt.Errorf("@odata.count=%v does not match returned item count %d", count, len(items))
			}
			actual, err := entityIDs(items)
			if err != nil {
				return err
			}
			if !maps.Equal(actual, expected) {
				return fmt.Errorf("body-supplied filter returned product IDs %v, want %v", actual, expected)
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
				[]byte("$filter=Price%GGgt%20900"),
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
			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnsupportedMediaType {
				return fmt.Errorf("non-text/plain query body returned %d, want 400 or 415", resp.StatusCode)
			}
			return nil
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
			if resp.StatusCode == http.StatusBadRequest {
				return nil
			}
			if err := ctx.AssertStatusCode(resp, http.StatusMethodNotAllowed); err != nil {
				return fmt.Errorf("GET /$query: %w", err)
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
