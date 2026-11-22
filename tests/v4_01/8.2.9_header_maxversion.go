package v4_01

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderMaxVersion creates the 8.2.9 Header OData-MaxVersion test suite for OData 4.01 behavior.
func HeaderMaxVersion() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.9 Header OData-MaxVersion",
		"Tests OData 4.01 max-version negotiation for default requests and negotiation with an OData 4.02 client.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderODataMaxVersion",
	)

	suite.AddTest(
		"test_request_without_maxversion_returns_highest_supported",
		"Requests without OData-MaxVersion return a supported OData version",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			version := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if version != "4.0" && version != "4.01" && version != "4.02" {
				return framework.NewError(fmt.Sprintf("expected OData-Version 4.0, 4.01, or 4.02, got %q", version))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_402_client_negotiation",
		"A 4.02 client can negotiate with a 4.01 or 4.02 service",
		func(ctx *framework.TestContext) error {
			for _, path := range []string{"/", "/Products"} {
				for _, maxVersion := range []string{"4.0", "4.01", "4.02"} {
					resp, err := ctx.GET(path, framework.Header{Key: "OData-MaxVersion", Value: maxVersion})
					if err != nil {
						return err
					}
					if err := ctx.AssertStatusCode(resp, 200); err != nil {
						return fmt.Errorf("%s with OData-MaxVersion %s: %w", path, maxVersion, err)
					}
					version := strings.TrimSpace(resp.Headers.Get("OData-Version"))
					if version != maxVersion && !(maxVersion == "4.02" && version == "4.01") {
						return fmt.Errorf("%s with OData-MaxVersion %s returned OData-Version %q", path, maxVersion, version)
					}
				}
			}
			return nil
		},
	)

	return suite
}
