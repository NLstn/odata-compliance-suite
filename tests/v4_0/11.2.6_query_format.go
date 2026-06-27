package v4_0

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// QueryFormat creates the 11.2.6 Query Option $format test suite
func QueryFormat() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.6 System Query Option $format",
		"Tests $format query option for specifying response format according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionformat",
	)

	// Test 1: $format=json returns JSON
	suite.AddTest(
		"test_format_json",
		"$format=json returns JSON response",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$format=json")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type to contain 'application/json', got: %s", contentType)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			return nil
		},
	)

	// Test 2: $format=xml returns XML (for metadata)
	suite.AddTest(
		"test_format_xml",
		"$format=xml on metadata returns XML",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata?$format=xml")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/xml") {
				return fmt.Errorf("expected Content-Type to contain 'application/xml', got: %s", contentType)
			}

			return nil
		},
	)

	// Test 3: Invalid $format returns error or is ignored
	suite.AddTest(
		"test_format_invalid",
		"Invalid $format value returns error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$format=invalid")
			if err != nil {
				return err
			}

			if resp.StatusCode != 406 && resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400 or 406 for invalid $format, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type to contain 'application/json', got: %s", contentType)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("response is not valid JSON: %v", err)
			}

			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("error response must have 'error' object property")
			}

			code, ok := errorObj["code"].(string)
			if !ok || code == "" {
				return fmt.Errorf("error object must have 'code' property as non-empty string")
			}

			message, ok := errorObj["message"].(string)
			if !ok || message == "" {
				return fmt.Errorf("error object must have 'message' property as non-empty string")
			}

			return nil
		},
	)

	// Test 4: $format=atom returns Atom/XML for collection.
	// The Atom format is OPTIONAL: per OData v4.0 Part 1 §3.1 a service MUST
	// support JSON and MAY support additional formats. A JSON-only service is
	// fully conformant and must reject an unsupported $format with 415 (or 406).
	// We therefore accept either a valid Atom feed or a clean rejection — but
	// NOT a 200 JSON response (that would mean the service ignored $format).
	suite.AddTest(
		"test_format_atom_collection",
		"$format=atom returns Atom/XML for entity collection, or rejects cleanly if unsupported",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$format=atom")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return assertFormatUnsupported(resp)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/atom+xml") {
				return fmt.Errorf("service returned 200 for $format=atom but Content-Type is %q; a service that does not support Atom must reject with 415/406 rather than return another format", contentType)
			}

			// Validate well-formed XML
			if err := xml.Unmarshal(resp.Body, new(interface{})); err != nil {
				return fmt.Errorf("response is not valid XML: %v", err)
			}

			// Validate Atom feed structure: must contain <feed element
			bodyStr := string(resp.Body)
			if !strings.Contains(bodyStr, "<feed") {
				return fmt.Errorf("Atom feed response must contain <feed> element")
			}

			return nil
		},
	)

	// Test 5: $format=atom returns Atom/XML for single entity
	suite.AddTest(
		"test_format_atom_entity",
		"$format=atom returns Atom/XML for single entity",
		func(ctx *framework.TestContext) error {
			// First get a product ID from the collection
			listResp, err := ctx.GET("/Products?$format=json&$top=1")
			if err != nil {
				return err
			}
			if listResp.StatusCode != 200 {
				return fmt.Errorf("failed to list products: status %d", listResp.StatusCode)
			}
			var listResult map[string]interface{}
			if err := json.Unmarshal(listResp.Body, &listResult); err != nil {
				return fmt.Errorf("failed to parse product list: %v", err)
			}
			values, ok := listResult["value"].([]interface{})
			if !ok || len(values) == 0 {
				// No products to test with; skip this test
				return nil
			}

			firstProduct, ok := values[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("unexpected product format")
			}

			// Build the entity URL using the first available numeric ID field
			var productID interface{}
			for _, key := range []string{"ID", "Id", "ProductID", "ProductId"} {
				if v, exists := firstProduct[key]; exists {
					productID = v
					break
				}
			}
			if productID == nil {
				// Can't determine key; skip
				return nil
			}

			entityURL := fmt.Sprintf("/Products(%v)?$format=atom", productID)
			resp, err := ctx.GET(entityURL)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return assertFormatUnsupported(resp)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/atom+xml") {
				return fmt.Errorf("service returned 200 for $format=atom but Content-Type is %q; a service that does not support Atom must reject with 415/406 rather than return another format", contentType)
			}

			// Validate well-formed XML
			if err := xml.Unmarshal(resp.Body, new(interface{})); err != nil {
				return fmt.Errorf("response is not valid XML: %v", err)
			}

			// Validate Atom entry structure
			bodyStr := string(resp.Body)
			if !strings.Contains(bodyStr, "<entry") {
				return fmt.Errorf("Atom entry response must contain <entry> element")
			}

			return nil
		},
	)

	// Test 6: Accept: application/atom+xml returns Atom/XML, or 406 if unsupported.
	// Atom is optional (see Test 4); a JSON-only service must answer an
	// Accept: application/atom+xml request with 406 Not Acceptable rather than
	// silently returning JSON.
	suite.AddTest(
		"test_accept_atom_xml",
		"Accept: application/atom+xml returns Atom/XML, or 406 if Atom is unsupported",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GETWithHeaders("/Products", map[string]string{
				"Accept": "application/atom+xml",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				if resp.StatusCode == 406 {
					return nil
				}
				return fmt.Errorf("expected 200 (Atom) or 406 (Atom unsupported), got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/atom+xml") {
				return fmt.Errorf("service returned 200 for Accept: application/atom+xml but Content-Type is %q; a service that does not support Atom must respond 406 rather than return another format", contentType)
			}

			// Validate well-formed XML
			if err := xml.Unmarshal(resp.Body, new(interface{})); err != nil {
				return fmt.Errorf("response is not valid XML: %v", err)
			}

			return nil
		},
	)

	return suite
}

// assertFormatUnsupported validates that a non-200 response to an unsupported
// $format value is a clean rejection. Per OData v4.0 Part 1 §11.2.10.2, an
// unsupported $format yields 415 Unsupported Media Type; 406 Not Acceptable is
// also accepted as a defensible alternative.
func assertFormatUnsupported(resp *framework.HTTPResponse) error {
	if resp.StatusCode == 415 || resp.StatusCode == 406 {
		return nil
	}
	return fmt.Errorf("expected 200 (format supported) or 415/406 (format unsupported), got %d", resp.StatusCode)
}
