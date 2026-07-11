package v4_01

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderIsolation validates both the 4.01 Isolation spelling and the legacy
// OData-Isolation spelling that remains necessary for 4.0 clients.
func HeaderIsolation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.6 Header Isolation",
		"Tests snapshot-isolation header compatibility for OData 4.01 and 4.0 clients.",
		"https://docs.oasis-open.org/odata/odata/v4.01/os/part1-protocol/odata-v4.01-os-part1-protocol.html#sec_HeaderIsolationODataIsolation",
	)

	for _, tc := range []struct {
		name       string
		headerName string
		maxVersion string
	}{
		{"test_isolation_snapshot_401", "Isolation", "4.01"},
		{"test_odata_isolation_snapshot_40_compatibility", "OData-Isolation", "4.0"},
	} {
		tc := tc
		suite.AddTest(
			tc.name,
			tc.headerName+":snapshot is honored or rejected with 412 Precondition Failed",
			func(ctx *framework.TestContext) error {
				resp, err := ctx.GET("/Products?$top=1",
					framework.Header{Key: tc.headerName, Value: "snapshot"},
					framework.Header{Key: "OData-MaxVersion", Value: tc.maxVersion},
				)
				if err != nil {
					return err
				}
				if resp.StatusCode != 200 && resp.StatusCode != 412 {
					return fmt.Errorf("%s:snapshot status = %d, want 200 if supported or 412 if unsupported", tc.headerName, resp.StatusCode)
				}
				return nil
			},
		)
	}

	return suite
}
