package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// BinaryType creates the 5.1.6 Binary Type test suite
func BinaryType() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.6 Binary Type",
		"Tests handling of Edm.Binary primitive type including base64 encoding with binary'' literal format, filtering, and metadata representation.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_binary_in_metadata",
		"Edm.Binary type appears in metadata as a genuine Property declaration",
		func(ctx *framework.TestContext) error {
			refs, err := propertiesDeclaredWithType(ctx, "Edm.Binary")
			if err != nil {
				return err
			}
			if len(refs) == 0 {
				return ctx.Skip("Edm.Binary is an optional primitive type not used by this model")
			}
			for _, ref := range refs {
				if ref.Property == "" {
					return framework.NewError("EntityType " + ref.EntityType + " declares an Edm.Binary property with no Name attribute")
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_binary_literal_format",
		"Binary literal binary'base64' returns entities whose Data matches",
		func(ctx *framework.TestContext) error {
			// base64url (unpadded, per OData JSON Format §7.1) of "test" is "dGVzdA".
			// Filed and fixed as NLstn/go-odata#801.
			return assertProductFilter(ctx, "Data eq binary'dGVzdA'", func(p map[string]interface{}) bool {
				return productString(p, "Data") == "dGVzdA"
			})
		},
	)

	suite.AddTest(
		"test_binary_empty_value",
		"Binary empty literal binary'' returns entities with empty Data",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Data eq binary''", func(p map[string]interface{}) bool {
				d, ok := p["Data"]
				return ok && d != nil && productString(p, "Data") == ""
			})
		},
	)

	suite.AddTest(
		"test_binary_equality",
		"Binary equality comparison returns exactly matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Data eq binary'AQID'", func(p map[string]interface{}) bool {
				return productString(p, "Data") == "AQID"
			})
		},
	)

	suite.AddTest(
		"test_binary_inequality",
		"Binary ne comparison returns entities where Data differs",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Data ne binary'AQID'", func(p map[string]interface{}) bool {
				d, ok := p["Data"]
				if !ok || d == nil {
					return false // null values excluded by OData three-valued logic
				}
				return productString(p, "Data") != "AQID"
			})
		},
	)

	suite.AddTest(
		"test_binary_null_comparison",
		"Binary ne null returns only entities with a non-null Data field",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Data ne null", func(p map[string]interface{}) bool {
				d, ok := p["Data"]
				return ok && d != nil
			})
		},
	)

	suite.AddTest(
		"test_binary_cast",
		"cast() function supports Edm.Binary",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(Data,'Edm.Binary') ne null")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_binary_isof",
		"isof() function supports Edm.Binary",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=isof(Data,'Edm.Binary')")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_binary_in_response",
		"Non-null Binary values are base64-encoded strings in the JSON response",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=10&$select=ID,Name,Data")
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
				d, ok := item["Data"]
				if !ok || d == nil {
					continue // null is allowed
				}
				s, isStr := d.(string)
				if !isStr {
					return fmt.Errorf("Data value is non-null but not a string (got %T); Edm.Binary must be base64-encoded in JSON", d)
				}
				// OData JSON Format §7.1 mandates base64url (RFC 4648 §5) without
				// padding: only A-Z, a-z, 0-9, '-', '_' — never '+', '/', or '='.
				for _, ch := range s {
					if !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
						return fmt.Errorf("Data value %q contains %q; Edm.Binary must use unpadded base64url (no '+', '/', or '=') per OData JSON Format §7.1", s, ch)
					}
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_binary_invalid_base64",
		"Invalid base64 encoding returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Data eq binary'!!!invalid!!!'")
			if err != nil {
				return err
			}
			// A syntactically invalid Edm.Binary literal must be rejected with
			// 400 Bad Request, not silently ignored (200) or surfaced as a
			// server error (5xx).
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_binary_large_value",
		"Binary filter with large base64 payload is accepted",
		func(ctx *framework.TestContext) error {
			largeBase64 := strings.Repeat("AQIDBAUG", 16)
			resp, err := ctx.GET("/Products?$filter=Data ne binary'" + largeBase64 + "'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
