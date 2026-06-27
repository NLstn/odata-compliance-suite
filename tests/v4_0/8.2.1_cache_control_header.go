package v4_0

import (
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// CacheControlHeader creates the 8.2.1 Cache-Control Header test suite
func CacheControlHeader() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.1 Cache-Control Header",
		"Validates Cache-Control header handling for HTTP caching according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderCacheControl",
	)

	suite.AddTest(
		"test_metadata_cacheable",
		"Metadata document returns correct Content-Type and does not prohibit caching",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// $metadata MUST be served as application/xml per OData spec §11.1.2
			ct := resp.Headers.Get("Content-Type")
			if !strings.Contains(ct, "application/xml") {
				return framework.NewError("$metadata response Content-Type must be application/xml, got: " + ct)
			}

			// If Cache-Control is present, it must not prohibit caching of the
			// metadata document (which is a public, rarely-changing resource).
			cc := resp.Headers.Get("Cache-Control")
			if strings.Contains(cc, "no-store") {
				return framework.NewError("$metadata Cache-Control must not contain 'no-store'; metadata should be cacheable")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_service_doc_cacheable",
		"Service document returns correct Content-Type and does not prohibit caching",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Service document MUST be served as application/json per OData spec §11.1.1
			ct := resp.Headers.Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				return framework.NewError("Service document Content-Type must contain application/json, got: " + ct)
			}

			// If Cache-Control is present, it must not prohibit caching of the
			// service document (which is a public, stable resource).
			cc := resp.Headers.Get("Cache-Control")
			if strings.Contains(cc, "no-store") {
				return framework.NewError("Service document Cache-Control must not contain 'no-store'; service document should be cacheable")
			}

			return nil
		},
	)

	return suite
}
