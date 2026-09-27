package v4_0

import (
	"fmt"
	"mime"
	"net/http"

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

			if err := assertResponseMediaType(resp, "application/json"); err != nil {
				return err
			}
			_, params, err := mime.ParseMediaType(resp.Headers.Get("Content-Type"))
			if err != nil {
				return err
			}
			if charset, ok := params["charset"]; ok {
				return fmt.Errorf("response added charset %q although Accept did not request one", charset)
			}
			return nil
		},
	)

	suite.AddTest(
		"Accept */* returns a representation",
		"Accept: */* permits any supported response format",
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

			if _, _, err := mime.ParseMediaType(resp.Headers.Get("Content-Type")); err != nil {
				return fmt.Errorf("response has invalid Content-Type %q: %w", resp.Headers.Get("Content-Type"), err)
			}
			if len(resp.Body) == 0 {
				return fmt.Errorf("wildcard Accept returned an empty entity representation")
			}
			return nil
		},
	)

	suite.AddTest(
		"Unsupported Accept returns 406",
		"An unsupported entity media type returns 406",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "application/x-odata-compliance-unsupported",
			})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusNotAcceptable)
		},
	)

	suite.AddTest(
		"Unknown Accept format parameter is rejected",
		"A JSON format with an unknown parameter is rejected",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{Key: "Accept", Value: "application/json;odata.compliance-unknown=true"})
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, http.StatusNotAcceptable)
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

			return assertResponseMediaType(resp, "application/json")
		},
	)

	suite.AddTest(
		"Accept quality values respected",
		"An unsupported lower-quality media type does not displace JSON",
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

			return assertResponseMediaType(resp, "application/json")
		},
	)

	suite.AddTest(
		"Accept q=0 rejects JSON",
		"Accept: application/json;q=0 makes JSON unacceptable when no other format is offered",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{Key: "Accept", Value: "application/json;q=0"})
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, http.StatusNotAcceptable)
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

			return assertResponseMediaType(resp, "application/xml")
		},
	)

	suite.AddTest(
		"Accept quality values choose JSON over Atom",
		"JSON q=1.0 is selected over Atom q=0.8 when both are supported",
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
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return assertResponseMediaType(resp, "application/json")
		},
	)

	suite.AddTest(
		"Accept quality values ignore header order",
		"JSON q=1.0 is selected over Atom q=0.8 regardless of header order",
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
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return assertResponseMediaType(resp, "application/json")
		},
	)

	suite.AddTest(
		"Accept quality values choose Atom",
		"Atom q=1.0 is selected over JSON q=0 when Atom is supported",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

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
			if err := ctx.AssertStatusCode(probe, http.StatusOK); err != nil {
				return err
			}
			if err := assertResponseMediaType(probe, "application/atom+xml"); err != nil {
				return err
			}

			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "application/json;q=0, application/atom+xml;q=1.0",
			})
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return assertResponseMediaType(resp, "application/atom+xml")
		},
	)

}

func assertResponseMediaType(resp *framework.HTTPResponse, expected string) error {
	actual, _, err := mime.ParseMediaType(resp.Headers.Get("Content-Type"))
	if err != nil {
		return fmt.Errorf("invalid Content-Type %q: %w", resp.Headers.Get("Content-Type"), err)
	}
	if actual != expected {
		return fmt.Errorf("expected Content-Type %s, got %s", expected, resp.Headers.Get("Content-Type"))
	}
	return nil
}
