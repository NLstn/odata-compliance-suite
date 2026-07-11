package v4_0

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderIsolation validates the OData 4.0 spelling and required fallback
// behavior of the snapshot-isolation request header.
func HeaderIsolation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.6 Header OData-Isolation",
		"Tests OData-Isolation:snapshot acceptance or the required 412 response when snapshot isolation is unsupported.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderODataIsolation",
	)

	suite.AddTest(
		"test_odata_isolation_snapshot",
		"OData-Isolation:snapshot is honored or rejected with 412 Precondition Failed",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1",
				framework.Header{Key: "OData-Isolation", Value: "snapshot"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.0"},
			)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 && resp.StatusCode != 412 {
				return fmt.Errorf("OData-Isolation:snapshot status = %d, want 200 if supported or 412 if unsupported", resp.StatusCode)
			}
			return nil
		},
	)

	return suite
}
