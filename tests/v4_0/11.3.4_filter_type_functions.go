package v4_0

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// FilterTypeFunctions creates the 11.3.4 Type Functions test suite.
//
// Tests verify the actual semantics of isof()/cast(): the filtered set is
// compared against an oracle computed in Go from a full fetch (the discriminator
// property ProductType identifies the derived SpecialProduct rows), not merely
// checked for HTTP 200.
func FilterTypeFunctions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.4 Type Functions in $filter",
		"Tests type checking and casting functions (isof, cast) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations",
	)

	// isof() against the derived type returns exactly the SpecialProduct rows.
	suite.AddTest("test_isof_derived_type", "isof() selects entities of a derived type",
		func(ctx *framework.TestContext) error {
			ns, err := schemaNamespace(ctx)
			if err != nil {
				return err
			}
			if ns == "" {
				return ctx.Skip("could not determine the schema namespace")
			}
			return assertProductFilter(ctx, fmt.Sprintf("isof(%s.SpecialProduct)", ns), func(p map[string]interface{}) bool {
				return productString(p, "ProductType") == "SpecialProduct"
			})
		})

	// not isof() is the complement: every non-derived row.
	suite.AddTest("test_isof_derived_type_negated", "not isof() selects entities not of a derived type",
		func(ctx *framework.TestContext) error {
			ns, err := schemaNamespace(ctx)
			if err != nil {
				return err
			}
			if ns == "" {
				return ctx.Skip("could not determine the schema namespace")
			}
			return assertProductFilter(ctx, fmt.Sprintf("not isof(%s.SpecialProduct)", ns), func(p map[string]interface{}) bool {
				return productString(p, "ProductType") != "SpecialProduct"
			})
		})

	// isof() against the property's own primitive type holds for every row.
	suite.AddTest("test_isof_primitive_type", "isof() holds for a property of the named primitive type",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "isof(Price,Edm.Double)", func(p map[string]interface{}) bool {
				_, ok := productFloat(p, "Price")
				return ok
			})
		})

	// cast() to a numeric type, compared against a value.
	suite.AddTest("test_cast_to_numeric", "cast() converts a value for numeric comparison",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "cast(Rating,Edm.Int32) eq 200", func(p map[string]interface{}) bool {
				rating, ok := productFloat(p, "Rating")
				return ok && int(rating) == 200
			})
		})

	// cast() to Edm.String, compared against a string literal. Uses Rating
	// (Edm.Byte) rather than the flags enum Status: casting an enum to Edm.String
	// yields member names ("InStock,Featured"), not the underlying number, so a
	// numeric literal would be representation-dependent. A numeric type's string
	// form is unambiguous.
	suite.AddTest("test_cast_to_string", "cast() converts a value to Edm.String",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "cast(Rating,Edm.String) eq '200'", func(p map[string]interface{}) bool {
				rating, ok := productFloat(p, "Rating")
				return ok && int(rating) == 200
			})
		})

	return suite
}
