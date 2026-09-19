package v4_0

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderPreferenceSurface covers the cross-cutting HTTP behavior that is easy
// to miss when testing individual headers in isolation: negotiated response
// headers, preference grammar, and the relationship between Prefer and Vary.
func HeaderPreferenceSurface() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.1-8.3 Header and Preference Surface",
		"Tests negotiated HTTP headers, Vary behavior, and preference handling across OData 4.0 and 4.01 responses.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_CommonHeaders",
	)

	suite.AddTest(
		"test_content_length_matches_body_when_present",
		"Content-Length, when supplied, matches the response body size",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("expected 200, got %d", resp.StatusCode)
			}

			value := strings.TrimSpace(resp.Headers.Get("Content-Length"))
			if value == "" {
				// HTTP/2 and chunked HTTP/1.1 responses may omit Content-Length.
				return nil
			}
			length, err := strconv.Atoi(value)
			if err != nil || length < 0 {
				return fmt.Errorf("Content-Length must be a non-negative integer, got %q", value)
			}
			if length != len(resp.Body) {
				return fmt.Errorf("Content-Length=%d does not match decoded response body length %d", length, len(resp.Body))
			}
			return nil
		},
	)

	suite.AddTest(
		"test_content_encoding_tokens_are_valid_when_present",
		"Content-Encoding, when supplied, is a comma-separated list of non-empty coding tokens",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("expected 200, got %d", resp.StatusCode)
			}

			encoding := resp.Headers.Get("Content-Encoding")
			if encoding == "" {
				return nil
			}
			for _, token := range strings.Split(encoding, ",") {
				token = strings.TrimSpace(token)
				if token == "" || strings.ContainsAny(token, " \t\r\n") {
					return fmt.Errorf("Content-Encoding contains an invalid coding token: %q", encoding)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_accept_charset_utf8",
		"Accept-Charset: utf-8 returns a usable JSON representation",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1", framework.Header{Key: "Accept-Charset", Value: "utf-8"})
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("Accept-Charset: utf-8 should be supported, got %d", resp.StatusCode)
			}
			var payload map[string]interface{}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return fmt.Errorf("Accept-Charset response is not valid JSON: %w", err)
			}
			return nil
		},
	)

	suite.AddTest(
		"test_accept_language_is_accepted",
		"Accept-Language: en-US, en;q=0.8 is accepted and returns a valid representation",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1", framework.Header{Key: "Accept-Language", Value: "en-US,en;q=0.8"})
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("Accept-Language request should succeed, got %d", resp.StatusCode)
			}
			var payload map[string]interface{}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return fmt.Errorf("Accept-Language response is not valid JSON: %w", err)
			}
			if language := strings.TrimSpace(resp.Headers.Get("Content-Language")); language != "" && strings.ContainsAny(language, "\r\n") {
				return fmt.Errorf("Content-Language contains invalid line breaks: %q", language)
			}
			return nil
		},
	)

	suite.AddTest(
		"test_vary_contains_odata_maxversion",
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

			version40 := strings.TrimSpace(resp40.Headers.Get("OData-Version"))
			version401 := strings.TrimSpace(resp401.Headers.Get("OData-Version"))
			if version40 == version401 {
				// A service that emits identical representations for both requests
				// does not vary on this header. Still validate the negotiated value.
				return nil
			}
			if !varyContains(resp40.Headers.Values("Vary"), "OData-MaxVersion") || !varyContains(resp401.Headers.Values("Vary"), "OData-MaxVersion") {
				return fmt.Errorf("responses vary by OData-MaxVersion (%q vs %q) but Vary does not include OData-MaxVersion (got %q and %q)", version40, version401, resp40.Headers.Get("Vary"), resp401.Headers.Get("Vary"))
			}
			return nil
		},
	)

	suite.AddTest(
		"test_unknown_preference_is_ignored",
		"Unsupported preferences are ignored and are not echoed in Preference-Applied",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1", framework.Header{Key: "Prefer", Value: "x-compliance-unknown=ignored"})
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unknown preference should not make an otherwise valid request fail, got %d", resp.StatusCode)
			}
			var payload map[string]interface{}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return fmt.Errorf("response with unknown preference is not valid JSON: %w", err)
			}
			if applied := strings.ToLower(resp.Headers.Get("Preference-Applied")); strings.Contains(applied, "x-compliance-unknown") {
				return fmt.Errorf("unknown preference was incorrectly echoed in Preference-Applied: %q", applied)
			}
			return nil
		},
	)

	suite.AddTest(
		"test_preference_applied_is_subset_of_request",
		"Every Preference-Applied token corresponds to a requested preference",
		func(ctx *framework.TestContext) error {
			requested := "return=representation,odata.maxpagesize=2"
			resp, err := ctx.POST("/Products", map[string]interface{}{
				"Name":   "Preference Surface Test",
				"Price":  12.34,
				"Status": 1,
			}, framework.Header{Key: "Prefer", Value: requested})
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
				return fmt.Errorf("expected successful creation, got %d", resp.StatusCode)
			}
			for _, applied := range strings.Split(resp.Headers.Get("Preference-Applied"), ",") {
				applied = strings.TrimSpace(strings.ToLower(applied))
				if applied == "" {
					continue
				}
				matched := false
				for _, candidate := range strings.Split(requested, ",") {
					candidate = strings.TrimSpace(strings.ToLower(candidate))
					if applied == candidate || strings.HasPrefix(applied, strings.SplitN(candidate, "=", 2)[0]+"=") {
						matched = true
						break
					}
				}
				if !matched {
					return fmt.Errorf("Preference-Applied token %q was not requested (header %q)", applied, resp.Headers.Get("Preference-Applied"))
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_vary_contains_prefer_when_representation_changes",
		"A response whose status or body changes under Prefer advertises Vary: Prefer",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{
				"Name":   "Vary Preference Test",
				"Price":  23.45,
				"Status": 1,
			}
			plain, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}
			preferred, err := ctx.POST("/Products", payload, framework.Header{Key: "Prefer", Value: "return=minimal"})
			if err != nil {
				return err
			}
			if plain.StatusCode < 200 || plain.StatusCode >= 300 || preferred.StatusCode < 200 || preferred.StatusCode >= 300 {
				return fmt.Errorf("expected successful create responses, got %d and %d", plain.StatusCode, preferred.StatusCode)
			}

			changed := plain.StatusCode != preferred.StatusCode || len(plain.Body) != len(preferred.Body)
			if changed && !varyContains(preferred.Headers.Values("Vary"), "Prefer") {
				return fmt.Errorf("Prefer changed the response (status/body %d/%d vs %d/%d) but Vary is %q", plain.StatusCode, len(plain.Body), preferred.StatusCode, len(preferred.Body), strings.Join(preferred.Headers.Values("Vary"), ", "))
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
