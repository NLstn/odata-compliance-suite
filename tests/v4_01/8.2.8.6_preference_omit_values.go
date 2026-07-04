package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// PreferenceOmitValues creates the 8.2.8.6 omit-values preference test suite.
func PreferenceOmitValues() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.8.6 Preference omit-values",
		"Validates OData 4.01 omit-values preference behavior and compatibility with the OData 4.0 prefixed form.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_Preferenceomitvalues",
	)

	suite.AddTest(
		"test_omit_values_unprefixed_is_accepted",
		"Prefer: omit-values=nulls is accepted on data requests",
		func(ctx *framework.TestContext) error {
			// First fetch the same products WITHOUT the preference to discover which ones
			// have null Description values.
			baseResp, err := ctx.GET("/Products?$top=3&$select=ID,Name,Description")
			if err != nil {
				return err
			}
			if baseResp.StatusCode < http.StatusOK || baseResp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("baseline request failed with %d", baseResp.StatusCode))
			}
			var basePayload struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(baseResp.Body, &basePayload); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse baseline response: %v", err))
			}

			// Determine which IDs have a null Description in normal mode.
			nullDescriptionIDs := map[interface{}]struct{}{}
			for _, entity := range basePayload.Value {
				if desc, hasDesc := entity["Description"]; !hasDesc || desc == nil {
					nullDescriptionIDs[entity["ID"]] = struct{}{}
				}
			}

			// Now make the omit-values=nulls request.
			resp, err := ctx.GET("/Products?$top=3&$select=ID,Name,Description", framework.Header{Key: "Prefer", Value: "omit-values=nulls"})
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx for omit-values request, got %d", resp.StatusCode))
			}

			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return framework.NewError(fmt.Sprintf("response must remain a valid collection payload: %v", err))
			}

			// If the preference was honoured and any products had null Description,
			// those entities must NOT contain a "Description" key at all.
			if len(nullDescriptionIDs) > 0 {
				var omitPayload struct {
					Value []map[string]interface{} `json:"value"`
				}
				if err := json.Unmarshal(resp.Body, &omitPayload); err != nil {
					return framework.NewError(fmt.Sprintf("failed to parse omit-values response: %v", err))
				}
				for i, entity := range omitPayload.Value {
					if _, isNullDesc := nullDescriptionIDs[entity["ID"]]; isNullDesc {
						if _, hasDesc := entity["Description"]; hasDesc {
							return framework.NewError(fmt.Sprintf("entity %d (ID=%v) has null Description but 'Description' key is still present with omit-values=nulls", i, entity["ID"]))
						}
					}
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_omit_values_prefixed_alias_is_accepted",
		"Prefer: odata.omit-values=nulls is accepted for 4.0 compatibility",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3", framework.Header{Key: "Prefer", Value: "odata.omit-values=nulls"})
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx for odata.omit-values request, got %d", resp.StatusCode))
			}

			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return framework.NewError(fmt.Sprintf("response must remain a valid collection payload: %v", err))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_omit_values_preference_applied_is_consistent",
		"If Preference-Applied is returned, it echoes the applied omit-values form",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3", framework.Header{Key: "Prefer", Value: "omit-values=defaults"})
			if err != nil {
				return err
			}

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx for omit-values=defaults request, got %d", resp.StatusCode))
			}

			applied := resp.Headers.Get("Preference-Applied")
			if applied == "" {
				return nil
			}

			if !strings.Contains(applied, "omit-values=defaults") && !strings.Contains(applied, "odata.omit-values=defaults") {
				return framework.NewError(fmt.Sprintf("unexpected Preference-Applied value for omit-values: %q", applied))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_omit_values_version_negotiation_4_01_vs_4_0",
		"unprefixed omit-values preference applies in 4.01 negotiation and is not reported as applied in 4.0 negotiation",
		func(ctx *framework.TestContext) error {
			v401Headers := []framework.Header{
				{Key: "OData-MaxVersion", Value: "4.01"},
				{Key: "Prefer", Value: "omit-values=nulls"},
			}
			v401Resp, err := ctx.GET("/Products?$top=3", v401Headers...)
			if err != nil {
				return err
			}
			if v401Resp.StatusCode < http.StatusOK || v401Resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx for 4.01 negotiated omit-values request, got %d", v401Resp.StatusCode))
			}

			v40Headers := []framework.Header{
				{Key: "OData-MaxVersion", Value: "4.0"},
				{Key: "Prefer", Value: "omit-values=nulls"},
			}
			v40Resp, err := ctx.GET("/Products?$top=3", v40Headers...)
			if err != nil {
				return err
			}
			if v40Resp.StatusCode < http.StatusOK || v40Resp.StatusCode >= 300 {
				return framework.NewError(fmt.Sprintf("expected 2xx for 4.0 negotiated request with unknown preference token, got %d", v40Resp.StatusCode))
			}

			applied := v40Resp.Headers.Get("Preference-Applied")
			if strings.Contains(applied, "omit-values=nulls") {
				return framework.NewError(fmt.Sprintf("4.0 negotiated response must not report unprefixed omit-values as applied, got %q", applied))
			}

			return nil
		},
	)

	return suite
}
