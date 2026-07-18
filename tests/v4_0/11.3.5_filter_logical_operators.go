package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// productStatusNames maps ProductStatus member names to their integer values.
var productStatusNames = map[string]int{
	"None":         0,
	"InStock":      1,
	"OnSale":       2,
	"Discontinued": 4,
	"Featured":     8,
}

// enumStatusValue returns the integer value of the flags-enum Status field.
//
// Per OData JSON Format §7.1, an enumeration value MUST be serialized as a JSON
// string containing the member name(s) — for a flags enum, a comma-separated
// list (e.g. "InStock,Featured"). This helper strictly requires that
// representation; a numeric Status is treated as non-compliant and produces an
// error so the test fails rather than silently accepting the legacy format.
func enumStatusValue(item map[string]interface{}) (int, error) {
	val, ok := item["Status"]
	if !ok {
		return 0, fmt.Errorf("item missing Status field")
	}
	s, ok := val.(string)
	if !ok {
		return 0, fmt.Errorf("Status must be serialized as an OData enum member-name string, got %T (%v)", val, val)
	}
	result := 0
	for _, name := range strings.Split(s, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		n, ok := productStatusNames[name]
		if !ok {
			return 0, fmt.Errorf("unknown ProductStatus member name %q", name)
		}
		result |= n
	}
	return result, nil
}

// wantStatus builds an assertProductFilter predicate from an enumStatusValue
// check; products whose Status is unparseable are excluded (a parse failure
// there is a data problem, not something $filter is expected to have excluded).
func wantStatus(match func(status int) bool) func(map[string]interface{}) bool {
	return func(p map[string]interface{}) bool {
		status, err := enumStatusValue(p)
		return err == nil && match(status)
	}
}

// FilterLogicalOperators creates the 11.3.5 Logical Operators test suite
func FilterLogicalOperators() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.5 Logical Operators in $filter",
		"Tests logical operators (and, or, not) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_LogicalOperators",
	)

	// Test 1: AND operator
	suite.AddTest(
		"test_and_operator",
		"AND operator works in filter expressions",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price gt 10 and Price lt 100", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && price > 10 && price < 100
			})
		},
	)

	// Test 2: OR operator
	suite.AddTest(
		"test_or_operator",
		"OR operator works in filter expressions",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price lt 10 or Price gt 100", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && (price < 10 || price > 100)
			})
		},
	)

	// Test 3: NOT operator
	suite.AddTest(
		"test_not_operator",
		"NOT operator works in filter expressions",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "not (Price gt 50)", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && !(price > 50)
			})
		},
	)

	// Test 4: Complex expression with AND and OR
	suite.AddTest(
		"test_complex_and_or",
		"Complex expression with AND and OR",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "(Price lt 10 or Price gt 100) and Status eq 9", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				if !ok {
					return false
				}
				status, err := enumStatusValue(p)
				return err == nil && (price < 10 || price > 100) && status == 9
			})
		},
	)

	// Test 5: Multiple AND operators
	suite.AddTest(
		"test_multiple_and",
		"Multiple AND operators chain correctly",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price gt 10 and Price lt 100 and Status eq 1", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				if !ok {
					return false
				}
				status, err := enumStatusValue(p)
				return err == nil && price > 10 && price < 100 && status == 1
			})
		},
	)

	// Test 6: Multiple OR operators
	suite.AddTest(
		"test_multiple_or",
		"Multiple OR operators chain correctly",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Status eq 1 or Status eq 2 or Status eq 3", wantStatus(func(status int) bool {
				return status == 1 || status == 2 || status == 3
			}))
		},
	)

	// Test 7: NOT with AND
	suite.AddTest(
		"test_not_with_and",
		"NOT with AND expression",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "not (Price gt 50 and Status eq 1)", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				if !ok {
					return false
				}
				status, err := enumStatusValue(p)
				return err == nil && !(price > 50 && status == 1)
			})
		},
	)

	// Test 8: Parentheses for precedence
	suite.AddTest(
		"test_parentheses_precedence",
		"Parentheses control operator precedence",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Price gt 10 and (Status eq 1 or Status eq 2)", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				if !ok {
					return false
				}
				status, err := enumStatusValue(p)
				return err == nil && price > 10 && (status == 1 || status == 2)
			})
		},
	)

	// Test 9: NOT with OR
	suite.AddTest(
		"test_not_with_or",
		"NOT with OR expression",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "not (Price lt 10 or Price gt 100)", func(p map[string]interface{}) bool {
				price, ok := productFloat(p, "Price")
				return ok && !(price < 10 || price > 100)
			})
		},
	)

	suite.AddTest(
		"test_and_precedence_over_or",
		"AND binds more tightly than OR",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Name eq 'Coffee Mug' or Name eq 'Laptop' and Price gt 1000", func(p map[string]interface{}) bool {
				name := productString(p, "Name")
				price, _ := productFloat(p, "Price")
				return name == "Coffee Mug" || (name == "Laptop" && price > 1000)
			})
		},
	)

	suite.AddTest(
		"test_parentheses_override_and_or_precedence",
		"Parentheses override default AND/OR precedence",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "(Name eq 'Coffee Mug' or Name eq 'Laptop') and Price gt 1000", func(p map[string]interface{}) bool {
				name := productString(p, "Name")
				price, _ := productFloat(p, "Price")
				return (name == "Coffee Mug" || name == "Laptop") && price > 1000
			})
		},
	)

	return suite
}
