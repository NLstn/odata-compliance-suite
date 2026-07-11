package v4_0

import (
	"net/http"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// CaseSensitivity creates the 11.2.17 Case Sensitivity test suite.
// These tests negotiate an OData 4.0 response and validate canonical 4.0 syntax.
// They deliberately do not require rejection of 4.01-compatible spellings: a
// 4.01 service must continue to accept those regardless of OData-MaxVersion.
func CaseSensitivity() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.17 Case Sensitivity",
		"Tests canonical OData 4.0 system query option syntax and rejection of unknown $-prefixed options.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_SystemQueryOptions",
	)

	v4Headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.0"}}

	suite.AddTest(
		"test_correct_lowercase_filter",
		"Correct lowercase $filter works under OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 10", v4Headers...)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			return ctx.AssertHeader(resp, "OData-Version", "4.0")
		},
	)

	suite.AddTest(
		"test_correct_lowercase_top",
		"Correct lowercase $top works under OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=5", v4Headers...)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	suite.AddTest(
		"test_correct_lowercase_select",
		"Correct lowercase $select works under OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=Name", v4Headers...)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	suite.AddTest(
		"test_correct_lowercase_orderby",
		"Correct lowercase $orderby works under OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=Name", v4Headers...)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	suite.AddTest(
		"test_correct_lowercase_count",
		"Correct lowercase $count works under OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true", v4Headers...)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	suite.AddTest(
		"test_unknown_dollar_option_rejected",
		"Unknown $-prefixed options are rejected with 400 under OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$unknownOption=value", v4Headers...)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	return suite
}
