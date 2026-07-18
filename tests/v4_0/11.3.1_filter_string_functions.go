package v4_0

import (
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// FilterStringFunctions creates the 11.3.1 String Functions test suite
func FilterStringFunctions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.1 String Functions in $filter",
		"Tests string functions (contains, startswith, endswith, length, etc.) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations",
	)

	// Test 1: contains function
	suite.AddTest(
		"test_contains_function",
		"contains() function filters string values",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "contains(Name,'Laptop')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "Laptop")
			})
		},
	)

	// Test 2: startswith function
	suite.AddTest(
		"test_startswith_function",
		"startswith() function filters by prefix",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "startswith(Name,'Wireless')", func(p map[string]interface{}) bool {
				return strings.HasPrefix(productString(p, "Name"), "Wireless")
			})
		},
	)

	// Test 3: endswith function
	suite.AddTest(
		"test_endswith_function",
		"endswith() function filters by suffix",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "endswith(Name,'Mouse')", func(p map[string]interface{}) bool {
				return strings.HasSuffix(productString(p, "Name"), "Mouse")
			})
		},
	)

	// Test 4: length function
	suite.AddTest(
		"test_length_function",
		"length() function returns string length",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "length(Name) gt 10", func(p map[string]interface{}) bool {
				return len(productString(p, "Name")) > 10
			})
		},
	)

	// Test 5: indexof function
	suite.AddTest(
		"test_indexof_function",
		"indexof() function finds substring position",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "indexof(Name,'Lap') eq 0", func(p map[string]interface{}) bool {
				return strings.Index(productString(p, "Name"), "Lap") == 0
			})
		},
	)

	// Test 6: substring function
	suite.AddTest(
		"test_substring_function",
		"substring() function extracts substring",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "substring(Name,0,4) eq 'Lapt'", func(p map[string]interface{}) bool {
				name := productString(p, "Name")
				return len(name) >= 4 && name[:4] == "Lapt"
			})
		},
	)

	// Test 7: tolower function
	suite.AddTest(
		"test_tolower_function",
		"tolower() function converts to lowercase",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "tolower(Name) eq 'laptop'", func(p map[string]interface{}) bool {
				return strings.ToLower(productString(p, "Name")) == "laptop"
			})
		},
	)

	// Test 8: toupper function
	suite.AddTest(
		"test_toupper_function",
		"toupper() function converts to uppercase",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "toupper(Name) eq 'LAPTOP'", func(p map[string]interface{}) bool {
				return strings.ToUpper(productString(p, "Name")) == "LAPTOP"
			})
		},
	)

	// Test 9: trim function
	suite.AddTest(
		"test_trim_function",
		"trim() function removes whitespace",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "trim(Name) eq 'Laptop'", func(p map[string]interface{}) bool {
				return strings.TrimSpace(productString(p, "Name")) == "Laptop"
			})
		},
	)

	// Test 10: concat function
	suite.AddTest(
		"test_concat_function",
		"concat() function concatenates strings",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "concat(Name,' Test') eq 'Laptop Test'", func(p map[string]interface{}) bool {
				return productString(p, "Name")+" Test" == "Laptop Test"
			})
		},
	)

	return suite
}
