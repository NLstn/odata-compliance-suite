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
// headers and preference grammar.
func HeaderPreferenceSurface() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.1-8.3 Header and Preference Surface",
		"Tests negotiated HTTP headers and preference handling in OData 4.0 responses.",
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
		"test_accept_language_negotiates_or_rejects",
		"Accept-Language: en-US, en;q=0.8 returns a valid representation or 406",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1", framework.Header{Key: "Accept-Language", Value: "en-US,en;q=0.8"})
			if err != nil {
				return err
			}
			if resp.StatusCode == http.StatusNotAcceptable {
				return nil
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("Accept-Language request should return 200 or 406, got %d", resp.StatusCode)
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
					if applied == candidate || (strings.HasPrefix(candidate, "odata.maxpagesize=") && strings.HasPrefix(applied, "odata.maxpagesize=")) {
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

	return suite
}
