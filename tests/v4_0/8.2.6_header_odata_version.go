package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderODataVersion creates the 8.2.6 Header OData-Version test suite
func HeaderODataVersion() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.6 Header OData-Version",
		"Tests that OData-Version is present and respects OData 4.0 negotiation constraints.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderODataVersion",
	)

	// Test 1: Service should return OData-Version: 4.0 header by default
	suite.AddTest(
		"test_odata_version_header",
		"Service returns OData-Version: 4.0 header by default",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}

			odataVersion := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if odataVersion == "" {
				return framework.NewError("OData-Version header not found")
			}

			if odataVersion != "4.0" && odataVersion != "4.01" {
				return framework.NewError(fmt.Sprintf("OData-Version must be \"4.0\" or \"4.01\" (OData Protocol §8.1.5), got: %q", odataVersion))
			}

			return nil
		},
	)

	// Test 3: Metadata should respond with OData-Version: 4.0 when OData-MaxVersion: 4.0
	suite.AddTest(
		"test_metadata_maxversion_40_response",
		"Metadata document responds with OData-Version: 4.0 when OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata",
				framework.Header{Key: "OData-MaxVersion", Value: "4.0"},
			)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			odataVersion = strings.TrimSpace(odataVersion)
			if odataVersion != "4.0" {
				return framework.NewError(fmt.Sprintf("Expected OData-Version: 4.0, got: %s (spec requires response version <= OData-MaxVersion)", odataVersion))
			}

			return nil
		},
	)

	// Test 3: Service document should respond with OData-Version: 4.0 when OData-MaxVersion: 4.0
	suite.AddTest(
		"test_service_document_maxversion_40_response",
		"Service document responds with OData-Version: 4.0 when OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/",
				framework.Header{Key: "OData-MaxVersion", Value: "4.0"},
			)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			odataVersion = strings.TrimSpace(odataVersion)
			if odataVersion != "4.0" {
				return framework.NewError(fmt.Sprintf("Expected OData-Version: 4.0, got: %s (service document response must respect OData-MaxVersion: 4.0)", odataVersion))
			}

			return nil
		},
	)

	// Test 4: Service should reject a request it cannot satisfy with OData-MaxVersion: 3.0.
	// The spec mandates the service respond with the greatest supported version <=
	// OData-MaxVersion (§8.2.6); when none exists (3.0 < 4.0) it must reject, but the
	// spec does not pin a specific status. Both 406 Not Acceptable and 400 Bad Request
	// are accepted; the request must not succeed.
	suite.AddTest(
		"test_maxversion_30",
		"Service rejects OData-MaxVersion: 3.0 (406 or 400)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/",
				framework.Header{Key: "OData-MaxVersion", Value: "3.0"},
			)
			if err != nil {
				return err
			}

			if resp.StatusCode != 406 && resp.StatusCode != 400 {
				return framework.NewError(fmt.Sprintf("expected 406 or 400 for unsatisfiable OData-MaxVersion: 3.0, got %d", resp.StatusCode))
			}
			return nil
		},
	)

	// Test 5: OData-Version header should be exactly "4.0" in all responses
	suite.AddTest(
		"test_entity_collection_header",
		"Entity collection response includes OData-Version: 4.0 header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			odataVersion := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if odataVersion == "" {
				return framework.NewError("OData-Version header not found")
			}

			if odataVersion != "4.0" && odataVersion != "4.01" {
				return framework.NewError(fmt.Sprintf("OData-Version must be \"4.0\" or \"4.01\" (OData Protocol §8.1.5), got: %q", odataVersion))
			}

			return nil
		},
	)

	// Test 6: OData-Version header should be exactly "4.0" in error responses
	suite.AddTest(
		"test_error_response_header",
		"Error response includes OData-Version: 4.0 header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/",
				framework.Header{Key: "OData-MaxVersion", Value: "3.0"},
			)
			if err != nil {
				return err
			}

			odataVersion := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if odataVersion == "" {
				return framework.NewError("OData-Version header missing from error response")
			}

			if odataVersion != "4.0" && odataVersion != "4.01" {
				return framework.NewError(fmt.Sprintf("OData-Version must be \"4.0\" or \"4.01\" in error responses (OData Protocol §8.1.5), got: %q", odataVersion))
			}

			return nil
		},
	)

	// Test 7: Entity collection respects version negotiation
	suite.AddTest(
		"test_entity_collection_version_negotiation",
		"Entity collection response respects OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products",
				framework.Header{Key: "OData-MaxVersion", Value: "4.0"},
			)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			odataVersion = strings.TrimSpace(odataVersion)
			if odataVersion != "4.0" {
				return framework.NewError(fmt.Sprintf("Expected OData-Version: 4.0, got: %s", odataVersion))
			}

			return nil
		},
	)

	// Test 8: Entity collection response must carry exactly one OData-Version header value.
	// A server that emits duplicate case-variant fields (e.g. "OData-Version" and
	// "Odata-Version") causes HTTP recipients to combine them into "4.0,4.0", which is
	// not a valid OData-Version value and breaks clients such as PowerQuery.
	// OData Protocol 4.0 §8.2.6 requires a single OData-Version response header.
	suite.AddTest(
		"test_no_duplicate_odata_version_entity",
		"Entity collection response contains exactly one OData-Version header value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			var values []string
			for key, vals := range resp.Headers {
				if strings.EqualFold(key, "OData-Version") {
					values = append(values, vals...)
				}
			}

			if len(values) == 0 {
				return framework.NewError("OData-Version header not found in entity collection response")
			}
			if len(values) != 1 {
				return framework.NewError(fmt.Sprintf("OData-Version header must appear exactly once (OData Protocol §8.2.6), got %d values: %v", len(values), values))
			}

			version := strings.TrimSpace(values[0])
			if version != "4.0" && version != "4.01" {
				return framework.NewError(fmt.Sprintf("OData-Version must be \"4.0\" or \"4.01\", got: %q", version))
			}

			return nil
		},
	)

	// Test 9: Error response must also carry exactly one OData-Version header value.
	// The original defect (go-odata#874) was triggered by layered response writers
	// that each set the header independently, so error paths are especially prone to
	// emitting duplicates.
	suite.AddTest(
		"test_no_duplicate_odata_version_error",
		"Error response contains exactly one OData-Version header value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/",
				framework.Header{Key: "OData-MaxVersion", Value: "3.0"},
			)
			if err != nil {
				return err
			}

			var values []string
			for key, vals := range resp.Headers {
				if strings.EqualFold(key, "OData-Version") {
					values = append(values, vals...)
				}
			}

			if len(values) == 0 {
				return framework.NewError("OData-Version header not found in error response")
			}
			if len(values) != 1 {
				return framework.NewError(fmt.Sprintf("OData-Version header must appear exactly once in error responses (OData Protocol §8.2.6), got %d values: %v", len(values), values))
			}

			version := strings.TrimSpace(values[0])
			if version != "4.0" && version != "4.01" {
				return framework.NewError(fmt.Sprintf("OData-Version must be \"4.0\" or \"4.01\" in error responses, got: %q", version))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_request_payload_supported_versions",
		"Create payloads declare OData-Version 4.0 and 4.01",
		func(ctx *framework.TestContext) error {
			namespace, err := schemaNamespace(ctx)
			if err != nil {
				return err
			}
			if namespace == "" {
				return framework.NewError("could not determine Product type namespace")
			}
			for _, version := range []string{"4.0", "4.01"} {
				name := "Payload Version " + version
				payload, err := buildProductPayload(ctx, name, 12.34)
				if err != nil {
					return err
				}
				if version == "4.0" {
					payload["@odata.type"] = "#" + namespace + ".Product"
				} else {
					payload["@type"] = namespace + ".Product"
				}
				resp, err := ctx.POST("/Products", payload,
					framework.Header{Key: "OData-Version", Value: version})
				if err != nil {
					return err
				}
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					return fmt.Errorf("OData-Version %q create status = %d, want 2xx", version, resp.StatusCode)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_request_payload_invalid_versions",
		"Malformed and unsupported payload versions return 4xx without mutation",
		func(ctx *framework.TestContext) error {
			for _, version := range []string{"4", "4.1", "4.00", "4.0, 4.01", "4.02", "5.0"} {
				name := "Rejected Payload Version " + version
				payload, err := buildProductPayload(ctx, name, 12.34)
				if err != nil {
					return err
				}
				resp, err := ctx.POST("/Products", payload,
					framework.Header{Key: "OData-Version", Value: version})
				if err != nil {
					return err
				}
				if resp.StatusCode < 400 || resp.StatusCode >= 500 {
					return fmt.Errorf("OData-Version %q create status = %d, want 4xx", version, resp.StatusCode)
				}
				if err := assertNoProductNamed(ctx, name); err != nil {
					return fmt.Errorf("OData-Version %q rejected request mutated data: %w", version, err)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_request_payload_repeated_versions",
		"Repeated and conflicting OData-Version fields return 4xx without mutation",
		func(ctx *framework.TestContext) error {
			cases := []struct {
				name     string
				versions []string
			}{
				{name: "Repeated Payload Version", versions: []string{"4.0", "4.0"}},
				{name: "Conflicting Payload Version", versions: []string{"4.0", "4.01"}},
			}
			for _, test := range cases {
				payload, err := buildProductPayload(ctx, test.name, 12.34)
				if err != nil {
					return err
				}
				resp, err := ctx.POST("/Products", payload,
					framework.Header{Key: "OData-Version", Value: test.versions[0]},
					framework.Header{Key: "OData-Version", Value: test.versions[1]})
				if err != nil {
					return err
				}
				if resp.StatusCode < 400 || resp.StatusCode >= 500 {
					return fmt.Errorf("OData-Version fields %v create status = %d, want 4xx", test.versions, resp.StatusCode)
				}
				if err := assertNoProductNamed(ctx, test.name); err != nil {
					return fmt.Errorf("OData-Version fields %v rejected request mutated data: %w", test.versions, err)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_bodyless_request_ignores_payload_version",
		"A malformed OData-Version does not cause payload-version failure on a bodyless request",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1",
				framework.Header{Key: "OData-Version", Value: "4.1"})
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_request_and_response_versions_independent",
		"A 4.01 payload with OData-MaxVersion 4.0 succeeds and receives a 4.0 response",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Independent Payload Version", 12.34)
			if err != nil {
				return err
			}
			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "OData-Version", Value: "4.01"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.0"})
			if err != nil {
				return err
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("create status = %d, want 2xx", resp.StatusCode)
			}
			if version := strings.TrimSpace(resp.Headers.Get("OData-Version")); version != "4.0" {
				return fmt.Errorf("response OData-Version = %q, want 4.0", version)
			}
			return nil
		},
	)

	suite.AddTest(
		"test_invalid_payload_version_rejected_before_async",
		"An invalid payload version with respond-async is rejected immediately without mutation",
		func(ctx *framework.TestContext) error {
			const name = "Rejected Async Payload Version"
			payload, err := buildProductPayload(ctx, name, 12.34)
			if err != nil {
				return err
			}
			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "OData-Version", Value: "5.0"},
				framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}
			if resp.StatusCode < 400 || resp.StatusCode >= 500 {
				return fmt.Errorf("invalid async payload-version status = %d, want 4xx", resp.StatusCode)
			}
			if location := resp.Headers.Get("Location"); location != "" {
				return fmt.Errorf("invalid request created async monitor resource %q", location)
			}
			return assertNoProductNamed(ctx, name)
		},
	)

	return suite
}
