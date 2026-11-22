package v4_02

import (
	"fmt"
	"net/http"

	"github.com/nlstn/odata-compliance-suite/framework"
)

const (
	protocolSpec = "https://docs.oasis-open.org/odata/odata/v4.02/cs01/part1-protocol/odata-v4.02-cs01-part1-protocol.html"
	urlSpec      = "https://docs.oasis-open.org/odata/odata/v4.02/cs01/part2-url-conventions/odata-v4.02-cs01-part2-url-conventions.html"
	jsonSpec     = "https://docs.oasis-open.org/odata/odata-json-format/v4.02/cs01/odata-json-format-v4.02-cs01.html"
)

var max402 = framework.Header{Key: "OData-MaxVersion", Value: "4.02"}
var payload402 = framework.Header{Key: "OData-Version", Value: "4.02"}

func assertResponse(ctx *framework.TestContext, resp *framework.HTTPResponse, status int, version string) error {
	if err := ctx.AssertStatusCode(resp, status); err != nil {
		return err
	}
	return ctx.AssertHeader(resp, "OData-Version", version)
}

func readCollection(ctx *framework.TestContext, path string, headers ...framework.Header) ([]map[string]interface{}, error) {
	resp, err := ctx.GET(path, headers...)
	if err != nil {
		return nil, err
	}
	if err := assertResponse(ctx, resp, http.StatusOK, "4.02"); err != nil {
		return nil, err
	}
	return ctx.ParseEntityCollection(resp)
}

func assertSeededLaptop(items []map[string]interface{}) error {
	if len(items) != 1 || items[0]["Name"] != "Laptop" {
		return fmt.Errorf("expected only the seeded Laptop, got %v", items)
	}
	if id, ok := items[0]["ID"].(string); !ok || id == "" {
		return fmt.Errorf("seeded Laptop is missing its ID")
	}
	return nil
}
