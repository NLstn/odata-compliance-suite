package capabilities

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// ChangeTracking validates the Capabilities.ChangeTracking annotation and
// checks that it agrees with the observable track-changes behavior.
func ChangeTracking() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities.ChangeTracking Annotation",
		"Validates that Products advertises change tracking and honors the corresponding preference.",
		"https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Capabilities.V1.html#ChangeTracking",
	)

	suite.AddTest(
		"metadata_declares_change_tracking",
		"Products declares Capabilities.ChangeTracking.Supported=true",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}
			value, found, err := capabilityBoolean(metadataXML, "Products", "ChangeTracking", "Supported")
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("Products is missing Capabilities.ChangeTracking.Supported")
			}
			if !value {
				return fmt.Errorf("Products advertises ChangeTracking.Supported=false")
			}
			return nil
		},
	)

	suite.AddTest(
		"track_changes_preference_is_honored",
		"A track-changes request is acknowledged when the capability is advertised",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "odata.track-changes"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			if !strings.Contains(strings.ToLower(resp.Headers.Get("Preference-Applied")), "odata.track-changes") {
				return fmt.Errorf("Preference-Applied does not acknowledge odata.track-changes")
			}
			return nil
		},
	)

	suite.AddTest(
		"track_changes_returns_delta_link",
		"A track-changes response includes an @odata.deltaLink",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "odata.track-changes"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			var payload map[string]interface{}
			if err := ctx.GetJSON(resp, &payload); err != nil {
				return err
			}
			if deltaLink, ok := payload["@odata.deltaLink"].(string); !ok || deltaLink == "" {
				return fmt.Errorf("track-changes response is missing @odata.deltaLink")
			}
			return nil
		},
	)

	return suite
}
