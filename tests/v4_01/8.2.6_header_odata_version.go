package v4_01

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderODataVersion creates the 8.2.6 Header OData-Version test suite for OData 4.01 negotiation.
func HeaderODataVersion() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.6 Header OData-Version",
		"Tests OData 4.01 version negotiation behavior for default responses and explicit 4.01 negotiation.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderODataVersion",
	)

	suite.AddTest(
		"test_default_response_version_401",
		"Service document returns a supported OData-Version when no maximum is supplied",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			version := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if version != "4.0" && version != "4.01" {
				return framework.NewError(fmt.Sprintf("expected OData-Version 4.0 or 4.01 by default, got %q", version))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_explicit_maxversion_401_response",
		"Service document responds with OData-Version 4.01 when OData-MaxVersion is 4.01",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/", framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			version := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if version != "4.01" {
				return framework.NewError(fmt.Sprintf("expected OData-Version 4.01, got %q", version))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_entity_collection_default_version_401",
		"Entity collection responses use a supported version when unconstrained",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			version := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if version != "4.0" && version != "4.01" {
				return framework.NewError(fmt.Sprintf("expected OData-Version 4.0 or 4.01 on default entity response, got %q", version))
			}

			return nil
		},
	)

	// Test: Entity collection response must carry exactly one OData-Version header value.
	// A server that emits duplicate case-variant fields causes HTTP recipients to combine
	// them into "4.01,4.01", which is not a valid OData-Version value.
	// OData Protocol 4.01 §8.1.5 requires a single OData-Version response header.
	suite.AddTest(
		"test_no_duplicate_odata_version_entity",
		"Entity collection response contains exactly one OData-Version header value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
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
				return framework.NewError(fmt.Sprintf("OData-Version header must appear exactly once (OData Protocol §8.1.5), got %d values: %v", len(values), values))
			}

			version := strings.TrimSpace(values[0])
			if version != "4.0" && version != "4.01" {
				return framework.NewError(fmt.Sprintf("OData-Version must be \"4.0\" or \"4.01\", got: %q", version))
			}

			return nil
		},
	)

	// Test: Error response must also carry exactly one OData-Version header value.
	// The original defect (go-odata#874) was triggered by layered response writers,
	// so error paths are especially prone to emitting duplicates.
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
				return framework.NewError(fmt.Sprintf("OData-Version header must appear exactly once in error responses (OData Protocol §8.1.5), got %d values: %v", len(values), values))
			}

			version := strings.TrimSpace(values[0])
			if version != "4.0" && version != "4.01" {
				return framework.NewError(fmt.Sprintf("OData-Version must be \"4.0\" or \"4.01\" in error responses, got: %q", version))
			}

			return nil
		},
	)

	return suite
}
