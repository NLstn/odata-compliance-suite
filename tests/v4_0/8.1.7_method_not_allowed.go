package v4_0

import (
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// MethodNotAllowed creates the 8.1.7 Method Not Allowed test suite
func MethodNotAllowed() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.1.7 Method Not Allowed (405)",
		"Tests that services properly return 405 Method Not Allowed for unsupported HTTP methods according to OData specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ResponseStatusCodes",
	)

	suite.AddTest(
		"test_put_on_collection",
		"PUT on entity collection returns 405 Method Not Allowed",
		func(ctx *framework.TestContext) error {
			// PUT is not allowed on collections, only on individual entities
			resp, err := ctx.PUT("/Products", []byte(`{}`))
			if err != nil {
				return err
			}

			// Should return 405 Method Not Allowed
			return ctx.AssertStatusCode(resp, 405)
		},
	)

	suite.AddTest(
		"test_patch_on_collection",
		"PATCH on entity collection returns 405 Method Not Allowed",
		func(ctx *framework.TestContext) error {
			// PATCH is not allowed on collections
			resp, err := ctx.PATCH("/Products", []byte(`{}`))
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 405)
		},
	)

	suite.AddTest(
		"test_post_on_metadata",
		"POST on $metadata returns 405 Method Not Allowed",
		func(ctx *framework.TestContext) error {
			// POST is not allowed on metadata document
			resp, err := ctx.POST("/$metadata", []byte(`{}`))
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 405)
		},
	)

	suite.AddTest(
		"test_put_on_metadata",
		"PUT on $metadata returns 405 Method Not Allowed",
		func(ctx *framework.TestContext) error {
			// PUT is not allowed on metadata document
			resp, err := ctx.PUT("/$metadata", []byte(`{}`))
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 405)
		},
	)

	suite.AddTest(
		"test_delete_on_metadata",
		"DELETE on $metadata returns 405 Method Not Allowed",
		func(ctx *framework.TestContext) error {
			// DELETE is not allowed on metadata document
			resp, err := ctx.DELETE("/$metadata")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 405)
		},
	)

	suite.AddTest(
		"test_post_on_service_document",
		"POST on service document returns 405 Method Not Allowed",
		func(ctx *framework.TestContext) error {
			// POST is not allowed on service document
			resp, err := ctx.POST("/", []byte(`{}`))
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 405)
		},
	)

	suite.AddTest(
		"test_put_on_service_document",
		"PUT on service document returns 405 Method Not Allowed",
		func(ctx *framework.TestContext) error {
			// PUT is not allowed on service document
			resp, err := ctx.PUT("/", []byte(`{}`))
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 405)
		},
	)

	suite.AddTest(
		"test_delete_on_service_document",
		"DELETE on service document returns 405 Method Not Allowed",
		func(ctx *framework.TestContext) error {
			// DELETE is not allowed on service document
			resp, err := ctx.DELETE("/")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 405)
		},
	)

	suite.AddTest(
		"test_405_includes_allow_header",
		"405 Method Not Allowed response includes the Allow header",
		func(ctx *framework.TestContext) error {
			// A 405 response MUST include an Allow header listing the methods the
			// resource supports (RFC 7231 §6.5.5, referenced by OData Part 1 §8.1.5).
			resp, err := ctx.PUT("/$metadata", []byte(`{}`))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 405); err != nil {
				return err
			}

			if strings.TrimSpace(resp.Headers.Get("Allow")) == "" {
				return framework.NewError("405 response is missing the required Allow header listing the permitted methods (RFC 7231 §6.5.5)")
			}
			return nil
		},
	)

	return suite
}
