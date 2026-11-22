package v4_01

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// PreferenceOmitValues creates the 8.2.8.6 omit-values preference test suite.
func PreferenceOmitValues() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.8.6 Preference omit-values",
		"Checks the response and confirmation header when a service applies omit-values.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_Preferenceomitvalues",
	)

	for _, value := range []string{"nulls", "defaults"} {
		suite.AddTest(
			"test_omit_values_"+value,
			fmt.Sprintf("omit-values=%s preserves non-default values and confirms any omission", value),
			func(ctx *framework.TestContext) error {
				version := framework.Header{Key: "OData-MaxVersion", Value: "4.01"}
				products, err := collectEntityCollection(ctx, "/Products", version)
				if err != nil {
					return err
				}

				for _, product := range products {
					if description, present := product["Description"]; !present || description != nil {
						continue
					}
					id, ok := product["ID"].(string)
					if !ok || id == "" {
						return fmt.Errorf("seeded Product with null Description has no ID")
					}
					path := fmt.Sprintf("/Products(%s)", id)
					baseline, err := ctx.GET(path, version)
					if err != nil {
						return err
					}
					if err := ctx.AssertStatusCode(baseline, http.StatusOK); err != nil {
						return err
					}
					var original map[string]interface{}
					if err := ctx.GetJSON(baseline, &original); err != nil {
						return err
					}
					if description, present := original["Description"]; !present || description != nil {
						continue
					}

					resp, err := ctx.GET(path, version, framework.Header{Key: "Prefer", Value: "omit-values=" + value})
					if err != nil {
						return err
					}
					return assertOmitValuesResponse(ctx, resp, original, value)
				}

				return ctx.Skip("no seeded Product has an explicit null Description in its entity response")
			},
		)
	}

	return suite
}

func assertOmitValuesResponse(ctx *framework.TestContext, resp *framework.HTTPResponse, original map[string]interface{}, value string) error {
	if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
		return err
	}
	if err := ctx.AssertHeader(resp, "OData-Version", "4.01"); err != nil {
		return err
	}
	var entity map[string]interface{}
	if err := ctx.GetJSON(resp, &entity); err != nil {
		return err
	}
	for _, field := range []string{"ID", "Name"} {
		want, ok := original[field].(string)
		if !ok || want == "" || entity[field] != want {
			return fmt.Errorf("expected unchanged %s %q, got %v", field, want, entity[field])
		}
	}

	description, present := entity["Description"]
	if present && description != nil {
		return fmt.Errorf("expected Description to remain null or be omitted, got %v", description)
	}
	if !present {
		for field := range original {
			if strings.HasPrefix(field, "Description@") {
				return fmt.Errorf("omit-values removed Description despite its annotation %q", field)
			}
		}
	}

	confirmed := false
	for _, header := range resp.Headers.Values("Preference-Applied") {
		for _, pref := range strings.Split(header, ",") {
			name, parameter, _ := strings.Cut(strings.TrimSpace(pref), "=")
			if !strings.EqualFold(strings.TrimSpace(name), "omit-values") {
				continue
			}
			parameter, _, _ = strings.Cut(parameter, ";")
			parameter = strings.TrimSpace(parameter)
			if parameter != value && parameter != `"`+value+`"` {
				return fmt.Errorf("unexpected Preference-Applied token %q for omit-values=%s", pref, value)
			}
			confirmed = true
		}
	}
	if !present && !confirmed {
		return fmt.Errorf("Description was omitted without Preference-Applied: omit-values=%s", value)
	}
	return nil
}
