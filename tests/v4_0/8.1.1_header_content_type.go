package v4_0

import (
	"fmt"
	"mime"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderContentType creates the 8.1.1 Header Content-Type test suite
func HeaderContentType() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.1.1 Header Content-Type",
		"Tests that Content-Type is properly set according to OData v4.0. The optional odata.metadata parameter is validated when present.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderContentType",
	)

	// Helper function to get product path for each test
	// Note: Must refetch on each call because database is reseeded between tests
	getProductPath := func(ctx *framework.TestContext) (string, error) {
		return firstEntityPath(ctx, "Products")
	}

	assertJSONContentType := func(resp *framework.HTTPResponse) error {
		contentType := resp.Headers.Get("Content-Type")
		if contentType == "" {
			return framework.NewError("Content-Type header is missing")
		}
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			return fmt.Errorf("failed to parse Content-Type %q: %w", contentType, err)
		}
		if !strings.EqualFold(mediaType, "application/json") {
			return fmt.Errorf("Content-Type media type = %q, want application/json", mediaType)
		}
		if metadataValue, present := params["odata.metadata"]; present &&
			metadataValue != "minimal" && metadataValue != "full" && metadataValue != "none" {
			return fmt.Errorf("invalid odata.metadata value %q", metadataValue)
		}
		return nil
	}

	// The odata.metadata parameter is optional and defaults to minimal.
	suite.AddTest(
		"test_service_doc_content_type",
		"Service Document returns application/json and a valid optional odata.metadata parameter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}
			return assertJSONContentType(resp)
		},
	)

	// Test 2: Metadata Document should return application/xml
	suite.AddTest(
		"test_metadata_xml_content_type",
		"Metadata Document returns application/xml",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")

			if !strings.Contains(contentType, "application/xml") {
				return framework.NewError(fmt.Sprintf("Expected application/xml, got: %s", contentType))
			}

			return nil
		},
	)

	// JSON CSDL is an OData 4.01 conformance feature and is tested in v4_01.
	suite.AddTest(
		"test_entity_collection_content_type",
		"Entity Collection returns application/json and a valid optional odata.metadata parameter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			return assertJSONContentType(resp)
		},
	)

	suite.AddTest(
		"test_single_entity_content_type",
		"Single Entity returns application/json and a valid optional odata.metadata parameter",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			return assertJSONContentType(resp)
		},
	)

	return suite
}
