package v4_02

import (
	"fmt"
	"net/http"

	"github.com/nlstn/odata-compliance-suite/framework"
)

func QueryRequestBody() *framework.TestSuite {
	suite := framework.NewTestSuite("4.17 OData 4.02 query bodies", "Exercises QUERY and the 4.02 query body encodings.", urlSpec+"#sec_PassingQueryOptionsintheRequestBody")

	for _, tc := range []struct {
		name, mediaType, body string
	}{
		{"text", "text/plain", "$filter=Name%20eq%20'Laptop'&$top=1"},
		{"form", "application/x-www-form-urlencoded", "%24filter=Name+eq+%27Laptop%27&%24top=1"},
		{"json", "application/json", `{"$filter":"Name eq 'Laptop'","$top":1}`},
	} {
		for _, method := range []string{"POST", "QUERY"} {
			if method == "POST" && tc.name == "text" {
				continue // Already exercised by the 4.01 suite.
			}
			suite.AddTest("test_"+method+"_"+tc.name, method+" combines "+tc.name+" options with URL options", func(ctx *framework.TestContext) error {
				path := "/Products"
				if method == "POST" {
					path += "/$query"
				}
				path += "?$select=ID,Name"
				var resp *framework.HTTPResponse
				var err error
				if method == "QUERY" {
					resp, err = ctx.QUERYRaw(path, []byte(tc.body), tc.mediaType, payload402, max402)
				} else {
					resp, err = ctx.POSTRaw(path, []byte(tc.body), tc.mediaType, payload402, max402)
				}
				if err != nil {
					return err
				}
				if err := assertResponse(ctx, resp, http.StatusOK, "4.02"); err != nil {
					return err
				}
				items, err := ctx.ParseEntityCollection(resp)
				if err != nil {
					return err
				}
				if err := assertSeededLaptop(items); err != nil {
					return err
				}
				if _, present := items[0]["Price"]; present {
					return fmt.Errorf("body processing discarded the URL projection")
				}
				return nil
			})
		}
	}
	suite.AddTest("test_query_402_payload_401_response", "A 4.02 query payload can negotiate a 4.01 response", func(ctx *framework.TestContext) error {
		resp, err := ctx.QUERYRaw("/Products", []byte(`{"$filter":"Name eq 'Laptop'","$top":1}`), "application/json", payload402,
			framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
		if err != nil {
			return err
		}
		if err := assertResponse(ctx, resp, http.StatusOK, "4.01"); err != nil {
			return err
		}
		items, err := ctx.ParseEntityCollection(resp)
		if err != nil {
			return err
		}
		return assertSeededLaptop(items)
	})
	suite.AddTest("test_json_query_function_parameters", "A JSON query body supplies typed function parameters without path parentheses", func(ctx *framework.TestContext) error {
		for _, tc := range []struct {
			path, mediaType string
			request         func(string, []byte, string, ...framework.Header) (*framework.HTTPResponse, error)
		}{
			{"/FindProducts?$top=1", "application/json", ctx.QUERYRaw},
			{"/FindProducts/$query?$top=1", "application/json", ctx.POSTRaw},
		} {
			resp, err := tc.request(tc.path, []byte(`{"name":"Laptop","maxPrice":1000}`), tc.mediaType, payload402, max402)
			if err != nil {
				return err
			}
			if err := assertResponse(ctx, resp, http.StatusOK, "4.02"); err != nil {
				return fmt.Errorf("%s: %w", tc.path, err)
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := assertSeededLaptop(items); err != nil {
				return err
			}
		}
		return nil
	})

	suite.AddTest("test_invalid_query_body", "Rejects invalid JSON options and duplicate URL/body options", func(ctx *framework.TestContext) error {
		for _, tc := range []struct{ suffix, body string }{
			{"", `{"$top":"1"}`},
			{"", `{"top":1}`},
			{"", `{"$filter":true}`},
			{"", `{"$top":1,"$top":2}`},
			{"?$top=1", `{"$top":1}`},
		} {
			resp, err := ctx.POSTRaw("/Products/$query"+tc.suffix, []byte(tc.body), "application/json", payload402, max402)
			if err != nil {
				return err
			}
			if err := ctx.AssertODataError(resp, http.StatusBadRequest, ""); err != nil {
				return fmt.Errorf("body %s with URL suffix %q: %w", tc.body, tc.suffix, err)
			}
		}
		return nil
	})
	return suite
}
