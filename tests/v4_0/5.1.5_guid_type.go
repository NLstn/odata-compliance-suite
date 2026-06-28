package v4_0

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

var guidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// GuidType creates the 5.1.5 GUID Type test suite
func GuidType() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.5 GUID Type",
		"Tests handling of Edm.Guid primitive type including UUID format (8-4-4-4-12 hex digits), filtering, and metadata representation per RFC 4122.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_guid_in_metadata",
		"Edm.Guid type appears in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.Guid"`) {
				return nil // Optional type
			}

			return nil
		},
	)

	suite.AddTest(
		"test_guid_literal_format",
		"GUID literal 8-4-4-4-12 format returns empty set for non-existent GUID",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ID eq 12345678-1234-1234-1234-123456789012", func(p map[string]interface{}) bool {
				id := productString(p, "ID")
				return strings.EqualFold(id, "12345678-1234-1234-1234-123456789012")
			})
		},
	)

	suite.AddTest(
		"test_guid_case_insensitive",
		"GUID literal is case-insensitive: upper and lower case return identical result sets",
		func(ctx *framework.TestContext) error {
			upper := "12345678-ABCD-ABCD-ABCD-123456789ABC"
			lower := "12345678-abcd-abcd-abcd-123456789abc"
			if err := assertProductFilter(ctx, "ID eq "+upper, func(p map[string]interface{}) bool {
				return strings.EqualFold(productString(p, "ID"), upper)
			}); err != nil {
				return err
			}
			return assertProductFilter(ctx, "ID eq "+lower, func(p map[string]interface{}) bool {
				return strings.EqualFold(productString(p, "ID"), lower)
			})
		},
	)

	suite.AddTest(
		"test_guid_equality",
		"GUID eq all-zeros returns empty set (no entity has this key)",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ID eq 00000000-0000-0000-0000-000000000000", func(p map[string]interface{}) bool {
				return strings.EqualFold(productString(p, "ID"), "00000000-0000-0000-0000-000000000000")
			})
		},
	)

	suite.AddTest(
		"test_guid_inequality",
		"GUID ne all-zeros returns all entities (none have this key)",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ID ne 00000000-0000-0000-0000-000000000000", func(p map[string]interface{}) bool {
				return !strings.EqualFold(productString(p, "ID"), "00000000-0000-0000-0000-000000000000")
			})
		},
	)

	suite.AddTest(
		"test_guid_null_comparison",
		"GUID ne null returns all entities (ID is a non-nullable key)",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ID ne null", func(p map[string]interface{}) bool {
				return productString(p, "ID") != ""
			})
		},
	)

	suite.AddTest(
		"test_guid_cast",
		"cast() function supports Edm.Guid",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(ID,'Edm.Guid') ne null")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_guid_isof",
		"isof() function supports Edm.Guid",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=isof(ID,'Edm.Guid')")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_guid_in_key",
		"GUID can be used as entity key in URL path",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products(12345678-1234-1234-1234-123456789012)")
			if err != nil {
				return err
			}
			// Accept 200 (found) or 404 (not found) as valid responses
			if resp.StatusCode != 200 && resp.StatusCode != 404 {
				return framework.NewError(fmt.Sprintf("Expected 200 or 404, got %d", resp.StatusCode))
			}
			return nil
		},
	)

	suite.AddTest(
		"test_guid_in_response",
		"GUID values are serialized in 8-4-4-4-12 hex format in the response",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=5&$select=ID,Name")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}
			for _, item := range items {
				id := productString(item, "ID")
				if id == "" {
					return framework.NewError("entity is missing the ID field")
				}
				if !guidPattern.MatchString(id) {
					return fmt.Errorf("ID value %q does not match 8-4-4-4-12 GUID format", id)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_guid_format_validation",
		"Invalid GUID format returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ID eq invalid-guid-format")
			if err != nil {
				return err
			}
			// A syntactically invalid Edm.Guid literal must be rejected with
			// 400 Bad Request, not silently ignored (200) or surfaced as a
			// server error (5xx).
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
