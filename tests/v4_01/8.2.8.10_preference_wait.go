package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// PreferenceWait creates the 8.2.8.10 wait preference test suite.
func PreferenceWait() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.8.10 Preference wait",
		"Validates that the OData 4.01 wait preference is accepted alongside synchronous and asynchronous requests.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_Preferencewait",
	)

	suite.AddTest(
		"test_wait_preference_is_accepted",
		"Prefer: wait=0 does not invalidate an otherwise valid OData 4.01 request",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1", framework.Header{Key: "Prefer", Value: "wait=0"}, framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
				return fmt.Errorf("Prefer: wait=0 should be accepted or ignored, got %d", resp.StatusCode)
			}
			if resp.StatusCode == http.StatusAccepted && resp.Headers.Get("Location") == "" {
				return framework.NewError("202 response for wait preference must include Location")
			}
			if resp.StatusCode == http.StatusOK {
				var payload map[string]interface{}
				if err := json.Unmarshal(resp.Body, &payload); err != nil {
					return fmt.Errorf("wait response is not valid JSON: %w", err)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_wait_with_respond_async",
		"Prefer: respond-async, wait=0 is accepted; an asynchronous response includes Location",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1", framework.Header{Key: "Prefer", Value: "respond-async, wait=0"}, framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
				return fmt.Errorf("combined async/wait preference should be accepted or ignored, got %d", resp.StatusCode)
			}
			if resp.StatusCode == http.StatusAccepted && resp.Headers.Get("Location") == "" {
				return framework.NewError("202 response for respond-async must include Location")
			}
			return nil
		},
	)

	return suite
}
