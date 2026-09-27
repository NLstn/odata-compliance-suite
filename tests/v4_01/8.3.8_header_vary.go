package v4_01

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderVary checks the OData 4.01 rules for HTTP cache variance.
func HeaderVary() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.3.8 Header Vary",
		"Tests Vary for response differences caused by OData-MaxVersion and Prefer.",
		"https://docs.oasis-open.org/odata/odata/v4.01/os/part1-protocol/odata-v4.01-os-part1-protocol.html#sec_HeaderVary",
	)

	suite.AddTest("test_vary_contains_odata_maxversion",
		"Responses negotiated with OData-MaxVersion advertise that variance through Vary",
		func(ctx *framework.TestContext) error {
			resp40, err := ctx.GET("/Products?$top=1", framework.Header{Key: "OData-MaxVersion", Value: "4.0"})
			if err != nil {
				return err
			}
			resp401, err := ctx.GET("/Products?$top=1", framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			if err != nil {
				return err
			}
			if resp40.StatusCode != http.StatusOK || resp401.StatusCode != http.StatusOK {
				return fmt.Errorf("OData-MaxVersion negotiation should succeed (got %d and %d)", resp40.StatusCode, resp401.StatusCode)
			}
			if resp40.Headers.Get("OData-Version") != resp401.Headers.Get("OData-Version") || resp40.Headers.Get("Content-Type") != resp401.Headers.Get("Content-Type") || string(resp40.Body) != string(resp401.Body) {
				if !varyContains(resp40.Headers.Values("Vary"), "OData-MaxVersion") || !varyContains(resp401.Headers.Values("Vary"), "OData-MaxVersion") {
					return fmt.Errorf("responses vary by OData-MaxVersion but Vary is %q and %q", resp40.Headers.Get("Vary"), resp401.Headers.Get("Vary"))
				}
			}
			return nil
		},
	)

	suite.AddTest("test_vary_contains_prefer_when_representation_changes",
		"A response whose status or body presence changes under Prefer advertises Vary: Prefer",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{"Name": "Vary Preference Test", "Price": 23.45, "Status": 1}
			plain, err := ctx.POST("/Products", payload, framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			if err != nil {
				return err
			}
			preferred, err := ctx.POST("/Products", payload, framework.Header{Key: "OData-MaxVersion", Value: "4.01"}, framework.Header{Key: "Prefer", Value: "return=minimal"})
			if err != nil {
				return err
			}
			if plain.StatusCode < 200 || plain.StatusCode >= 300 || preferred.StatusCode < 200 || preferred.StatusCode >= 300 {
				return fmt.Errorf("expected successful create responses, got %d and %d", plain.StatusCode, preferred.StatusCode)
			}
			changed := plain.StatusCode != preferred.StatusCode || (len(plain.Body) == 0) != (len(preferred.Body) == 0)
			if changed && (!varyContains(plain.Headers.Values("Vary"), "Prefer") || !varyContains(preferred.Headers.Values("Vary"), "Prefer")) {
				return fmt.Errorf("Prefer changed the response (status/body %d/%d vs %d/%d; Preference-Applied %q) but Vary is %q and %q", plain.StatusCode, len(plain.Body), preferred.StatusCode, len(preferred.Body), preferred.Headers.Get("Preference-Applied"), plain.Headers.Get("Vary"), preferred.Headers.Get("Vary"))
			}
			return nil
		},
	)
	return suite
}

func varyContains(values []string, wanted string) bool {
	for _, value := range values {
		for _, token := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(token), wanted) || strings.TrimSpace(token) == "*" {
				return true
			}
		}
	}
	return false
}
