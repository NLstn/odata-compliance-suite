package v4_0

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// HeaderAccept creates the 8.2.7 Accept Header test suite
func HeaderAccept() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.7 Header Accept",
		"Tests Accept header content negotiation and handling of different media types",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderAccept",
	)

	registerHeaderAcceptTests(suite)
	return suite
}

func registerHeaderAcceptTests(suite *framework.TestSuite) {
	suite.AddTest(
		"Accept application/json",
		"Accept: application/json should return JSON",
		func(ctx *framework.TestContext) error {
			// Use a valid product key for the compliance server (GUID keys)
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "application/json",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			return nil
		},
	)

	suite.AddTest(
		"Accept */* returns JSON",
		"Accept: */* should return JSON (default)",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "*/*",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			return nil
		},
	)

	suite.AddTest(
		"Unsupported Accept returns 406",
		"Unsupported entity media type returns 406 with an OData error payload",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "application/xml",
			})
			if err != nil {
				return err
			}

			return ctx.AssertODataError(resp, http.StatusNotAcceptable, "requested format")
		},
	)

	suite.AddTest(
		"Accept with odata.metadata parameter",
		"Accept with parameters should be supported",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "application/json;odata.metadata=minimal",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			return nil
		},
	)

	suite.AddTest(
		"Accept quality values respected",
		"Accept with quality values should prefer higher quality",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "text/plain;q=0.5, application/json;q=1.0",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected JSON, got %s", contentType)
			}

			return nil
		},
	)

	suite.AddTest(
		"Metadata accepts application/xml",
		"Metadata document should support application/xml",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{
				Key:   "Accept",
				Value: "application/xml",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/xml") {
				return fmt.Errorf("expected Content-Type application/xml, got %s", contentType)
			}

			return nil
		},
	)

	suite.AddTest(
		"Accept quality values JSON preferred over Atom",
		"JSON q=1.0 beats Atom q=0.8; service must return application/json",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "application/json;odata.metadata=minimal;q=1.0, application/atom+xml;q=0.8",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type application/json (higher q-value), got %s", contentType)
			}

			return nil
		},
	)

	suite.AddTest(
		"Accept quality values JSON preferred over Atom (reversed order)",
		"JSON q=1.0 beats Atom q=0.8 regardless of header order; service must return application/json",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "application/atom+xml;q=0.8, application/json;odata.metadata=minimal;q=1.0",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type application/json (higher q-value), got %s", contentType)
			}

			return nil
		},
	)

	suite.AddTest(
		"Accept quality values Atom preferred over JSON",
		"Atom q=1.0 beats JSON q=0; service must return application/atom+xml when Atom is supported",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			// Probe whether the service supports Atom at all.
			probe, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "application/atom+xml",
			})
			if err != nil {
				return err
			}
			if probe.StatusCode == http.StatusNotAcceptable {
				return ctx.Skip("service does not support application/atom+xml; skipping Atom quality-value test")
			}
			probeContentType := strings.ToLower(probe.Headers.Get("Content-Type"))
			if !strings.Contains(probeContentType, "application/atom+xml") {
				return ctx.Skip("service does not serve application/atom+xml; skipping Atom quality-value test")
			}

			// Atom has q=1.0, JSON has q=0 (effectively excluded).
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "application/json;q=0, application/atom+xml;q=1.0",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/atom+xml") {
				return fmt.Errorf("expected Content-Type application/atom+xml (higher q-value), got %s", contentType)
			}

			return nil
		},
	)
}
